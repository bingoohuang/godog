package godog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

type Profile interface {
	ProfileName() string
	Close() error
}

// CreateMemProfile 创建内存性能分析文件
func CreateMemProfile(dir string, pid int) (Profile, error) {
	name := filepath.Join(dir, fmt.Sprintf("Dog.mem.%d.prof", pid))
	f, err := os.Create(name)
	if err != nil {
		return nil, fmt.Errorf("create profile file %s: %w", name, err)
	}
	defer f.Close()

	// 进行内存性能分析并写入文件
	runtime.GC() // 触发 GC，获取最新的内存分配信息
	if err := pprof.WriteHeapProfile(f); err != nil {
		return nil, fmt.Errorf("write heap profile: %w", err)
	}

	return &profile{Name: name}, nil
}

// CreateCPUProfile 创建 CPU 性能分析文件
func CreateCPUProfile(dir string, pid int) (Profile, error) {
	name := filepath.Join(dir, fmt.Sprintf("Dog.cpu.%d.prof", pid))
	f, err := os.Create(name)
	if err != nil {
		return nil, fmt.Errorf("create profile file %s: %w", name, err)
	}

	// 启动 CPU 性能分析
	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, fmt.Errorf("start CPU profile: %w", err)
	}

	return &profile{
		Name: name,
		File: f,
	}, nil
}

type profile struct {
	Name string
	File *os.File
}

func (c *profile) ProfileName() string { return c.Name }

func (c *profile) Close() error {
	if c.File != nil {
		pprof.StopCPUProfile()
		if err := c.File.Close(); err != nil {
			return fmt.Errorf("close CPU profile: %w", err)
		}
	}

	return nil
}

func (c *profile) RemoveFile() error {
	if err := os.Remove(c.Name); err != nil {
		return fmt.Errorf("remove profile file %s: %w", c.Name, err)
	}

	return nil
}

type noopProfile struct{}

func (n noopProfile) ProfileName() string { return "" }
func (n noopProfile) Close() error        { return nil }
func (n noopProfile) RemoveFile() error   { return nil }
