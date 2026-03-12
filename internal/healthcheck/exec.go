package healthcheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ExecResult holds the output of executing a healthcheck.
type ExecResult struct {
	Stdout   string
	Stderr   string
	Duration time.Duration
	ExitCode int
	Env      []string
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

	var cmd *exec.Cmd
	if ape, _ := isAPEBinary(opts.Path); ape {
		cmd = exec.CommandContext(ctx, "/bin/sh", opts.Path)
	} else {
		cmd = exec.CommandContext(ctx, opts.Path)
	}
	env := buildEnv(opts)
	cmd.Env = env

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
		Env:      env,
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

// isAPEBinary reports whether the file at path is an APE (Actually Portable
// Executable) binary by checking for the MZ magic bytes.
func isAPEBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	var magic [2]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false, err
	}
	return magic[0] == 'M' && magic[1] == 'Z', nil
}

func buildEnv(opts ExecOpts) []string {
	env := []string{
		"HEALTHCHECK_TRIGGER=" + opts.TriggerType,
	}

	for k, v := range opts.Args {
		envKey := "HEALTHCHECK_ARG_" + strings.ToUpper(k)
		env = append(env, envKey+"="+formatArg(v))
	}

	return env
}

// formatArg converts a config arg value to a string, preserving the decimal
// point for float values (e.g. 8.0 stays "8.0" instead of becoming "8").
func formatArg(v any) string {
	switch n := v.(type) {
	case float64:
		s := strconv.FormatFloat(n, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	case float32:
		s := strconv.FormatFloat(float64(n), 'f', -1, 32)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	default:
		return fmt.Sprint(v)
	}
}
