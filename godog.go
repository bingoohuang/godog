package godog

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v4/process"
)

type Dog struct {
	*Config

	RSSState *thresholdState
	CPUState *thresholdState
}

const (
	DefaultInterval     = time.Minute
	DefaultTimes        = 5
	DefaultRSSThreshold = 256 * 1024 * 1024 // 256 M
	DefaultJitter       = 10 * time.Second
)

var DefaultCPUThreshold = 66 * runtime.NumCPU()

func New(options ...ConfigFn) *Dog {
	c := &Config{
		RSSThreshold:        DefaultRSSThreshold,
		CPUPercentThreshold: float64(DefaultCPUThreshold),
		Interval:            DefaultInterval,
		Times:               DefaultTimes, // 连续5次
		Jitter:              DefaultJitter,
	}
	for _, option := range options {
		option(c)
	}

	if c.Pid <= 0 {
		c.Pid = os.Getpid() // 获取当前进程的PID
	}
	if c.Interval <= 0 {
		c.Interval = DefaultInterval
	}
	if c.Times == 0 {
		c.Times = DefaultTimes
	}
	if c.Action == nil {
		c.Action = ActionFn(DefaultAction)
	}

	return &Dog{
		Config: c,

		RSSState: newThresholdState("RSS"),
		CPUState: newThresholdState("CPU"),
	}
}

type ExitFile struct {
	Time    string       `json:"time"`
	Reasons []ReasonItem `json:"reasons,omitempty"`
}

var DefaultAction = func(dir string, reasons []ReasonItem) {
	log.Printf("program exit by godog, reason: %v", reasons)

	data, _ := json.Marshal(ExitFile{
		Time:    time.Now().Format(time.RFC3339),
		Reasons: reasons,
	})

	name := filepath.Join(dir, "Dog.exit")
	_ = os.WriteFile(name, data, os.ModePerm)
	os.Exit(1)
}

type State struct {
	RSS        uint64
	VMS        uint64
	CPUPercent float64
}

type Action interface {
	DoAction(dir string, reasons []ReasonItem)
}

type ActionFn func(dir string, reasons []ReasonItem)

func (f ActionFn) DoAction(dir string, reasons []ReasonItem) {
	f(dir, reasons)
}

func (w *Dog) Watch(ctx context.Context) error {
	pid := w.Pid

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return fmt.Errorf("get process %d: %w", pid, err)
	}

	return tick(ctx, w.Interval, w.Jitter, func() error {
		w.stat(p)
		if reasons, yes := w.reachTimes(); yes {
			if w.Debug {
				log.Printf("godo reach times: %v", reasons)
			}

			w.Action.DoAction(w.Dir, reasons)
		}

		return nil
	})

}

func tick(ctx context.Context, interval, jitter time.Duration, f func() error) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for ctx.Err() == nil {
		if jitter > 0 {
			RandomSleep(ctx, jitter)
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if err := f(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(interval)
		}
	}

	return ctx.Err()
}

func (w *Dog) stat(p *process.Process) {

	var debugMessage string
	if w.RSSThreshold > 0 {
		// 获取内存信息
		memInfo, err1 := p.MemoryInfo()
		if err1 == nil {
			rss := memInfo.RSS // 常驻集大小，即实际使用的物理内存

			if w.RSSThreshold > 0 {
				w.RSSState.setReached(rss > w.RSSThreshold, rss)
			}
			if w.Debug {
				debugMessage += fmt.Sprintf("current RSS: %s", humanize.IBytes(rss))
			}
		} else if w.Debug {
			log.Printf("E! get memory %d error: %v", p.Pid, err1)
		}
	}

	if w.CPUPercentThreshold > 0 {
		// 获取CPU使用情况
		cpuPercent, err := p.CPUPercent()
		if err == nil {
			w.CPUState.setReached(cpuPercent > w.CPUPercentThreshold, cpuPercent)
			if w.Debug {
				if debugMessage != "" {
					debugMessage += ", "
				}
				debugMessage += fmt.Sprintf("CPU: %f", cpuPercent)
			}
		} else if w.Debug {
			log.Printf("E! get cpu percent %d error: %v", p.Pid, err)
		}
	}

	if debugMessage != "" {
		log.Println(debugMessage)
	}
}

type ReasonItem struct {
	Reason    string `json:"reason"`
	Values    []any  `json:"values"`
	Threshold any    `json:"threshold"`
}

func (w *Dog) reachTimes() (reasons []ReasonItem, reached bool) {
	if values, yes := w.RSSState.reached(w.Times, w.Debug); yes {
		reasons = append(reasons, ReasonItem{
			Reason:    fmt.Sprintf("RSS 连续 %d 次超标", w.Times),
			Values:    values,
			Threshold: w.RSSThreshold,
		})
		reached = true
	}
	if values, yes := w.CPUState.reached(w.Times, w.Debug); yes {
		reasons = append(reasons, ReasonItem{
			Reason:    fmt.Sprintf("CPU 连续 %d 次超标", w.Times),
			Values:    values,
			Threshold: w.CPUPercentThreshold,
		})

		reached = true
	}

	return reasons, reached
}

type Config struct {
	Pid int

	// RSSThreshold RSS 上限
	RSSThreshold uint64

	// CPUPercentThreshold 上限
	CPUPercentThreshold float64
	// Interval 检查间隔
	Interval time.Duration
	// Jitter 间隔时间附加随机抖动
	Jitter time.Duration
	// Times 连续多少次
	Times int
	// Action 采取的动作
	Action Action
	// Debug 调试模式
	Debug bool

	// Dir 检查 Dog.busy 和生成 Dog.exit 的路径
	Dir string
}

type ConfigFn func(c *Config)

func WithConfig(nc *Config) ConfigFn {
	return func(c *Config) {
		*c = *nc
	}
}

func WithPid(pid int) ConfigFn {
	return func(c *Config) {
		c.Pid = pid
	}
}

func WithRSSThreshold(threshold uint64) ConfigFn {
	return func(c *Config) {
		c.RSSThreshold = threshold
	}
}

func WithCPUPercentThreshold(threshold float64) ConfigFn {
	return func(c *Config) {
		c.CPUPercentThreshold = threshold
	}
}

func WithInterval(interval, jitter time.Duration) ConfigFn {
	return func(c *Config) {
		c.Interval = interval
		c.Jitter = jitter
	}
}

func WithTimes(times int) ConfigFn {
	return func(c *Config) {
		c.Times = times
	}
}

type thresholdState struct {
	Name   string
	Values []any
}

func newThresholdState(name string) *thresholdState {
	return &thresholdState{
		Name: name,
	}
}

func (t *thresholdState) reached(maxTimes int, debug bool) (values []any, reached bool) {
	if debug && len(t.Values) > 0 {
		log.Printf("current %s thresholdState: %v", t.Name, t.Values)
	}

	reached = len(t.Values) >= maxTimes
	if reached {
		values = t.Values
		t.Values = make([]any, 0)
	}

	return values, reached
}

func (t *thresholdState) setReached(reached bool, value any) {
	if reached {
		t.Values = append(t.Values, value)
	} else if len(t.Values) > 0 {
		t.Values = t.Values[:0]
	}
}

// RandomSleep will sleep for a random amount of time up to max.
// If the shutdown channel is closed, it will return before it has finished
// sleeping.
func RandomSleep(ctx context.Context, max time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var ns time.Duration
	maxSleep := big.NewInt(max.Nanoseconds())
	if j, err := rand.Int(rand.Reader, maxSleep); err == nil {
		ns = time.Duration(j.Int64())
	}

	select {
	case <-ctx.Done():
	case <-time.After(ns):
	}
}
