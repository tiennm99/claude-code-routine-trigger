package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	httpClientTimeout = 30 * time.Second
	httpFireTimeout   = 30 * time.Second
	shutdownTimeout   = 35 * time.Second
)

func main() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	fire := &FireClient{
		URL:      cfg.FireURL,
		Token:    cfg.Token,
		HTTP:     &http.Client{Timeout: httpClientTimeout},
		Template: cfg.Template,
		TZ:       cfg.Location,
		Log:      logger,
	}

	sched, err := NewScheduler(cfg, fire, logger)
	if err != nil {
		logger.Error("scheduler init failed", "err", err.Error())
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sched.Start()
	logger.Info("started",
		"schedules", cfg.Schedules,
		"tz", cfg.Location.String(),
		"entries", sched.EntryCount(),
	)

	<-ctx.Done()
	logger.Info("shutting down")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer stopCancel()
	sched.Stop(stopCtx)
	logger.Info("stopped cleanly")
}
