package sideeffect

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// ExecResult holds the outcome of a single side effect command.
type ExecResult struct {
	Command string
	Stderr  string
	Err     error
}

// Exec runs a single shell command via /bin/sh -c with the given env.
func Exec(ctx context.Context, command string, env []string) ExecResult {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Env = env

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("side effect timed out: %s", command)
		}
	}

	return ExecResult{
		Command: command,
		Stderr:  stderr.String(),
		Err:     err,
	}
}

// ExecAll runs all commands in parallel and returns their results.
func ExecAll(ctx context.Context, commands []string, env []string) []ExecResult {
	results := make([]ExecResult, len(commands))
	var wg sync.WaitGroup
	for i, cmd := range commands {
		wg.Add(1)
		go func(i int, cmd string) {
			defer wg.Done()
			results[i] = Exec(ctx, cmd, env)
		}(i, cmd)
	}
	wg.Wait()
	return results
}
