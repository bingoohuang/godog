package busy

import (
	"context"
	"runtime"
	"time"
)

// ControlCPULoad run CPU load in specify cores count and percentage
//
//	coresCount: 使用核数
//	percentage: 每核 CPU 百分比 (默认 100), 0 时不开启 CPU 耗用
//	lockOsThread: 是否在 CPU 耗用时锁定 OS 线程
func ControlCPULoad(ctx context.Context, coresCount, percentage int, lockOsThread bool) {
	runtime.GOMAXPROCS(coresCount)

	// second     ,s  * 1
	// millisecond,ms * 1000
	// microsecond,μs * 1000 * 1000
	// nanosecond ,ns * 1000 * 1000 * 1000

	// every loop : run + sleep = 1 unit

	// 1 unit = 100 ms may be the best
	const unitHundredOfMs = 1000
	runMs := unitHundredOfMs * percentage
	sleepMs := unitHundredOfMs*100 - runMs
	runDuration := time.Duration(runMs) * time.Microsecond
	sleepDuration := time.Duration(sleepMs) * time.Microsecond
	for i := 0; i < coresCount; i++ {
		go func() {
			if lockOsThread {
				// https://github.com/golang/go/wiki/LockOSThread
				// Some libraries—especially graphical frameworks and libraries like Cocoa, OpenGL, and libSDL—use thread-local state and can require functions to be called only
				// from a specific OS thread, typically the 'main' thread. Go provides the runtime.LockOSThread function for this, but it's notoriously difficult to use correctly.
				// https://stackoverflow.com/a/25362395
				// With the Go threading model, calls to C code, assembler code, or blocking system calls occur in the same thread as the calling Go code, which is managed by the Go runtime scheduler.
				// The os.LockOSThread() mechanism is mostly useful when Go has to interface with some foreign library (a C library for instance). It guarantees that several successive calls to this library will be done in the same thread.
				// This is interesting in several situations:
				//   1. a number of graphic libraries (OS X Cocoa, OpenGL, SDL, ...) require all the calls to be done on a specific thread (or the main thread in some cases).
				//   2. some foreign libraries are based on thread local storage (TLS) facilities. They store some context in a data structure attached to the thread. Or some functions of the API provide results whose memory lifecycle is attached to the thread. This concept is used in both Windows and Unix-like systems. A typical example is the errno global variable commonly used in C libraries to store error codes. On systems supporting multi-threading, errno is generally defined as a thread-local variable.
				//   3. more generally, some foreign libraries may use a thread identifier to index/manage internal resources.
				//   4. doing any sort of linux namespace switch (e.g. unsharing a network or process namespace) is also bound to a thread, so if you don't lock the OS thread before you might get part of your code randomly scheduled into a different network/process namespace.
				runtime.LockOSThread()
				// runtime.UnlockOSThread()
			}
			for ctx.Err() == nil {
				begin := time.Now()
				for { // run 100%
					if time.Since(begin) > runDuration {
						break
					}
				}
				time.Sleep(sleepDuration)
			}
		}()
	}
}
