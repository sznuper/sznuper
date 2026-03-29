package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goccy/go-yaml"
)

// sectionBreak matches top-level YAML keys (no leading whitespace) to insert
// blank lines between sections like options, globals, channels, alerts.
var sectionBreak = regexp.MustCompile(`(?m)^(\S)`)

// Marshal serializes cfg to formatted YAML bytes.
func Marshal(cfg *Config) ([]byte, error) {
	data, err := yaml.MarshalWithOptions(cfg, yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	// Add blank lines between top-level sections.
	out := sectionBreak.ReplaceAllString(string(data), "\n$1")
	return []byte(out[1:]), nil // trim leading blank line
}

// Write serializes cfg to YAML and writes it to path, creating parent dirs.
func Write(cfg *Config, path string) error {
	data, err := Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// DefaultWritePath returns the default config write location:
// /etc/sznuper/config.yml for root, ~/.config/sznuper/config.yml otherwise.
func DefaultWritePath() string {
	if os.Getuid() == 0 {
		return "/etc/sznuper/config.yml"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yml"
	}
	return filepath.Join(home, ".config", "sznuper", "config.yml")
}

// DefaultOptions returns sensible default options based on whether
// the process is running as root.
func DefaultOptions() Options {
	if os.Getuid() == 0 {
		return Options{
			HealthchecksDir: "/etc/sznuper/healthchecks",
			CacheDir:        "/var/cache/sznuper",
			LogsDir:         "/var/log/sznuper",
		}
	}
	home, _ := os.UserHomeDir()
	return Options{
		HealthchecksDir: filepath.Join(home, ".config", "sznuper", "healthchecks"),
		CacheDir:        filepath.Join(home, ".cache", "sznuper"),
		LogsDir:         filepath.Join(home, ".local", "state", "sznuper", "logs"),
	}
}

// DefaultGlobals returns default global values.
func DefaultGlobals() map[string]any {
	hostname, _ := os.Hostname()
	return map[string]any{
		"hostname": hostname,
	}
}
