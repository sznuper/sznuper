package healthcheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_FileRelative(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "my_check", "#!/bin/sh\necho status=ok")

	rc, err := Resolve("file://my_check", dir)
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

	rc, err := Resolve("file://"+path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Path != path {
		t.Errorf("path = %q, want %q", rc.Path, path)
	}
}

func TestResolve_FileMissing(t *testing.T) {
	_, err := Resolve("file://nonexistent", t.TempDir())
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

	_, err := Resolve("file://noexec", dir)
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

	_, err := Resolve("file://subdir", dir)
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestResolve_HttpsNotImplemented(t *testing.T) {
	_, err := Resolve("https://example.com/check", "")
	if err == nil {
		t.Fatal("expected error for https://")
	}
}

func TestResolve_UnsupportedScheme(t *testing.T) {
	_, err := Resolve("ftp://something", "")
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
