package scheduler

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/cooldown"
	"github.com/sznuper/sznuper/internal/runner"
)

func (s *Scheduler) runWatchLoop(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State) {
	path := alert.Trigger.Watch
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.logger.Warn("watch: failed to create watcher", "alert", alert.Name, "error", err)
		return
	}
	defer watcher.Close()

	// Watch parent dir for CREATE events (log rotation recovery).
	if err := watcher.Add(dir); err != nil {
		s.logger.Warn("watch: failed to watch directory", "alert", alert.Name, "dir", dir, "error", err)
		return
	}

	// Open file and seek to end (skip pre-existing content).
	f, offset := openAndSeekEnd(path)
	if f != nil {
		_ = watcher.Add(path)
	}

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
			if f != nil {
				f.Close()
			}
			return

		case event, ok := <-watcher.Events:
			if !ok {
				if f != nil {
					f.Close()
				}
				return
			}
			switch {
			case event.Has(fsnotify.Write) && f != nil:
				// Truncation check: if file shrunk, reset to start.
				if info, err := os.Stat(path); err == nil && info.Size() < offset {
					offset = 0
					_, _ = f.Seek(0, io.SeekStart)
				}
				newData, _ := io.ReadAll(f)
				offset += int64(len(newData))
				buf = append(buf, newData...)
				if resultCh == nil && len(buf) > 0 {
					fire()
				}

			case event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove):
				// Log rotation: close handle, wait for CREATE.
				if f != nil {
					f.Close()
					f = nil
				}

			case event.Has(fsnotify.Create) && filepath.Base(event.Name) == base:
				// File re-created after rotation or fresh creation.
				if f != nil {
					f.Close()
				}
				f, _ = os.Open(path)
				offset = 0
				if f != nil {
					_ = watcher.Add(path)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				if f != nil {
					f.Close()
				}
				return
			}
			s.logger.Warn("watch: fsnotify error", "alert", alert.Name, "error", err)

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

// openAndSeekEnd opens the file at path and seeks to the end.
// Returns (nil, 0) if the file does not exist or cannot be opened.
func openAndSeekEnd(path string) (*os.File, int64) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0
	}
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		f.Close()
		return nil, 0
	}
	return f, offset
}
