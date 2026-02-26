package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExampleConfig(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "bot123:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw")
	t.Setenv("TELEGRAM_CHAT_ID", "-100123456789")

	root := findProjectRoot(t)
	cfg, err := Load(filepath.Join(root, "config.example.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Hostname != "vps-01" {
		t.Errorf("hostname = %q, want %q", cfg.Hostname, "vps-01")
	}

	if cfg.Dirs == nil || cfg.Dirs.Checks != "/home/niar/checks" {
		t.Errorf("dirs.checks = %v, want /home/niar/checks", cfg.Dirs)
	}

	// envsubst in service URL
	svc, ok := cfg.Services["telegram"]
	if !ok {
		t.Fatal("missing service 'telegram'")
	}
	if want := "telegram://bot123:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw@telegram"; svc.URL != want {
		t.Errorf("service url = %q, want %q", svc.URL, want)
	}
	if svc.Options["chats"] != "-100123456789" {
		t.Errorf("service options[chats] = %q, want %q", svc.Options["chats"], "-100123456789")
	}

	if len(cfg.Alerts) != 1 {
		t.Fatalf("alerts count = %d, want 1", len(cfg.Alerts))
	}
	a := cfg.Alerts[0]
	if a.Name != "disk_check" {
		t.Errorf("alert name = %q, want %q", a.Name, "disk_check")
	}
	if a.Check != "file://disk_usage" {
		t.Errorf("alert check = %q, want %q", a.Check, "file://disk_usage")
	}
	if a.Trigger.Interval != "30s" {
		t.Errorf("trigger interval = %q, want %q", a.Trigger.Interval, "30s")
	}
	if a.Timeout != "10s" {
		t.Errorf("timeout = %q, want %q", a.Timeout, "10s")
	}

	// Simple cooldown
	if a.Cooldown.Simple != "5m" {
		t.Errorf("cooldown simple = %q, want %q", a.Cooldown.Simple, "5m")
	}

	// String notify
	if len(a.Notify) != 1 || a.Notify[0].Service != "telegram" {
		t.Errorf("notify = %v, want [telegram]", a.Notify)
	}
}

func TestCooldownStructured(t *testing.T) {
	yml := `
alerts:
  - name: test
    check: file://test
    trigger:
      interval: 1m
    cooldown:
      warning: 10m
      critical: 1m
      recovery: true
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	cd := cfg.Alerts[0].Cooldown
	if cd.Warning != "10m" {
		t.Errorf("cooldown warning = %q, want %q", cd.Warning, "10m")
	}
	if cd.Critical != "1m" {
		t.Errorf("cooldown critical = %q, want %q", cd.Critical, "1m")
	}
	if !cd.Recovery {
		t.Error("cooldown recovery = false, want true")
	}
	if cd.Simple != "" {
		t.Errorf("cooldown simple = %q, want empty", cd.Simple)
	}
}

func TestSHA256False(t *testing.T) {
	yml := `
alerts:
  - name: test
    check: https://example.com/check
    sha256: false
    trigger:
      interval: 1h
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	s := cfg.Alerts[0].SHA256
	if !s.Disabled {
		t.Error("sha256 disabled = false, want true")
	}
	if s.Hash != "" {
		t.Errorf("sha256 hash = %q, want empty", s.Hash)
	}
}

func TestSHA256String(t *testing.T) {
	yml := `
alerts:
  - name: test
    check: https://example.com/check
    sha256: a1b2c3d4e5f6
    trigger:
      interval: 1h
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	s := cfg.Alerts[0].SHA256
	if s.Hash != "a1b2c3d4e5f6" {
		t.Errorf("sha256 hash = %q, want %q", s.Hash, "a1b2c3d4e5f6")
	}
	if s.Disabled {
		t.Error("sha256 disabled = true, want false")
	}
}

func TestNotifyMixed(t *testing.T) {
	yml := `
alerts:
  - name: test
    check: file://test
    trigger:
      interval: 1m
    template: "test"
    notify:
      - logfile
      - service: telegram
        template: "*bold*"
        options:
          parsemode: MarkdownV2
`
	cfg := loadFromString(t, yml)
	notify := cfg.Alerts[0].Notify
	if len(notify) != 2 {
		t.Fatalf("notify count = %d, want 2", len(notify))
	}

	if notify[0].Service != "logfile" {
		t.Errorf("notify[0] service = %q, want %q", notify[0].Service, "logfile")
	}
	if notify[0].Template != "" {
		t.Errorf("notify[0] template = %q, want empty", notify[0].Template)
	}

	if notify[1].Service != "telegram" {
		t.Errorf("notify[1] service = %q, want %q", notify[1].Service, "telegram")
	}
	if notify[1].Template != "*bold*" {
		t.Errorf("notify[1] template = %q, want %q", notify[1].Template, "*bold*")
	}
	if notify[1].Options["parsemode"] != "MarkdownV2" {
		t.Errorf("notify[1] options = %v, want parsemode=MarkdownV2", notify[1].Options)
	}
}

func TestEnvsubst(t *testing.T) {
	yml := `
services:
  test:
    url: https://${TEST_TOKEN}@example.com
`
	t.Setenv("TEST_TOKEN", "secret123")
	cfg := loadFromString(t, yml)
	if cfg.Services["test"].URL != "https://secret123@example.com" {
		t.Errorf("url = %q, want envsubst applied", cfg.Services["test"].URL)
	}
}

// helpers

func loadFromString(t *testing.T, yml string) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yml), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
