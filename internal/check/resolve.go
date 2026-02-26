package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvedCheck holds the result of resolving a check URI.
type ResolvedCheck struct {
	URI    string
	Path   string
	Scheme string
}

// Resolve resolves a check URI to an executable path.
//
// Supported schemes:
//   - file://name      → filepath.Join(checksDir, name)
//   - file:///abs/path → absolute path as-is
//   - https://...      → not yet implemented
func Resolve(uri, checksDir string) (*ResolvedCheck, error) {
	switch {
	case strings.HasPrefix(uri, "file://"):
		return resolveFile(uri, checksDir)
	case strings.HasPrefix(uri, "https://"):
		return nil, fmt.Errorf("https:// checks are not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported check URI scheme: %s", uri)
	}
}

func resolveFile(uri, checksDir string) (*ResolvedCheck, error) {
	raw := strings.TrimPrefix(uri, "file://")

	var path string
	if strings.HasPrefix(raw, "/") {
		path = raw
	} else {
		path = filepath.Join(checksDir, raw)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("check not found: %s", path)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("check is a directory: %s", path)
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("check is not executable: %s", path)
	}

	return &ResolvedCheck{
		URI:    uri,
		Path:   path,
		Scheme: "file",
	}, nil
}
