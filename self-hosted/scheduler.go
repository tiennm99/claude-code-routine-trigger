package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Scheduler wires cron schedules to FireClient invocations.
type Scheduler struct {
	cron *cron.Cron
	fire *FireClient
	log  *slog.Logger
}

// NewScheduler builds a Scheduler with all schedules registered. Returns an
// error if any registration fails (defensive — config.go already validates
// expressions, but the cron library may have stricter rules at AddFunc time).
func NewScheduler(cfg *Config, fire *FireClient, log *slog.Logger) (*Scheduler, error) {
	c := cron.New(cron.WithLocation(cfg.Location))
	s := &Scheduler{cron: c, fire: fire, log: log}

	for _, expr := range cfg.Schedules {
		expr := expr
		_, err := c.AddFunc(expr, func() {
			ctx, cancel := context.WithTimeout(context.Background(), httpFireTimeout)
			defer cancel()
			if err := fire.Fire(ctx, expr); err != nil {
				log.Error("fire returned error", "cron", expr, "err", err.Error())
			}
		})
		if err != nil {
			return nil, fmt.Errorf("register cron %q: %w", expr, err)
		}
	}
	return s, nil
}

// Start the underlying cron scheduler. Non-blocking.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop the scheduler and wait for in-flight tasks to finish or for the
// supplied context to be cancelled, whichever comes first.
func (s *Scheduler) Stop(ctx context.Context) {
	stopped := s.cron.Stop().Done()
	select {
	case <-stopped:
	case <-ctx.Done():
		s.log.Warn("shutdown deadline reached before all fires completed")
	}
}

// EntryCount exposes the number of registered schedules for tests/diagnostics.
func (s *Scheduler) EntryCount() int {
	return len(s.cron.Entries())
}
