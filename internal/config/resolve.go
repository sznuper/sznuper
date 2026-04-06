package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfigPaths returns the search order for config files.
func DefaultConfigPaths() []string {
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "sznuper", "config.yml"))
	}
	paths = append(paths, "/etc/sznuper/config.yml")
	return paths
}

// Resolve loads the config from the given explicit path, or searches the
// default locations.
func Resolve(explicit string) (*Config, error) {
	path, err := FindPath(explicit)
	if err != nil {
		return nil, err
	}

	return Load(path)
}

// FindPath returns the path to the config file. If explicit is non-empty, it
// validates and returns that path. Otherwise it searches the default locations.
func FindPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file not found: %s", explicit)
		}
		return explicit, nil
	}

	for _, p := range DefaultConfigPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no config file found (searched %v)", DefaultConfigPaths())
}
