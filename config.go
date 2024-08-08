package godog

import (
	"os"
	"runtime"
	"time"
)

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

const (
	DefaultInterval     = time.Minute
	DefaultTimes        = 5
	DefaultRSSThreshold = 256 * 1024 * 1024 // 256 M
	DefaultJitter       = 10 * time.Second
)

var DefaultCPUThreshold = uint64(50 * runtime.NumCPU())

func createConfig(options []ConfigFn) *Config {
	c := &Config{
		RSSThreshold:        DefaultRSSThreshold,
		CPUPercentThreshold: DefaultCPUThreshold,
		Interval:            DefaultInterval,
		Times:               DefaultTimes,
		Jitter:              DefaultJitter,
	}
	for _, option := range options {
		option(c)
	}

	if c.Pid <= 0 {
		c.Pid = os.Getpid()
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
	return c
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
