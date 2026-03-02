package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
)

func writeScript(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func newRunner(t *testing.T, cfg *config.Config) *runner.Runner {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return runner.New(cfg, logger)
}

func TestScheduler_ValidInterval_FiresMultipleTimes(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho status=ok\n")

	const interval = 50 * time.Millisecond
	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "tick",
				Healthcheck: "file://check.sh",
				Trigger:     config.Trigger{Interval: interval.String()},
			},
		},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), interval*5/2) // 2.5 × interval
	defer cancel()

	sched.Start(ctx, cfg.Alerts, false)

	got := count.Load()
	if got < 2 {
		t.Errorf("onResult called %d times, want >= 2", got)
	}
}

func TestScheduler_NoInterval_NeverFires(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho status=ok\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "no-trigger",
				Healthcheck: "file://check.sh",
				// Trigger.Interval is empty
			},
		},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, false)

	if got := count.Load(); got != 0 {
		t.Errorf("onResult called %d times, want 0", got)
	}
}

func TestScheduler_ContextCancel_ExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho status=ok\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "cancel-me",
				Healthcheck: "file://check.sh",
				Trigger:     config.Trigger{Interval: "20ms"},
			},
		},
	}

	sched := New(newRunner(t, cfg), slog.Default(), nil)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sched.Start(ctx, cfg.Alerts, false)
	}()

	time.Sleep(60 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// clean exit
	case <-time.After(500 * time.Millisecond):
		t.Error("Start did not return after context cancellation")
	}
}
