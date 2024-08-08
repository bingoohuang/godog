package godog

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
)

func Tick(ctx context.Context, interval, jitter time.Duration, f func() error) error {
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

func GetEnvSize(name string, defaultValue uint64) uint64 {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := humanize.ParseBytes(env)
	if err != nil {
		log.Fatalf("parse env %s error: %v", name, err)
	}
	return val
}

func GetEnvInt(name string, defaultValue uint64) uint64 {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := strconv.ParseUint(env, 10, 64)
	if err != nil {
		log.Fatalf("parse env %s error: %v", name, err)
	}
	return val
}

func GetEnvDuration(name string, defaultValue time.Duration) time.Duration {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := time.ParseDuration(env)
	if err != nil {
		log.Fatalf("parse env %s error: %v", name, err)
	}
	return val
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
