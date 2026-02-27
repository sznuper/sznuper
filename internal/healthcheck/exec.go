package healthcheck

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecResult holds the output of executing a healthcheck.
type ExecResult struct {
	Stdout   string
	Stderr   string
	Duration time.Duration
	ExitCode int
}

// ExecOpts configures healthcheck execution.
type ExecOpts struct {
	Path        string
	Timeout     time.Duration
	TriggerType string
	Args        map[string]any
	Stdin       []byte
}

// Exec runs a healthcheck executable and captures its output.
// Non-zero exit codes are captured (not treated as errors).
// Timeouts are treated as errors.
func Exec(ctx context.Context, opts ExecOpts) (*ExecResult, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, opts.Path)
	cmd.Env = buildEnv(opts)

	if len(opts.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("healthcheck timed out after %s", opts.Timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("executing healthcheck: %w", err)
	}

	return result, nil
}

func buildEnv(opts ExecOpts) []string {
	env := []string{
		"HEALTHCHECK_TRIGGER=" + opts.TriggerType,
	}

	for k, v := range opts.Args {
		envKey := "HEALTHCHECK_ARG_" + strings.ToUpper(k)
		env = append(env, envKey+"="+fmt.Sprint(v))
	}

	return env
}
