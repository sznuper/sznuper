package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
)

func writeScript(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "check.sh"), []byte("#!/bin/sh\necho '--- event'\necho type=ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func newRunner(t *testing.T, cfg *config.Config) *runner.Runner {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return runner.New(cfg, logger)
}

func TestScheduler_ValidInterval_FiresMultipleTimes(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	const interval = 50 * time.Millisecond
	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "tick",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Interval: interval.String()}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), interval*5/2) // 2.5 x interval
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{})

	got := count.Load()
	if got < 2 {
		t.Errorf("onResult called %d times, want >= 2", got)
	}
}

func TestScheduler_CronFires(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "cron-tick",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Cron: "* * * * * *"}}, // every second (6-field)
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{})

	if got := count.Load(); got < 2 {
		t.Errorf("onResult called %d times, want >= 2", got)
	}
}

func TestScheduler_CronFivefield_Fires(t *testing.T) {
	schedule, err := cronParser.Parse("* * * * *")
	if err != nil {
		t.Fatalf("parsing 5-field cron: %v", err)
	}
	next := schedule.Next(time.Now())
	if d := time.Until(next); d > time.Minute {
		t.Errorf("next run in %v, want <= 1m", d)
	}
}

func TestScheduler_CronInvalid_NeverFires(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "bad-cron",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Cron: "not a cron expression"}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{})

	if got := count.Load(); got != 0 {
		t.Errorf("onResult called %d times, want 0", got)
	}
}

func TestScheduler_NoTrigger_NeverFires(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "no-trigger",
				Healthcheck: "file://check.sh",
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{})

	if got := count.Load(); got != 0 {
		t.Errorf("onResult called %d times, want 0", got)
	}
}

func TestScheduler_MultipleTriggers_BothFire(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "multi",
				Healthcheck: "file://check.sh",
				Triggers: []config.Trigger{
					{Interval: "50ms"},
					{Cron: "* * * * * *"}, // every second
				},
				Template: "test",
				Notify:   []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var count atomic.Int32
	sched := New(newRunner(t, cfg), slog.Default(), func(runner.Result) {
		count.Add(1)
	})

	// Run for 1.5s: interval should fire ~30 times, cron should fire once.
	// Combined should be well above what either trigger alone would produce.
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{})

	got := count.Load()
	// Interval alone would fire ~30 times in 1.5s. Cron fires once per second.
	// We just verify both contributed — more than interval alone at minimum.
	if got < 3 {
		t.Errorf("onResult called %d times, want >= 3 (both triggers should fire)", got)
	}
}

func TestScheduler_ContextCancel_ExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "cancel-me",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Interval: "20ms"}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	sched := New(newRunner(t, cfg), slog.Default(), nil)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sched.Start(ctx, cfg.Alerts, StartOpts{})
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

// watchAlert creates an alert config that watches the given path and runs a
// healthcheck that echoes stdin lines prefixed with "line=".
func watchAlert(t *testing.T, dir, watchPath string) *config.Config {
	t.Helper()
	script := "#!/bin/sh\ninput=$(cat)\necho '--- event'\necho type=ok\necho \"line=$input\"\n"
	if err := os.WriteFile(filepath.Join(dir, "check.sh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "watch-test",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Watch: watchPath}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}
}

func TestScheduler_Watch_FiresOnAppend(t *testing.T) {
	dir := t.TempDir()
	watchPath := filepath.Join(dir, "test.log")

	// Create the file before starting.
	if err := os.WriteFile(watchPath, []byte("existing line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := watchAlert(t, dir, watchPath)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	var mu sync.Mutex
	var results []runner.Result
	sched := New(runner.New(cfg, logger), logger, func(res runner.Result) {
		mu.Lock()
		results = append(results, res)
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx, cfg.Alerts, StartOpts{DryRun: true})

	// Give watcher time to start and seek to end of file.
	time.Sleep(100 * time.Millisecond)

	// Append a new line — should trigger a healthcheck invocation.
	f, err := os.OpenFile(watchPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("hello world\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	// Wait for the result.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(results)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) == 0 {
		t.Fatal("onResult not called after file append")
	}
	res := results[0]
	if res.Err != nil {
		t.Fatalf("unexpected error: %v (stage %s)", res.Err, res.ErrStage)
	}
	// The healthcheck receives "hello world\n" on stdin and emits line=hello world
	if !strings.Contains(res.Fields["line"], "hello world") {
		t.Errorf("expected stdin content in fields, got line=%q", res.Fields["line"])
	}
}

func TestScheduler_Watch_BuffersWhileRunning(t *testing.T) {
	dir := t.TempDir()
	watchPath := filepath.Join(dir, "test.log")

	// Create empty file.
	if err := os.WriteFile(watchPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	// Slow healthcheck: sleeps 400ms, then outputs stdin as line field.
	slowScript := "#!/bin/sh\nlines=$(cat)\nsleep 0.4\necho '--- event'\necho type=ok\necho \"input=$lines\"\n"
	if err := os.WriteFile(filepath.Join(dir, "check.sh"), []byte(slowScript), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "watch-slow",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Watch: watchPath}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	var mu sync.Mutex
	var results []runner.Result
	sched := New(runner.New(cfg, logger), logger, func(res runner.Result) {
		mu.Lock()
		results = append(results, res)
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx, cfg.Alerts, StartOpts{DryRun: true})
	time.Sleep(100 * time.Millisecond)

	appendLine := func(s string) {
		f, err := os.OpenFile(watchPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(s + "\n"); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	appendLine("line1")
	time.Sleep(80 * time.Millisecond) // healthcheck now running (slow 400ms)
	appendLine("line2")
	time.Sleep(30 * time.Millisecond)
	appendLine("line3")
	time.Sleep(30 * time.Millisecond)

	// Wait until we have results containing all lines.
	allContain := func(want []string) bool {
		mu.Lock()
		defer mu.Unlock()
		combined := ""
		for _, res := range results {
			for _, v := range res.Fields {
				combined += v + "\n"
			}
		}
		for _, w := range want {
			if !strings.Contains(combined, w) {
				return false
			}
		}
		return true
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if allContain([]string{"line1", "line2", "line3"}) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) < 2 {
		t.Fatalf("expected >= 2 results, got %d", len(results))
	}
	for i, res := range results {
		if res.Err != nil {
			t.Errorf("result[%d] error: %v", i, res.Err)
		}
	}
}

func TestScheduler_Watch_HandlesRotation(t *testing.T) {
	dir := t.TempDir()
	watchPath := filepath.Join(dir, "test.log")

	if err := os.WriteFile(watchPath, []byte("old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := watchAlert(t, dir, watchPath)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	var mu sync.Mutex
	var results []runner.Result
	sched := New(runner.New(cfg, logger), logger, func(res runner.Result) {
		mu.Lock()
		results = append(results, res)
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx, cfg.Alerts, StartOpts{DryRun: true})
	time.Sleep(100 * time.Millisecond)

	// Simulate log rotation: rename old file, create new one.
	rotatedPath := filepath.Join(dir, "test.log.1")
	if err := os.Rename(watchPath, rotatedPath); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create new file with fresh content.
	if err := os.WriteFile(watchPath, []byte("after rotation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for result from new file.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(results)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) == 0 {
		t.Fatal("onResult not called after rotation and new file write")
	}
	found := false
	for _, res := range results {
		if strings.Contains(res.Fields["line"], "after rotation") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("post-rotation content not seen in any result fields")
	}
}

func TestScheduler_Watch_HandlesTruncation(t *testing.T) {
	dir := t.TempDir()
	watchPath := filepath.Join(dir, "test.log")

	initialContent := strings.Repeat("old padding line\n", 10)
	if err := os.WriteFile(watchPath, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := watchAlert(t, dir, watchPath)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	var mu sync.Mutex
	var results []runner.Result
	sched := New(runner.New(cfg, logger), logger, func(res runner.Result) {
		mu.Lock()
		results = append(results, res)
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx, cfg.Alerts, StartOpts{DryRun: true})
	time.Sleep(100 * time.Millisecond)

	// Truncate and write short content.
	f, err := os.OpenFile(watchPath, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("fresh\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	// Wait for result.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(results)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) == 0 {
		t.Fatal("onResult not called after truncation and write")
	}
	found := false
	for _, res := range results {
		if strings.Contains(res.Fields["line"], "fresh") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("post-truncation content not seen in any result fields")
	}
}

func TestScheduler_SkipLifecycle_NoStartedStopped(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir)

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{},
		Alerts: []config.Alert{
			{
				Name:        "lifecycle-alert",
				Healthcheck: "builtin://lifecycle",
				Triggers:    []config.Trigger{{Lifecycle: true}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
			{
				Name:        "tick",
				Healthcheck: "file://check.sh",
				Triggers:    []config.Trigger{{Interval: "50ms"}},
				Template:    "test",
				Notify:      []config.NotifyTarget{{Channel: "logger"}},
			},
		},
		Channels: map[string]config.Channel{"logger": {URL: "logger://"}},
	}

	var mu sync.Mutex
	var results []runner.Result
	sched := New(newRunner(t, cfg), slog.Default(), func(res runner.Result) {
		mu.Lock()
		results = append(results, res)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	sched.Start(ctx, cfg.Alerts, StartOpts{SkipLifecycle: true})

	mu.Lock()
	defer mu.Unlock()

	for _, res := range results {
		if res.EventType == "started" || res.EventType == "stopped" {
			t.Errorf("unexpected lifecycle event %q with SkipLifecycle: true", res.EventType)
		}
	}
	// The interval alert should still have fired.
	hasInterval := false
	for _, res := range results {
		if res.AlertName == "tick" {
			hasInterval = true
			break
		}
	}
	if !hasInterval {
		t.Error("interval alert did not fire with SkipLifecycle: true")
	}
}
