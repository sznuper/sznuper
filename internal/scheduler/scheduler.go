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

// StartOpts holds options for Scheduler.Start.
type StartOpts struct {
	DryRun        bool
	SkipLifecycle bool // suppress started/stopped lifecycle events (used during reload)
}

// Start launches one goroutine per alert and blocks until ctx is done.
// Lifecycle alerts fire at start (before loops) and stop (after loops exit),
// unless SkipLifecycle is set.
func (s *Scheduler) Start(ctx context.Context, alerts []config.Alert, opts StartOpts) {
	var lifecycle, regular []config.Alert
	for _, a := range alerts {
		if HasLifecycleTrigger(a.Triggers) {
			lifecycle = append(lifecycle, a)
		} else {
			regular = append(regular, a)
		}
	}

	totalAlerts := len(alerts)

	if !opts.SkipLifecycle {
		// Fire lifecycle alerts with event=started (blocking).
		s.FireLifecycle(ctx, lifecycle, "started", totalAlerts, opts.DryRun)
	}

	// Run regular alert loops.
	var wg sync.WaitGroup
	for i := range regular {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.runAlertLoop(ctx, &regular[i], opts.DryRun)
		}(i)
	}
	wg.Wait()

	if !opts.SkipLifecycle {
		// Fire lifecycle alerts with event=stopped (fresh context so HTTP still works).
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.FireLifecycle(stopCtx, lifecycle, "stopped", totalAlerts, opts.DryRun)
	}
}

// FireLifecycle runs all lifecycle alerts with the given event, blocking until done.
func (s *Scheduler) FireLifecycle(ctx context.Context, alerts []config.Alert, event string, totalAlerts int, dryRun bool) {
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

	if len(alert.Triggers) == 0 {
		s.logger.Warn("skipping: no triggers configured", "alert", alert.Name)
		return
	}

	var wg sync.WaitGroup
	for i := range alert.Triggers {
		wg.Add(1)
		go func(trigger config.Trigger) {
			defer wg.Done()
			s.runTrigger(ctx, alert, trigger, opts)
		}(alert.Triggers[i])
	}
	wg.Wait()
}

func (s *Scheduler) runTrigger(ctx context.Context, alert *config.Alert, trigger config.Trigger, opts runner.RunOpts) {
	triggerType := detectTriggerType(trigger)

	fire := func() {
		callOpts := opts
		callOpts.TriggerType = triggerType
		for result := range s.runner.RunAlertOpts(ctx, alert, callOpts) {
			if s.onResult != nil {
				s.onResult(result)
			}
		}
	}

	switch {
	case trigger.Interval != "":
		interval, err := time.ParseDuration(trigger.Interval)
		if err != nil || interval <= 0 {
			s.logger.Warn("skipping: invalid interval", "alert", alert.Name, "interval", trigger.Interval)
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
	case trigger.Cron != "":
		s.runCronLoop(ctx, alert.Name, trigger.Cron, fire)
	case trigger.Watch != "":
		s.runWatchLoop(ctx, alert, trigger, opts)
	case trigger.Pipe != "":
		s.runPipeLoop(ctx, alert, trigger, opts)
	default:
		s.logger.Warn("skipping: empty trigger", "alert", alert.Name)
	}
}

// HasLifecycleTrigger returns true if any trigger in the list is a lifecycle trigger.
func HasLifecycleTrigger(triggers []config.Trigger) bool {
	for _, t := range triggers {
		if t.Lifecycle {
			return true
		}
	}
	return false
}

func detectTriggerType(t config.Trigger) string {
	switch {
	case t.Lifecycle:
		return "lifecycle"
	case t.Pipe != "":
		return "pipe"
	case t.Watch != "":
		return "watch"
	case t.Cron != "":
		return "cron"
	default:
		return "interval"
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
