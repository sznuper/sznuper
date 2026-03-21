package healthcheck

import (
	"context"
	"path/filepath"
	"sort"
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
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=ok\necho usage=10\n")

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
	if !strings.Contains(result.Stdout, "type=ok") {
		t.Errorf("stdout = %q, missing type=ok", result.Stdout)
	}
}

func TestExec_NonZeroExit(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=critical_usage\nexit 1\n")

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
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=ok\necho trigger=$HEALTHCHECK_TRIGGER\necho mount=$HEALTHCHECK_ARG_MOUNT\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "cron",
		AlertName:   "disk_check",
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

func TestFormatArg(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		// YAML decodes floats as float64
		{"float64 with decimal", float64(7.9), "7.9"},
		{"float64 whole number", float64(8.0), "8.0"},
		{"float64 zero", float64(0.0), "0.0"},
		{"float64 large", float64(100.0), "100.0"},
		{"float64 precision", float64(3.14159), "3.14159"},

		// YAML decodes integers as int
		{"int positive", 42, "42"},
		{"int zero", 0, "0"},
		{"int negative", -1, "-1"},

		// YAML decodes strings
		{"string simple", "hello", "hello"},
		{"string path", "/data", "/data"},
		{"string empty", "", ""},

		// YAML decodes booleans
		{"bool true", true, "true"},
		{"bool false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatArg(tt.val)
			if got != tt.want {
				t.Errorf("formatArg(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestBuildEnv(t *testing.T) {
	env := buildEnv(ExecOpts{
		TriggerType: "cron",
		AlertName:   "disk_check",
		Args: map[string]any{
			"mount":                  "/data",
			"threshold_warn_percent": float64(80.0),
			"threshold_crit_percent": float64(95.5),
			"raw":                    true,
			"count":                  42,
		},
	})

	sort.Strings(env)

	want := []string{
		"HEALTHCHECK_ALERT_NAME=disk_check",
		"HEALTHCHECK_ARG_COUNT=42",
		"HEALTHCHECK_ARG_MOUNT=/data",
		"HEALTHCHECK_ARG_RAW=true",
		"HEALTHCHECK_ARG_THRESHOLD_CRIT_PERCENT=95.5",
		"HEALTHCHECK_ARG_THRESHOLD_WARN_PERCENT=80.0",
		"HEALTHCHECK_TRIGGER=cron",
	}

	if len(env) != len(want) {
		t.Fatalf("got %d env vars, want %d:\n  got:  %v\n  want: %v", len(env), len(want), env, want)
	}
	for i := range want {
		if env[i] != want[i] {
			t.Errorf("env[%d] = %q, want %q", i, env[i], want[i])
		}
	}
}

func TestBuildEnv_NoArgs(t *testing.T) {
	env := buildEnv(ExecOpts{TriggerType: "interval"})
	if len(env) != 1 {
		t.Fatalf("got %d env vars, want 1: %v", len(env), env)
	}
	if env[0] != "HEALTHCHECK_TRIGGER=interval" {
		t.Errorf("env[0] = %q, want %q", env[0], "HEALTHCHECK_TRIGGER=interval")
	}
}

func TestExec_EnvArgTypes(t *testing.T) {
	// Script echoes back all HEALTHCHECK_ARG_* env vars
	script := tempScript(t, `#!/bin/sh
echo '--- event'
echo type=ok
echo float=$HEALTHCHECK_ARG_THRESHOLD
echo int=$HEALTHCHECK_ARG_COUNT
echo str=$HEALTHCHECK_ARG_MOUNT
echo bool=$HEALTHCHECK_ARG_RAW
`)

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
		Args: map[string]any{
			"threshold": float64(8.0),
			"count":     42,
			"mount":     "/data",
			"raw":       true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]string{
		"float=8.0": "float arg should preserve .0",
		"int=42":    "int arg should pass through",
		"str=/data": "string arg should pass through",
		"bool=true": "bool arg should pass through",
	}
	for substr, msg := range checks {
		if !strings.Contains(result.Stdout, substr) {
			t.Errorf("%s: stdout missing %q, got:\n%s", msg, substr, result.Stdout)
		}
	}
}

func TestExec_EnvInResult(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=ok\n")

	result, err := Exec(context.Background(), ExecOpts{
		Path:        script,
		TriggerType: "interval",
		AlertName:   "disk_check",
		Args:        map[string]any{"mount": "/"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Env) != 3 {
		t.Fatalf("got %d env vars, want 3: %v", len(result.Env), result.Env)
	}

	found := false
	for _, e := range result.Env {
		if e == "HEALTHCHECK_ARG_MOUNT=/" {
			found = true
		}
	}
	if !found {
		t.Errorf("Env missing HEALTHCHECK_ARG_MOUNT=/, got: %v", result.Env)
	}
}

func TestExec_Stderr(t *testing.T) {
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=ok\necho debug info >&2\n")

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
	script := tempScript(t, "#!/bin/sh\nread line\necho '--- event'\necho type=ok\necho line=$line\n")

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
	script := tempScript(t, "#!/bin/sh\necho '--- event'\necho type=ok\necho home=$HOME\n")

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
