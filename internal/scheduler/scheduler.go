package scheduler

import (
	"context"
	"log/slog"
	"strconv"
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
// Lifecycle alerts fire at start (before loops) and stop (after loops exit).
func (s *Scheduler) Start(ctx context.Context, alerts []config.Alert, dryRun bool) {
	var lifecycle, regular []config.Alert
	for _, a := range alerts {
		if a.Trigger.Lifecycle {
			lifecycle = append(lifecycle, a)
		} else {
			regular = append(regular, a)
		}
	}

	totalAlerts := len(alerts)

	// Fire lifecycle alerts with event=started (blocking).
	s.fireLifecycle(ctx, lifecycle, "started", totalAlerts, dryRun)

	// Run regular alert loops.
	var wg sync.WaitGroup
	for i := range regular {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.runAlertLoop(ctx, &regular[i], dryRun)
		}(i)
	}
	wg.Wait()

	// Fire lifecycle alerts with event=stopped (fresh context so HTTP still works).
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.fireLifecycle(stopCtx, lifecycle, "stopped", totalAlerts, dryRun)
}

// fireLifecycle runs all lifecycle alerts with the given event, blocking until done.
func (s *Scheduler) fireLifecycle(ctx context.Context, alerts []config.Alert, event string, totalAlerts int, dryRun bool) {
	params := map[string]string{
		"event":  event,
		"alerts": strconv.Itoa(totalAlerts),
	}
	for i := range alerts {
		for result := range s.runner.RunAlertOpts(ctx, &alerts[i], runner.RunOpts{
			DryRun:        dryRun,
			BuiltinParams: params,
		}) {
			if s.onResult != nil {
				s.onResult(result)
			}
		}
	}
}

func (s *Scheduler) runAlertLoop(ctx context.Context, alert *config.Alert, dryRun bool) {
	opts := buildRunOpts(alert, dryRun)

	fire := func() {
		for result := range s.runner.RunAlertOpts(ctx, alert, opts) {
			if s.onResult != nil {
				s.onResult(result)
			}
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
	case alert.Trigger.Watch != "":
		s.runWatchLoop(ctx, alert, opts)
	case alert.Trigger.Pipe != "":
		s.runPipeLoop(ctx, alert, opts)
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

func buildRunOpts(alert *config.Alert, dryRun bool) runner.RunOpts {
	opts := runner.RunOpts{
		DryRun:   dryRun,
		Cooldown: cooldown.New(nil),
	}
	if alert.Events != nil && len(alert.Events.Healthy) > 0 {
		opts.State = &runner.AlertState{Healthy: true}
	}
	return opts
}
