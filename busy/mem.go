package busy

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func ControlMem(ctx context.Context, totalMem uint64) error {
	pid := os.Getpid()
	p, err := process.NewProcessWithContext(ctx, int32(pid))
	if err != nil {
		return fmt.Errorf("get process %d: %w", pid, err)
	}

	for ctx.Err() == nil {
		info, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			return fmt.Errorf("get process %d memory info: %w", pid, err)
		}
		if info.RSS >= totalMem {
			return nil
		}

		lastRss := info.RSS
		for info.RSS-lastRss < 10*1024*1024 {
			mem = append(mem, make([]byte, 1024*1024))
			info, err := p.MemoryInfoWithContext(ctx)
			if err != nil {
				return fmt.Errorf("get process %d memory info: %w", pid, err)
			}
			if info.RSS >= totalMem {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			break
		case <-time.After(time.Second):
		}
	}

	return ctx.Err()
}

func ClearMem() {
	mem = nil
	runtime.GC()
}

var mem [][]byte
