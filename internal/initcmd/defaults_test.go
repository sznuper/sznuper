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

			_, err = config.LoadRaw(data)
			if err != nil {
				t.Fatalf("parsing %s: %v", e.Name(), err)
			}
		})
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

	if len(cfg.Alerts) < 3 {
		t.Errorf("expected at least 3 alerts, got %d", len(cfg.Alerts))
	}
}

func TestMergeConfig(t *testing.T) {
	base := &config.Config{
		Services: map[string]config.Service{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{Name: "disk_usage"},
		},
	}

	overlay := &config.Config{
		Services: map[string]config.Service{
			"extra": {URL: "extra://"},
		},
		Alerts: []config.Alert{
			{Name: "ssh_journal"},
		},
	}

	mergeConfig(base, overlay)

	if len(base.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(base.Services))
	}
	if _, ok := base.Services["extra"]; !ok {
		t.Error("expected extra service after merge")
	}
	if len(base.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(base.Alerts))
	}
	if base.Alerts[1].Name != "ssh_journal" {
		t.Errorf("expected ssh_journal alert, got %s", base.Alerts[1].Name)
	}
}
