package initcmd

import (
	"embed"
	"fmt"
	"os/exec"

	"github.com/sznuper/sznuper/internal/config"
)

//go:embed defaults/*.yml
var defaultConfigs embed.FS

// overlay pairs a condition with the filename to merge when true.
type overlay struct {
	name  string
	check func() bool
}

var overlays = []overlay{
	{"systemd.yml", hasSystemd},
}

// DefaultConfig loads base.yml and merges all applicable overlays on top.
func DefaultConfig() (*config.Config, error) {
	cfg, err := loadEmbedded("base.yml")
	if err != nil {
		return nil, err
	}

	for _, o := range overlays {
		if !o.check() {
			continue
		}
		over, err := loadEmbedded(o.name)
		if err != nil {
			return nil, err
		}
		mergeConfig(cfg, over)
	}

	return cfg, nil
}

func loadEmbedded(name string) (*config.Config, error) {
	data, err := defaultConfigs.ReadFile("defaults/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading embedded default %s: %w", name, err)
	}
	cfg, err := config.LoadRaw(data)
	if err != nil {
		return nil, fmt.Errorf("parsing embedded default %s: %w", name, err)
	}
	return cfg, nil
}

// mergeConfig applies overlay on top of base: appends alerts, merges services.
func mergeConfig(base, overlay *config.Config) {
	for k, v := range overlay.Services {
		if base.Services == nil {
			base.Services = make(map[string]config.Service)
		}
		base.Services[k] = v
	}
	base.Alerts = append(base.Alerts, overlay.Alerts...)
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}
