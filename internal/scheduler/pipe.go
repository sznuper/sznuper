package scheduler

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/cooldown"
	"github.com/sznuper/sznuper/internal/runner"
)

func (s *Scheduler) runPipeLoop(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State) {
	for {
		err := s.runPipeOnce(ctx, alert, dryRun, cd)
		if ctx.Err() != nil {
			return
		}
		s.logger.Warn("pipe: command exited, restarting", "alert", alert.Name, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (s *Scheduler) runPipeOnce(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", alert.Trigger.Pipe)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pipe: start: %w", err)
	}

	dataCh := make(chan []byte, 16)
	go func() {
		defer close(dataCh)
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				dataCh <- chunk
			}
			if err != nil {
				return
			}
		}
	}()

	var buf []byte
	var resultCh <-chan runner.Result

	fire := func() {
		input := make([]byte, len(buf))
		copy(input, buf)
		buf = buf[:0]
		resultCh = s.runner.RunAlert(ctx, alert, dryRun, cd, input)
	}

	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil

		case chunk, ok := <-dataCh:
			if !ok {
				_ = cmd.Wait()
				return fmt.Errorf("pipe exited")
			}
			buf = append(buf, chunk...)
			if resultCh == nil && len(buf) > 0 {
				fire()
			}

		case res, ok := <-resultCh:
			if s.onResult != nil {
				s.onResult(res)
			}
			if !ok {
				resultCh = nil
				if len(buf) > 0 {
					fire()
				}
			}
		}
	}
}
