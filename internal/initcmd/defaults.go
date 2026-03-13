package initcmd

import (
	"embed"
	"fmt"
	"os/exec"

	"github.com/sznuper/sznuper/internal/config"
)

//go:embed defaults/*.yaml
var defaultConfigs embed.FS

// DefaultConfig returns a Config parsed from the appropriate embedded default.
func DefaultConfig() (*config.Config, error) {
	name := selectDefault()

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

func selectDefault() string {
	if hasSystemd() {
		return "systemd.yaml"
	}
	return "base.yaml"
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}
