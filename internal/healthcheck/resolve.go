package healthcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sznuper/sznuper/internal/config"
)

// ResolvedHealthcheck holds the result of resolving a healthcheck URI.
type ResolvedHealthcheck struct {
	URI    string
	Path   string
	Scheme string
}

// ResolveOpts holds options for resolving a healthcheck URI.
type ResolveOpts struct {
	HealthchecksDir string
	CacheDir        string
	SHA256          config.SHA256
	ForceVerify     bool // skip cache hit; always re-download and verify
}

// Resolve resolves a healthcheck URI to an executable path.
//
// Supported schemes:
//   - file://name      → filepath.Join(opts.HealthchecksDir, name)
//   - file:///abs/path → absolute path as-is
//   - https://...      → download with sha256 verification and caching
func Resolve(uri string, opts ResolveOpts) (*ResolvedHealthcheck, error) {
	switch {
	case strings.HasPrefix(uri, "builtin://"):
		name := strings.TrimPrefix(uri, "builtin://")
		return &ResolvedHealthcheck{URI: uri, Path: name, Scheme: "builtin"}, nil
	case strings.HasPrefix(uri, "file://"):
		return resolveFile(uri, opts.HealthchecksDir)
	case strings.HasPrefix(uri, "https://"), strings.HasPrefix(uri, "http://"):
		return resolveHTTPS(uri, opts.SHA256, opts.CacheDir, opts.ForceVerify)
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
