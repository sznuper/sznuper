package sideeffect

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExec_EnvPassed(t *testing.T) {
	env := []string{"HEALTHCHECK_ALERT_NAME=disk_check", "HEALTHCHECK_EVENT_TYPE=high_usage"}
	r := Exec(context.Background(), `echo "$HEALTHCHECK_ALERT_NAME:$HEALTHCHECK_EVENT_TYPE" >&2`, env)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	want := "disk_check:high_usage\n"
	if r.Stderr != want {
		t.Errorf("stderr = %q, want %q", r.Stderr, want)
	}
}

func TestExec_HealthcheckEventEnv(t *testing.T) {
	env := []string{"HEALTHCHECK_EVENT_TYPE=ok", "HEALTHCHECK_EVENT_USAGE=42"}
	r := Exec(context.Background(), `echo "$HEALTHCHECK_EVENT_TYPE:$HEALTHCHECK_EVENT_USAGE" >&2`, env)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if !strings.Contains(r.Stderr, "ok:42") {
		t.Errorf("stderr = %q, want 'ok:42'", r.Stderr)
	}
}

func TestExec_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	r := Exec(ctx, "sleep 10", nil)
	if r.Err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(r.Err.Error(), "timed out") {
		t.Errorf("error = %q, want timeout message", r.Err)
	}
}

func TestExec_FailureCaptured(t *testing.T) {
	r := Exec(context.Background(), "exit 1", nil)
	if r.Err == nil {
		t.Fatal("expected error for non-zero exit")
	}
}

func TestExecAll_Parallel(t *testing.T) {
	commands := []string{
		"echo a >&2",
		"echo b >&2",
		"echo c >&2",
	}
	results := ExecAll(context.Background(), commands, nil)
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("result[%d] error: %v", i, r.Err)
		}
	}
}

func TestExecAll_FailureCapturedNotFatal(t *testing.T) {
	commands := []string{
		"echo ok >&2",
		"exit 1",
		"echo ok2 >&2",
	}
	results := ExecAll(context.Background(), commands, nil)
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Err == nil {
		t.Errorf("result[1] should fail")
	}
	if results[2].Err != nil {
		t.Errorf("result[2] should succeed")
	}
}
