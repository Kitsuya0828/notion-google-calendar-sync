package main

import (
	"log"
	"os"
	"time"

	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/task"
	"github.com/robfig/cron/v3"
	"golang.org/x/exp/slog"
)

func main() {
	opt := &slog.HandlerOptions{
		// AddSource: true,
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opt))
	slog.SetDefault(logger)

	c := cron.New()
	c.AddFunc("@every 3m", func() {
		log.Println("cron job started")
		err := task.Exec()
		if err != nil {
			log.Fatal(err)
		}
	})
	c.Start()
	time.Sleep(10 * time.Minute)
}
