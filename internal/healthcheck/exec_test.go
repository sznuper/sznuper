package healthcheck

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tempScript(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, dir, "check.sh", content)
	return filepath.Join(dir, "check.sh")
}

func TestExec_Success(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho status=ok\necho usage=10\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "status=ok") {
		t.Errorf("stdout = %q, missing status=ok", result.Stdout)
	}
}

func TestExec_NonZeroExit(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho status=critical\nexit 1\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
	})
	if err != nil {
		t.Fatalf("non-zero exit should not be an error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
}

func TestExec_Timeout(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\nsleep 10\n")

	_, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		Timeout:     100 * time.Millisecond,
		TriggerType: "interval",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want timeout message", err.Error())
	}
}

func TestExec_EnvArgs(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho status=ok\necho trigger=$HEALTHCHECK_TRIGGER\necho mount=$HEALTHCHECK_ARG_MOUNT\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "cron",
		Args:        map[string]any{"mount": "/data"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stdout, "trigger=cron") {
		t.Errorf("stdout missing trigger=cron: %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "mount=/data") {
		t.Errorf("stdout missing mount=/data: %q", result.Stdout)
	}
}

func TestExec_Stderr(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho status=ok\necho debug info >&2\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stderr, "debug info") {
		t.Errorf("stderr = %q, missing debug info", result.Stderr)
	}
}

func TestExec_Stdin(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\nread line\necho status=ok\necho line=$line\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "watch",
		Stdin:       []byte("hello world\n"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stdout, "line=hello world") {
		t.Errorf("stdout = %q, missing stdin content", result.Stdout)
	}
}

func TestExec_IsolatedEnv(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho status=ok\necho home=$HOME\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stdout, "home=\n") {
		t.Errorf("HOME should be empty in isolated env, stdout: %q", result.Stdout)
	}
}
