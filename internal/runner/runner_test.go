package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sznuper/sznuper/internal/config"
)

func writeScript(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRunAlert_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho '--- event'\necho type=high_usage\necho usage=84\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "test-host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{
				Name:        "test_alert",
				Healthcheck: "file://check.sh",
				Template:    `{{event.type | upper}} {{globals.hostname}}: usage={{event.usage}}%`,
				Notify:      []config.NotifyTarget{{Service: "logger"}},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(cfg, logger)

	result := <-r.RunAlert(context.Background(), &cfg.Alerts[0], true, nil, nil)
	if result.Err != nil {
		t.Fatalf("unexpected error at stage %q: %v", result.ErrStage, result.Err)
	}
	if result.EventType != "high_usage" {
		t.Errorf("event_type = %q, want %q", result.EventType, "high_usage")
	}
	if result.Fields["usage"] != "84" {
		t.Errorf("usage = %q, want %q", result.Fields["usage"], "84")
	}
	if result.Rendered["logger"] != "HIGH_USAGE test-host: usage=84%" {
		t.Errorf("rendered = %q, want %q", result.Rendered["logger"], "HIGH_USAGE test-host: usage=84%")
	}
	if len(result.Notified) != 1 || result.Notified[0] != "logger" {
		t.Errorf("notified = %v, want [logger]", result.Notified)
	}
	if !result.DryRun {
		t.Error("expected dry run")
	}
}

func TestRunAlert_ResolveFails(t *testing.T) {
	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: t.TempDir()},
		Globals: map[string]any{"hostname": "host"},
		Alerts: []config.Alert{
			{Name: "bad", Healthcheck: "file://nonexistent"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	result := <-r.RunAlert(context.Background(), &cfg.Alerts[0], false, nil, nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if result.ErrStage != "resolve" {
		t.Errorf("err_stage = %q, want %q", result.ErrStage, "resolve")
	}
}

func TestRunAlert_ParseFails(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho '--- event'\necho no_type_key=here\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "host"},
		Alerts: []config.Alert{
			{Name: "bad_parse", Healthcheck: "file://check.sh", Template: "test"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	result := <-r.RunAlert(context.Background(), &cfg.Alerts[0], false, nil, nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if result.ErrStage != "parse" {
		t.Errorf("err_stage = %q, want %q", result.ErrStage, "parse")
	}
}

func TestRunAlert_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\n# no output\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{
				Name:        "empty",
				Healthcheck: "file://check.sh",
				Template:    "test",
				Notify:      []config.NotifyTarget{{Service: "logger"}},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	var results []Result
	for res := range r.RunAlert(context.Background(), &cfg.Alerts[0], true, nil, nil) {
		results = append(results, res)
	}
	// Zero events = zero results
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
}

func TestFindAlert(t *testing.T) {
	cfg := &config.Config{
		Alerts: []config.Alert{
			{Name: "first"},
			{Name: "second"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	r := New(cfg, logger)

	if a := r.FindAlert("second"); a == nil || a.Name != "second" {
		t.Errorf("FindAlert(second) = %v", a)
	}
	if a := r.FindAlert("nonexistent"); a != nil {
		t.Errorf("FindAlert(nonexistent) = %v, want nil", a)
	}
}

func TestRunAlert_SideEffectsRun(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho '--- event'\necho type=ok\necho usage=42\n")

	outFile := filepath.Join(dir, "se-output.txt")
	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "test-host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{
				Name:        "se_test",
				Healthcheck: "file://check.sh",
				Template:    "msg",
				Notify:      []config.NotifyTarget{{Service: "logger"}},
				SideEffects: []string{
					fmt.Sprintf("echo \"$HEALTHCHECK_EVENT_TYPE:$HEALTHCHECK_EVENT_USAGE\" > %s", outFile),
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(cfg, logger)

	// Not dry-run so side effects actually run.
	result := <-r.RunAlertOpts(context.Background(), &cfg.Alerts[0], RunOpts{})
	if result.Err != nil {
		t.Fatalf("unexpected error at stage %q: %v", result.ErrStage, result.Err)
	}
	if result.SideEffectsRun != 1 {
		t.Errorf("SideEffectsRun = %d, want 1", result.SideEffectsRun)
	}

	// Verify the side effect received per-field env vars.
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading side effect output: %v", err)
	}
	if !strings.Contains(string(data), "ok:42") {
		t.Errorf("side effect output = %q, want to contain 'ok:42'", string(data))
	}
}

func TestRunAlert_SideEffectsSkippedDryRun(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho '--- event'\necho type=ok\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "test-host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{
				Name:        "se_dry",
				Healthcheck: "file://check.sh",
				Template:    "msg",
				Notify:      []config.NotifyTarget{{Service: "logger"}},
				SideEffects: []string{"echo should-not-run"},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(cfg, logger)

	result := <-r.RunAlert(context.Background(), &cfg.Alerts[0], true, nil, nil)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.SideEffectsRun != 0 {
		t.Errorf("SideEffectsRun = %d, want 0 in dry-run", result.SideEffectsRun)
	}
}

func TestRunAll(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho '--- event'\necho type=ok\n")

	cfg := &config.Config{
		Options: config.Options{HealthchecksDir: dir},
		Globals: map[string]any{"hostname": "host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{Name: "a1", Healthcheck: "file://check.sh", Template: "msg1", Notify: []config.NotifyTarget{{Service: "logger"}}},
			{Name: "a2", Healthcheck: "file://check.sh", Template: "msg2", Notify: []config.NotifyTarget{{Service: "logger"}}},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	var results []Result
	for res := range r.RunAll(context.Background(), true) {
		results = append(results, res)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for i, res := range results {
		if res.Err != nil {
			t.Errorf("result[%d] error: %v", i, res.Err)
		}
	}
}
