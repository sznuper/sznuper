package healthcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sznuper/sznuper/internal/config"
)

func TestResolve_FileRelative(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "my_check", "#!/bin/sh\necho status=ok")

	rc, err := Resolve("file://my_check", ResolveOpts{HealthchecksDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Path != filepath.Join(dir, "my_check") {
		t.Errorf("path = %q, want %q", rc.Path, filepath.Join(dir, "my_check"))
	}
	if rc.Scheme != "file" {
		t.Errorf("scheme = %q, want %q", rc.Scheme, "file")
	}
}

func TestResolve_FileAbsolute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abs_check")
	writeExecutable(t, dir, "abs_check", "#!/bin/sh\necho status=ok")

	rc, err := Resolve("file://"+path, ResolveOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Path != path {
		t.Errorf("path = %q, want %q", rc.Path, path)
	}
}

func TestResolve_FileMissing(t *testing.T) {
	_, err := Resolve("file://nonexistent", ResolveOpts{HealthchecksDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolve_FileNotExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noexec")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve("file://noexec", ResolveOpts{HealthchecksDir: dir})
	if err == nil {
		t.Fatal("expected error for non-executable file")
	}
}

func TestResolve_FileIsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve("file://subdir", ResolveOpts{HealthchecksDir: dir})
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestResolve_UnsupportedScheme(t *testing.T) {
	_, err := Resolve("ftp://something", ResolveOpts{})
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestResolve_HTTPSPinned_DownloadsAndCaches(t *testing.T) {
	content := []byte("#!/bin/sh\necho status=ok\n")
	hash := sha256hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	opts := ResolveOpts{
		CacheDir: cacheDir,
		SHA256:   config.SHA256{Hash: hash},
	}

	rc, err := Resolve(srv.URL+"/check", opts)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if rc.Scheme != "https" {
		t.Errorf("scheme = %q, want https", rc.Scheme)
	}
	cached := filepath.Join(cacheDir, hash)
	if rc.Path != cached {
		t.Errorf("path = %q, want %q", rc.Path, cached)
	}

	// Second resolve: server shut down, must use cache.
	srv.Close()
	rc2, err := Resolve(srv.URL+"/check", opts)
	if err != nil {
		t.Fatalf("second resolve (cached): %v", err)
	}
	if rc2.Path != cached {
		t.Errorf("cached path = %q, want %q", rc2.Path, cached)
	}
}

func TestResolve_HTTPSPinned_HashMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\necho status=ok\n"))
	}))
	defer srv.Close()

	_, err := Resolve(srv.URL+"/check", ResolveOpts{
		CacheDir: t.TempDir(),
		SHA256:   config.SHA256{Hash: strings.Repeat("a", 64)},
	})
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("error %q should mention mismatch", err.Error())
	}
}

func TestResolve_HTTPSPinned_ServerDown_UsesCache(t *testing.T) {
	content := []byte("#!/bin/sh\necho status=ok\n")
	hash := sha256hex(content)

	cacheDir := t.TempDir()
	cached := filepath.Join(cacheDir, hash)
	if err := os.WriteFile(cached, content, 0o755); err != nil {
		t.Fatal(err)
	}

	// Server is never started — resolve must use the pre-existing cache.
	rc, err := Resolve("https://down.invalid/check", ResolveOpts{
		CacheDir: cacheDir,
		SHA256:   config.SHA256{Hash: hash},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Path != cached {
		t.Errorf("path = %q, want %q", rc.Path, cached)
	}
}

func TestResolve_HTTPSUnpinned(t *testing.T) {
	content := []byte("#!/bin/sh\necho status=ok\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	rc, err := Resolve(srv.URL+"/check", ResolveOpts{
		SHA256: config.SHA256{Disabled: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Scheme != "https" {
		t.Errorf("scheme = %q, want https", rc.Scheme)
	}
	info, err := os.Stat(rc.Path)
	if err != nil {
		t.Fatalf("stat temp file: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("temp file is not executable")
	}
}

func TestResolve_HTTPSMissingSHA256(t *testing.T) {
	_, err := Resolve("https://example.com/check", ResolveOpts{})
	if err == nil {
		t.Fatal("expected error for missing sha256")
	}
}

func TestResolve_HTTPSNoCacheDir(t *testing.T) {
	_, err := Resolve("https://example.com/check", ResolveOpts{
		SHA256: config.SHA256{Hash: strings.Repeat("a", 64)},
	})
	if err == nil {
		t.Fatal("expected error for missing cache_dir")
	}
	if !strings.Contains(err.Error(), "cache_dir") {
		t.Errorf("error %q should mention cache_dir", err.Error())
	}
}

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
