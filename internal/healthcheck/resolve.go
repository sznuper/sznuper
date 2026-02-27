package healthcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvedHealthcheck holds the result of resolving a healthcheck URI.
type ResolvedHealthcheck struct {
	URI    string
	Path   string
	Scheme string
}

// Resolve resolves a healthcheck URI to an executable path.
//
// Supported schemes:
//   - file://name      → filepath.Join(healthchecksDir, name)
//   - file:///abs/path → absolute path as-is
//   - https://...      → not yet implemented
func Resolve(uri, healthchecksDir string) (*ResolvedHealthcheck, error) {
	switch {
	case strings.HasPrefix(uri, "file://"):
		return resolveFile(uri, healthchecksDir)
	case strings.HasPrefix(uri, "https://"):
		return nil, fmt.Errorf("https:// healthchecks are not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported healthcheck URI scheme: %s", uri)
	}
}

func resolveFile(uri, healthchecksDir string) (*ResolvedHealthcheck, error) {
	raw := strings.TrimPrefix(uri, "file://")

	var path string
	if strings.HasPrefix(raw, "/") {
		path = raw
	} else {
		path = filepath.Join(healthchecksDir, raw)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("healthcheck not found: %s", path)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("healthcheck is a directory: %s", path)
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("healthcheck is not executable: %s", path)
	}

	return &ResolvedHealthcheck{
		URI:    uri,
		Path:   path,
		Scheme: "file",
	}, nil
}
