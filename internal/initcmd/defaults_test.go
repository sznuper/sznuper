package initcmd

import (
	"testing"

	"github.com/sznuper/sznuper/internal/config"
)

func TestEmbeddedDefaultsParse(t *testing.T) {
	entries, err := defaultConfigs.ReadDir("defaults")
	if err != nil {
		t.Fatalf("reading embedded defaults dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no embedded default configs found")
	}

	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			data, err := defaultConfigs.ReadFile("defaults/" + e.Name())
			if err != nil {
				t.Fatalf("reading %s: %v", e.Name(), err)
			}

			cfg, err := config.LoadRaw(data)
			if err != nil {
				t.Fatalf("parsing %s: %v", e.Name(), err)
			}

			if len(cfg.Services) == 0 {
				t.Errorf("%s: expected at least one service", e.Name())
			}
			if len(cfg.Alerts) == 0 {
				t.Errorf("%s: expected at least one alert", e.Name())
			}
		})
	}
}

func TestSelectDefault(t *testing.T) {
	name := selectDefault()
	if name != "base.yaml" && name != "systemd.yaml" {
		t.Fatalf("unexpected default: %s", name)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error: %v", err)
	}

	if _, ok := cfg.Services["logger"]; !ok {
		t.Error("expected logger service in default config")
	}

	if len(cfg.Alerts) < 2 {
		t.Errorf("expected at least 2 alerts, got %d", len(cfg.Alerts))
	}
}
