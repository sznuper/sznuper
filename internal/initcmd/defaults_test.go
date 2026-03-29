package initcmd

import (
	"os"
	"path/filepath"
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

	// Exact count is system-dependent (overlays check /proc/meminfo, /proc/stat, systemctl).
	// On Linux: lifecycle + disk + memory + cpu = 4 minimum; with systemd: 5.
	if len(cfg.Alerts) < 2 {
		t.Errorf("expected at least 2 alerts (lifecycle + disk), got %d", len(cfg.Alerts))
	}
}

func TestMergeConfig(t *testing.T) {
	base := &config.Config{
		Channels: map[string]config.Channel{
			"logger": {URL: "logger://"},
		},
		Alerts: []config.Alert{
			{Name: "disk_usage"},
		},
	}

	overlay := &config.Config{
		Channels: map[string]config.Channel{
			"extra": {URL: "extra://"},
		},
		Alerts: []config.Alert{
			{Name: "ssh_journal"},
		},
	}

	mergeConfig(base, overlay)

	if len(base.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(base.Channels))
	}
	if _, ok := base.Channels["extra"]; !ok {
		t.Error("expected extra channel after merge")
	}
	if len(base.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(base.Alerts))
	}
	if base.Alerts[1].Name != "ssh_journal" {
		t.Errorf("expected ssh_journal alert, got %s", base.Alerts[1].Name)
	}
}

func TestFileExists(t *testing.T) {
	// Existing file should return true.
	tmp := filepath.Join(t.TempDir(), "exists")
	if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(tmp) {
		t.Error("fileExists returned false for existing file")
	}

	// Non-existent path should return false.
	if fileExists(filepath.Join(t.TempDir(), "nope")) {
		t.Error("fileExists returned true for non-existent path")
	}
}
