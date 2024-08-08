package autoload

import (
	"context"
	"log"
	"os"

	"github.com/bingoohuang/godog"
	"github.com/bingoohuang/godog/busy"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	c := &godog.Config{
		Pid:                 os.Getpid(),
		Dir:                 os.Getenv("DOG_DIR"),
		Debug:               os.Getenv("DOG_DEBUG") == "1",
		RSSThreshold:        godog.GetEnvSize("DOG_RSS", godog.DefaultRSSThreshold),
		CPUPercentThreshold: godog.GetEnvInt("DOG_CPU", uint64(godog.DefaultCPUThreshold)),
		Interval:            godog.GetEnvDuration("DOG_INTERVAL", godog.DefaultInterval),
		Jitter:              godog.GetEnvDuration("DOG_JITTER", godog.DefaultJitter),
		Times:               int(godog.GetEnvInt("DOG_TIMES", godog.DefaultTimes)),
	}

	ctx := context.Background()
	dog := godog.New(godog.WithConfig(c))
	go func() {
		if err := dog.Watch(ctx); err != nil && c.Debug {
			log.Printf("watch error: %v", err)
		}
	}()

	bi := godog.GetEnvDuration("DOG_BUSY_INTERVAL", busy.DefaultCheckBusyInterval)
	go busy.Watch(ctx, c.Dir, c.Debug, bi)
}
