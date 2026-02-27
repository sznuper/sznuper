package runner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
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
	writeScript(t, dir, "#!/bin/sh\necho status=warning\necho usage=84\n")

	cfg := &config.Config{
		Options: config.Options{ChecksDir: dir},
		Globals: map[string]any{"hostname": "test-host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{
				Name:     "test_alert",
				Check:    "file://check.sh",
				Template: `{{check.status | upper}} {{globals.hostname}}: usage={{check.usage}}%`,
				Notify:   []config.NotifyTarget{{Service: "logger"}},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(cfg, logger)

	result := r.RunAlert(context.Background(), &cfg.Alerts[0], true)
	if result.Err != nil {
		t.Fatalf("unexpected error at stage %q: %v", result.ErrStage, result.Err)
	}
	if result.Status != "warning" {
		t.Errorf("status = %q, want %q", result.Status, "warning")
	}
	if result.Fields["usage"] != "84" {
		t.Errorf("usage = %q, want %q", result.Fields["usage"], "84")
	}
	if result.Rendered["logger"] != "WARNING test-host: usage=84%" {
		t.Errorf("rendered = %q, want %q", result.Rendered["logger"], "WARNING test-host: usage=84%")
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
		Options: config.Options{ChecksDir: t.TempDir()},
		Globals: map[string]any{"hostname": "host"},
		Alerts: []config.Alert{
			{Name: "bad", Check: "file://nonexistent"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	result := r.RunAlert(context.Background(), &cfg.Alerts[0], false)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if result.ErrStage != "resolve" {
		t.Errorf("err_stage = %q, want %q", result.ErrStage, "resolve")
	}
}

func TestRunAlert_ParseFails(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho no_status_key=here\n")

	cfg := &config.Config{
		Options: config.Options{ChecksDir: dir},
		Globals: map[string]any{"hostname": "host"},
		Alerts: []config.Alert{
			{Name: "bad_parse", Check: "file://check.sh", Template: "test"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	result := r.RunAlert(context.Background(), &cfg.Alerts[0], false)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if result.ErrStage != "parse" {
		t.Errorf("err_stage = %q, want %q", result.ErrStage, "parse")
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

func TestRunAll(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "#!/bin/sh\necho status=ok\n")

	cfg := &config.Config{
		Options: config.Options{ChecksDir: dir},
		Globals: map[string]any{"hostname": "host"},
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{Name: "a1", Check: "file://check.sh", Template: "msg1", Notify: []config.NotifyTarget{{Service: "logger"}}},
			{Name: "a2", Check: "file://check.sh", Template: "msg2", Notify: []config.NotifyTarget{{Service: "logger"}}},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	r := New(cfg, logger)

	results := r.RunAll(context.Background(), true)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for i, res := range results {
		if res.Err != nil {
			t.Errorf("result[%d] error: %v", i, res.Err)
		}
	}
}
