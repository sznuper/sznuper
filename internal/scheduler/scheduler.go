package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
)

// OnResult is called with each alert result as it completes.
type OnResult func(runner.Result)

// Scheduler runs alerts on their configured trigger schedules.
type Scheduler struct {
	runner   *runner.Runner
	logger   *slog.Logger
	onResult OnResult
}

// New creates a Scheduler.
func New(r *runner.Runner, logger *slog.Logger, onResult OnResult) *Scheduler {
	return &Scheduler{runner: r, logger: logger, onResult: onResult}
}

// Start launches one goroutine per alert and blocks until ctx is done.
func (s *Scheduler) Start(ctx context.Context, alerts []config.Alert, dryRun bool) {
	var wg sync.WaitGroup
	for i := range alerts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.runAlertLoop(ctx, &alerts[i], dryRun)
		}(i)
	}
	wg.Wait()
}

func (s *Scheduler) runAlertLoop(ctx context.Context, alert *config.Alert, dryRun bool) {
	interval, err := time.ParseDuration(alert.Trigger.Interval)
	if err != nil || interval <= 0 {
		s.logger.Warn("skipping: no valid interval trigger", "alert", alert.Name)
		return
	}

	fire := func() {
		result := <-s.runner.RunAlert(ctx, alert, dryRun)
		if s.onResult != nil {
			s.onResult(result)
		}
	}

	fire() // immediate first run

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fire()
		}
	}
}
