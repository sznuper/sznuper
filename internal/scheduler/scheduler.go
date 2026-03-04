package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/cooldown"
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
	cd := buildCooldownState(alert.Cooldown)

	fire := func() {
		result := <-s.runner.RunAlert(ctx, alert, dryRun, cd)
		if s.onResult != nil {
			s.onResult(result)
		}
	}

	switch {
	case alert.Trigger.Interval != "":
		interval, err := time.ParseDuration(alert.Trigger.Interval)
		if err != nil || interval <= 0 {
			s.logger.Warn("skipping: invalid interval", "alert", alert.Name, "interval", alert.Trigger.Interval)
			return
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
	case alert.Trigger.Cron != "":
		s.runCronLoop(ctx, alert.Name, alert.Trigger.Cron, fire)
	default:
		s.logger.Warn("skipping: no trigger configured", "alert", alert.Name)
	}
}

// cronParser accepts both 5-field (minute-level) and 6-field (with seconds) expressions.
var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func (s *Scheduler) runCronLoop(ctx context.Context, alertName, expr string, fire func()) {
	cr := cron.New(cron.WithParser(cronParser))
	if _, err := cr.AddFunc(expr, fire); err != nil {
		s.logger.Warn("skipping: invalid cron expression", "alert", alertName, "cron", expr, "error", err)
		return
	}
	cr.Start()
	<-ctx.Done()
	cr.Stop()
}

func buildCooldownState(cd config.Cooldown) *cooldown.State {
	w := parseCooldownValue(effectiveCooldownValue(cd.Warning, cd.Simple))
	c := parseCooldownValue(effectiveCooldownValue(cd.Critical, cd.Simple))
	if w == 0 && c == 0 {
		return nil
	}
	return cooldown.New(w, c, cd.Recovery, nil)
}

func effectiveCooldownValue(specific, simple string) string {
	if specific != "" {
		return specific
	}
	return simple
}

func parseCooldownValue(s string) time.Duration {
	if s == "inf" {
		return cooldown.Infinite
	}
	d, _ := time.ParseDuration(s)
	return d
}
