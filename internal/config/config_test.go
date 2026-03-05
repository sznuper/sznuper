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

	if cfg.Globals["hostname"] != "vps-01" {
		t.Errorf("globals[hostname] = %q, want %q", cfg.Globals["hostname"], "vps-01")
	}

	if cfg.Options.HealthchecksDir == "" {
		t.Error("options.healthchecks_dir is empty")
	}

	// envsubst in service URL
	svc, ok := cfg.Services["telegram"]
	if !ok {
		t.Fatal("missing service 'telegram'")
	}
	if want := "telegram://bot123:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw@telegram"; svc.URL != want {
		t.Errorf("service url = %q, want %q", svc.URL, want)
	}
	if svc.Params["chats"] != "-100123456789" {
		t.Errorf("service params[chats] = %q, want %q", svc.Params["chats"], "-100123456789")
	}

	if len(cfg.Alerts) != 1 {
		t.Fatalf("alerts count = %d, want 1", len(cfg.Alerts))
	}
	a := cfg.Alerts[0]
	if a.Name != "disk_check_https" {
		t.Errorf("alert name = %q, want %q", a.Name, "disk_check_https")
	}
	if a.Healthcheck != "https://github.com/sznuper/healthchecks/releases/download/v0.1.0/disk_usage" {
		t.Errorf("alert healthcheck = %q, want %q", a.Healthcheck, "https://github.com/sznuper/healthchecks/releases/download/v0.1.0/disk_usage")
	}
	if a.Trigger.Interval == "" {
		t.Error("trigger interval is empty")
	}
	if a.Timeout != "10s" {
		t.Errorf("timeout = %q, want %q", a.Timeout, "10s")
	}

	// Cooldown is set (simple or per-status)
	if a.Cooldown.Simple == "" && a.Cooldown.Warning == "" {
		t.Error("cooldown not set in example config")
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
    healthcheck: file://test
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
    healthcheck: https://example.com/check
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
    healthcheck: https://example.com/check
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
    healthcheck: file://test
    trigger:
      interval: 1m
    template: "test"
    notify:
      - logfile
      - service: telegram
        template: "*bold*"
        params:
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
	if notify[1].Params["parsemode"] != "MarkdownV2" {
		t.Errorf("notify[1] params = %v, want parsemode=MarkdownV2", notify[1].Params)
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

// Validation tests

func TestValidation_AlertMissingName(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - healthcheck: file://test
    template: "test"
    notify: [log]
`); err == nil {
		t.Fatal("expected error for missing alert name")
	}
}

func TestValidation_AlertMissingHealthcheck(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - name: test
    template: "test"
    notify: [log]
`); err == nil {
		t.Fatal("expected error for missing alert healthcheck")
	}
}

func TestValidation_AlertMissingTemplate(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - name: test
    healthcheck: file://test
    notify: [log]
`); err == nil {
		t.Fatal("expected error for missing alert template")
	}
}

func TestValidation_AlertEmptyNotify(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - name: test
    healthcheck: file://test
    template: "test"
`); err == nil {
		t.Fatal("expected error for missing notify targets")
	}
}

func TestValidation_ServiceMissingURL(t *testing.T) {
	if err := loadErr(t, `
services:
  bad:
    params:
      foo: bar
`); err == nil {
		t.Fatal("expected error for service missing url")
	}
}

func TestValidation_NotifyTargetMissingService(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - name: test
    healthcheck: file://test
    template: "test"
    notify:
      - service: ""
        template: "override"
`); err == nil {
		t.Fatal("expected error for notify target with empty service")
	}
}

func TestStrict_UnknownTopLevelField(t *testing.T) {
	if err := loadErr(t, `unknown_field: foo`); err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
}

func TestStrict_UnknownAlertField(t *testing.T) {
	if err := loadErr(t, `
alerts:
  - name: test
    healthcheck: file://test
    template: "test"
    notify: [log]
    typo_field: oops
`); err == nil {
		t.Fatal("expected error for unknown alert field")
	}
}

// helpers

func loadErr(t *testing.T, yml string) error {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yml), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	_, err := Load(path)
	return err
}

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
