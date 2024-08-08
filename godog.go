package godog

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v4/process"
)

type Dog struct {
	*Config

	states []*thresholdState
}

func New(options ...ConfigFn) *Dog {
	d := &Dog{
		Config: createConfig(options),
	}

	if d.RSSThreshold > 0 {
		d.states = append(d.states, newThresholdState(RSS, d.RSSThreshold, d.statRSS, d.Dir, d.Pid))
	}
	if d.CPUPercentThreshold > 0 {
		d.states = append(d.states, newThresholdState(CPU, d.CPUPercentThreshold, d.statCPU, d.Dir, d.Pid))
	}

	return d
}

type State struct {
	RSS        uint64
	VMS        uint64
	CPUPercent float64
}

func (w *Dog) Watch(ctx context.Context) error {
	pid := w.Pid

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return fmt.Errorf("get process %d: %w", pid, err)
	}

	return Tick(ctx, w.Interval, w.Jitter, func() error {
		w.stat(p)
		if reasons, yes := w.reachTimes(); yes {
			if w.Debug {
				log.Printf("godo reach times: %v", reasons)
			}

			w.Action.DoAction(w.Dir, w.Debug, reasons)
		}

		return nil
	})

}

type statFn func(p *process.Process, state *thresholdState) (debugMessage string)

func (w *Dog) stat(p *process.Process) {
	var debugMessages []string
	for _, state := range w.states {
		if msg := state.statFn(p, state); msg != "" {
			debugMessages = append(debugMessages, msg)
		}
	}

	if len(debugMessages) > 0 {
		log.Printf("%s", strings.Join(debugMessages, ", "))
	}
}

func (w *Dog) statRSS(p *process.Process, state *thresholdState) (debugMessage string) {
	// 获取内存信息
	if memInfo, err := p.MemoryInfo(); err == nil {
		rss := memInfo.RSS // 常驻集大小，即实际使用的物理内存
		state.setReached(w.Debug, rss)

		if w.Debug {
			debugMessage = fmt.Sprintf("current RSS: %s", humanize.IBytes(rss))
		}
	} else if w.Debug {
		log.Printf("E! get memory %d error: %v", p.Pid, err)
	}

	return
}

func (w *Dog) statCPU(p *process.Process, state *thresholdState) (debugMessage string) {
	// 获取CPU使用情况
	if cpuPercent, err := p.CPUPercent(); err == nil {
		state.setReached(w.Debug, uint64(cpuPercent))
		if w.Debug {
			debugMessage = fmt.Sprintf("CPU: %f", cpuPercent)
		}
	} else if w.Debug {
		log.Printf("E! get cpu percent %d error: %v", p.Pid, err)
	}

	return
}

type ReasonItem struct {
	Type      ThresholdType `json:"type"`
	Reason    string        `json:"reason"`
	Values    []uint64      `json:"values"`
	Threshold any           `json:"threshold"`
	Profile   string        `json:"profile"`
}

func (w *Dog) reachTimes() (reasons []ReasonItem, reached bool) {
	for _, state := range w.states {
		if r := state.reached(w.Times, w.Debug); r.Reached {
			reasons = append(reasons, ReasonItem{
				Type:      state.Type,
				Reason:    fmt.Sprintf("连续 %d 次超标", w.Times),
				Values:    r.Values,
				Threshold: state.Threshold,
				Profile:   r.Profile,
			})
			reached = true
		}
	}

	return reasons, reached
}

type ThresholdType string

const (
	RSS ThresholdType = "RSS"
	CPU ThresholdType = "CPU"
)

type thresholdState struct {
	Type      ThresholdType
	Threshold uint64
	Values    []uint64

	statFn
	profile Profile
	Dir     string
	Pid     int
}

func newThresholdState(typ ThresholdType, threshold uint64, fn statFn, dir string, pid int) *thresholdState {
	return &thresholdState{
		Type:      typ,
		Threshold: threshold,
		statFn:    fn,
		Dir:       dir,
		Pid:       pid,
	}
}

type reachResult struct {
	Profile string
	Values  []uint64
	Reached bool
}

func (t *thresholdState) reached(maxTimes int, debug bool) (r reachResult) {
	if debug && len(t.Values) > 0 {
		log.Printf("current %s thresholdState: %v", t.Type, t.Values)
	}

	if r.Reached = len(t.Values) >= maxTimes; r.Reached {
		r.Values = t.Values
		t.Values = nil

		switch t.Type {
		case RSS:
			if p, err := CreateMemProfile(t.Dir, t.Pid); err != nil {
				if debug {
					log.Printf("E! create mem profile error: %v", err)
				}
			} else {
				r.Profile = p.ProfileName()
			}
		default:
			if t.profile != nil {
				if err := t.profile.Close(); err != nil && debug {
					log.Printf("E! close profile error: %v", err)
				}
				r.Profile = t.profile.ProfileName()
			}
		}

		t.profile = nil
	}

	return
}

func (t *thresholdState) setReached(debug bool, value uint64) {
	if reached := value > t.Threshold; reached {
		if t.profile == nil {
			switch t.Type {
			case CPU:
				var err error
				t.profile, err = CreateCPUProfile(t.Dir, t.Pid)
				if err != nil {
					if debug {
						log.Printf("E! create cpu profile error: %v", err)
					}
					t.profile = &noopProfile{}
				}
			default:
				t.profile = &noopProfile{}
			}
		}
		t.Values = append(t.Values, value)
	} else {
		if t.profile != nil {
			if err := t.profile.Close(); err != nil && debug {
				log.Printf("E! profile close error: %v", err)
			}
		}
		if len(t.Values) > 0 {
			t.Values = t.Values[:0]
		}
	}
}
