package autoload

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bingoohuang/godog"
	"github.com/bingoohuang/godog/busy"
	"github.com/dustin/go-humanize"
	_ "github.com/joho/godotenv/autoload"
)

func getEnvSize(name string, defaultValue uint64) uint64 {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := humanize.ParseBytes(env)
	if err != nil {
		log.Fatalf("failed to parse %s env var: %s", name, err)
	}
	return val
}
func getEnvInt(name string, defaultValue uint64) uint64 {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := strconv.ParseUint(env, 10, 64)
	if err != nil {
		log.Fatalf("failed to parse %s env var: %s", name, err)
	}
	return val
}

func getEnvDuration(name string, defaultValue time.Duration) time.Duration {
	env := os.Getenv(name)
	if env == "" {
		return defaultValue
	}

	val, err := time.ParseDuration(env)
	if err != nil {
		log.Fatalf("failed to parse %s env var: %s", name, err)
	}
	return val
}

func init() {
	c := &godog.Config{
		Pid:                 os.Getpid(),
		Dir:                 os.Getenv("DOG_DIR"),
		Debug:               os.Getenv("DOG_DEBUG") == "1",
		RSSThreshold:        getEnvSize("DOG_RSS", godog.DefaultRSSThreshold),
		CPUPercentThreshold: getEnvInt("DOG_CPU", uint64(godog.DefaultCPUThreshold)),
		Interval:            getEnvDuration("DOG_INTERVAL", godog.DefaultInterval),
		Jitter:              getEnvDuration("DOG_JITTER", godog.DefaultJitter),
		Times:               int(getEnvInt("DOG_TIMES", godog.DefaultTimes)),
	}

	ctx := context.Background()
	dog := godog.New(godog.WithConfig(c))
	go func() {
		if err := dog.Watch(ctx); err != nil && c.Debug {
			log.Printf("watch error: %v", err)
		}
	}()

	bi := getEnvDuration("DOG_BUSY_INTERVAL", busy.DefaultCheckBusyInterval)
	go busy.Watch(ctx, c.Dir, c.Debug, bi)
}
