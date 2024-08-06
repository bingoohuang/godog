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
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v4/process"
)

type Dog struct {
	*Config

	States []*thresholdState
}

const (
	DefaultInterval     = time.Minute
	DefaultTimes        = 5
	DefaultRSSThreshold = 256 * 1024 * 1024 // 256 M
	DefaultJitter       = 10 * time.Second
)

var DefaultCPUThreshold = uint64(66 * runtime.NumCPU())

func New(options ...ConfigFn) *Dog {
	c := &Config{
		RSSThreshold:        DefaultRSSThreshold,
		CPUPercentThreshold: DefaultCPUThreshold,
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

	d := &Dog{
		Config: c,
	}

	if c.RSSThreshold > 0 {
		d.States = append(d.States, newThresholdState("RSS", c.RSSThreshold, d.statRSS))
	}
	if c.CPUPercentThreshold > 0 {
		d.States = append(d.States, newThresholdState("CPU", c.CPUPercentThreshold, d.statCPU))
	}

	return d
}

type ExitFile struct {
	Pid     int          `json:"pid"`
	Time    string       `json:"time"`
	Reasons []ReasonItem `json:"reasons"`
}

var DefaultAction = func(dir string, debug bool, reasons []ReasonItem) {
	log.Printf("program exit by godog, reason: %v", reasons)

	data, _ := json.Marshal(ExitFile{
		Pid:     os.Getgid(),
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
	DoAction(dir string, debug bool, reasons []ReasonItem)
}

type ActionFn func(dir string, debug bool, reasons []ReasonItem)

func (f ActionFn) DoAction(dir string, debug bool, reasons []ReasonItem) {
	f(dir, debug, reasons)
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

			w.Action.DoAction(w.Dir, w.Debug, reasons)
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

type statFn func(p *process.Process, state *thresholdState) (debugMessage string)

func (w *Dog) stat(p *process.Process) {
	var debugMessages []string
	for _, state := range w.States {
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
		state.setReached(rss > w.RSSThreshold, rss)

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
		state.setReached(cpuPercent > float64(state.Threshold), cpuPercent)
		if w.Debug {
			debugMessage = fmt.Sprintf("CPU: %f", cpuPercent)
		}
	} else if w.Debug {
		log.Printf("E! get cpu percent %d error: %v", p.Pid, err)
	}

	return
}

type ReasonItem struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Values    []any  `json:"values"`
	Threshold any    `json:"threshold"`
}

func (w *Dog) reachTimes() (reasons []ReasonItem, reached bool) {
	for _, state := range w.States {
		if values, yes := state.reached(w.Times, w.Debug); yes {
			reasons = append(reasons, ReasonItem{
				Type:      state.Type,
				Reason:    fmt.Sprintf("连续 %d 次超标", w.Times),
				Values:    values,
				Threshold: state.Threshold,
			})
			reached = true
		}
	}

	return reasons, reached
}

type Config struct {
	Pid int

	// RSSThreshold RSS 上限
	RSSThreshold uint64

	// CPUPercentThreshold 上限
	CPUPercentThreshold uint64
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

func WithCPUPercentThreshold(threshold uint64) ConfigFn {
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

type stater interface {
	Stat(p *process.Process)
}

type thresholdState struct {
	Type      string
	Threshold uint64
	Values    []any

	statFn
}

func (t *thresholdState) Stat(p *process.Process) {

}

func newThresholdState(typ string, threshold uint64, fn statFn) *thresholdState {
	return &thresholdState{
		Type:      typ,
		Threshold: threshold,

		statFn: fn,
	}
}

func (t *thresholdState) reached(maxTimes int, debug bool) (values []any, reached bool) {
	if debug && len(t.Values) > 0 {
		log.Printf("current %s thresholdState: %v", t.Type, t.Values)
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
