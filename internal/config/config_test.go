package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultipleTriggers(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: file://test
    triggers:
      - interval: 30s
      - cron: "0 */6 * * *"
      - watch: /var/log/auth.log
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	triggers := cfg.Alerts[0].Triggers
	if len(triggers) != 3 {
		t.Fatalf("triggers count = %d, want 3", len(triggers))
	}
	if triggers[0].Interval != "30s" {
		t.Errorf("triggers[0].Interval = %q, want %q", triggers[0].Interval, "30s")
	}
	if triggers[1].Cron != "0 */6 * * *" {
		t.Errorf("triggers[1].Cron = %q, want %q", triggers[1].Cron, "0 */6 * * *")
	}
	if triggers[2].Watch != "/var/log/auth.log" {
		t.Errorf("triggers[2].Watch = %q, want %q", triggers[2].Watch, "/var/log/auth.log")
	}
}

func TestCooldownSimple(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: file://test
    triggers:
      - interval: 1m
    cooldown: 10m
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	if cfg.Alerts[0].Cooldown != "10m" {
		t.Errorf("cooldown = %q, want %q", cfg.Alerts[0].Cooldown, "10m")
	}
}

func TestSHA256False(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: https://example.com/check
    sha256: false
    triggers:
      - interval: 1h
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
    triggers:
      - interval: 1h
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

func TestNotifyStringAndObjectWithParams(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: file://test
    triggers:
      - interval: 1m
    template: "test"
    notify:
      - logfile
      - telegram:
          params:
            notification: "false"
`
	cfg := loadFromString(t, yml)
	notify := cfg.Alerts[0].Notify
	if len(notify) != 2 {
		t.Fatalf("notify count = %d, want 2", len(notify))
	}

	if notify[0].Service != "logfile" {
		t.Errorf("notify[0] service = %q, want %q", notify[0].Service, "logfile")
	}
	if len(notify[0].Params) != 0 {
		t.Errorf("notify[0] params = %v, want empty", notify[0].Params)
	}

	if notify[1].Service != "telegram" {
		t.Errorf("notify[1] service = %q, want %q", notify[1].Service, "telegram")
	}
	if notify[1].Params["notification"] != "false" {
		t.Errorf("notify[1] params = %v, want notification=false", notify[1].Params)
	}
}

func TestEventsConfig(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: file://test
    triggers:
      - interval: 1m
    template: "default"
    cooldown: 5m
    notify:
      - logger
    events:
      healthy: [ok]
      on_unmatched: drop
      override:
        failure:
          cooldown: 1m
          notify:
            - telegram:
                params:
                  notification: "false"
            - logger
        login:
          template: "Login by {{event.user}}"
          notify:
            - telegram
`
	cfg := loadFromString(t, yml)
	ev := cfg.Alerts[0].Events
	if ev == nil {
		t.Fatal("events is nil")
	}
	if len(ev.Healthy) != 1 || ev.Healthy[0] != "ok" {
		t.Errorf("healthy = %v, want [ok]", ev.Healthy)
	}
	if ev.OnUnmatched != "drop" {
		t.Errorf("on_unmatched = %q, want %q", ev.OnUnmatched, "drop")
	}

	failure, ok := ev.Override["failure"]
	if !ok {
		t.Fatal("missing override for failure")
	}
	if failure.Cooldown != "1m" {
		t.Errorf("failure cooldown = %q, want %q", failure.Cooldown, "1m")
	}
	if len(failure.Notify) != 2 {
		t.Fatalf("failure notify count = %d, want 2", len(failure.Notify))
	}
	if failure.Notify[0].Service != "telegram" {
		t.Errorf("failure notify[0] service = %q, want %q", failure.Notify[0].Service, "telegram")
	}
	if failure.Notify[0].Params["notification"] != "false" {
		t.Errorf("failure notify[0] params = %v, want notification=false", failure.Notify[0].Params)
	}

	login, ok := ev.Override["login"]
	if !ok {
		t.Fatal("missing override for login")
	}
	if login.Template != "Login by {{event.user}}" {
		t.Errorf("login template = %q, want %q", login.Template, "Login by {{event.user}}")
	}
	if len(login.Notify) != 1 || login.Notify[0].Service != "telegram" {
		t.Errorf("login notify = %v, want [telegram]", login.Notify)
	}
}

func TestSideEffects(t *testing.T) {
	yml := `
alerts:
  - name: test
    healthcheck: file://test
    triggers:
      - interval: 1m
    side_effects:
      - cat > /tmp/se-test.txt
      - curl -X POST http://localhost:8080/webhook
    template: "test"
    notify:
      - log
`
	cfg := loadFromString(t, yml)
	se := cfg.Alerts[0].SideEffects
	if len(se) != 2 {
		t.Fatalf("side_effects count = %d, want 2", len(se))
	}
	if se[0] != "cat > /tmp/se-test.txt" {
		t.Errorf("side_effects[0] = %q, want %q", se[0], "cat > /tmp/se-test.txt")
	}
	if se[1] != "curl -X POST http://localhost:8080/webhook" {
		t.Errorf("side_effects[1] = %q, want %q", se[1], "curl -X POST http://localhost:8080/webhook")
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

func TestNotifyOptional(t *testing.T) {
	cfg := loadFromString(t, `
alerts:
  - name: test
    healthcheck: file://test
    template: "test"
`)
	if len(cfg.Alerts[0].Notify) != 0 {
		t.Errorf("notify = %v, want empty", cfg.Alerts[0].Notify)
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
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(yml), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	_, err := Load(path)
	return err
}

func loadFromString(t *testing.T, yml string) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(yml), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}
