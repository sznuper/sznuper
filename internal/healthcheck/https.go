package healthcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sznuper/sznuper/internal/config"
)

func resolveHTTPS(uri string, sha config.SHA256, cacheDir string) (*ResolvedHealthcheck, error) {
	if sha.Hash == "" && !sha.Disabled {
		return nil, fmt.Errorf("https:// healthcheck requires sha256 field; use a hash or set sha256: false")
	}

	if sha.Hash != "" {
		if cacheDir == "" {
			return nil, fmt.Errorf("cache_dir must be set to use pinned https:// healthchecks")
		}
		cached := filepath.Join(cacheDir, sha.Hash)
		if _, err := os.Stat(cached); err == nil {
			return &ResolvedHealthcheck{URI: uri, Path: cached, Scheme: "https"}, nil
		}
		if err := downloadVerifyCache(uri, sha.Hash, cached); err != nil {
			return nil, err
		}
		return &ResolvedHealthcheck{URI: uri, Path: cached, Scheme: "https"}, nil
	}

	// Unpinned: sha.Disabled == true
	path, err := downloadToTemp(uri)
	if err != nil {
		return nil, err
	}
	return &ResolvedHealthcheck{URI: uri, Path: path, Scheme: "https"}, nil
}

func downloadVerifyCache(url, expectedHash, destPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("https:// download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("https:// download failed: HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destPath), "sznuper-hc-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	h := sha256.New()
	if _, err := io.Copy(tmp, io.TeeReader(resp.Body, h)); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("https:// download failed: %w", err)
	}
	tmp.Close()

	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedHash {
		os.Remove(tmpPath)
		return fmt.Errorf("sha256 mismatch for %s: expected %s, got %s", url, expectedHash, got)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("caching healthcheck: %w", err)
	}
	if err := os.Chmod(destPath, 0o755); err != nil {
		return fmt.Errorf("chmod cached healthcheck: %w", err)
	}
	return nil
}

func downloadToTemp(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("https:// download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("https:// download failed: HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "sznuper-hc-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("https:// download failed: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("chmod temp healthcheck: %w", err)
	}
	return tmp.Name(), nil
}
