package runner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/cooldown"
	"github.com/sznuper/sznuper/internal/healthcheck"
	"github.com/sznuper/sznuper/internal/notify"
)

// Runner orchestrates the healthcheck → parse → template → notify pipeline.
type Runner struct {
	cfg    *config.Config
	logger *slog.Logger
}

// New creates a Runner with the given config and logger.
func New(cfg *config.Config, logger *slog.Logger) *Runner {
	return &Runner{cfg: cfg, logger: logger}
}

// FindAlert returns the alert with the given name, or nil if not found.
func (r *Runner) FindAlert(name string) *config.Alert {
	for i := range r.cfg.Alerts {
		if r.cfg.Alerts[i].Name == name {
			return &r.cfg.Alerts[i]
		}
	}
	return nil
}

// RunAll fires every alert concurrently and returns a channel that yields results
// as they complete. The channel is closed once all alerts have finished.
func (r *Runner) RunAll(ctx context.Context, dryRun bool) <-chan Result {
	out := make(chan Result, len(r.cfg.Alerts))
	var wg sync.WaitGroup
	for i := range r.cfg.Alerts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out <- <-r.RunAlert(ctx, &r.cfg.Alerts[i], dryRun, nil)
		}(i)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// RunAlert executes a single alert through the full pipeline asynchronously.
// It returns a channel that will receive exactly one Result when the pipeline completes.
// Pass a non-nil cd to enforce cooldown; nil preserves the original behaviour (ok skips notify).
func (r *Runner) RunAlert(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State) <-chan Result {
	ch := make(chan Result, 1)
	go func() {
		ch <- r.runAlert(ctx, alert, dryRun, cd)
	}()
	return ch
}

func (r *Runner) runAlert(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State) Result {
	log := r.logger.With("alert", alert.Name)
	start := time.Now()

	result := Result{
		AlertName:      alert.Name,
		HealthcheckURI: alert.Healthcheck,
		DryRun:         dryRun,
	}

	// Stage 1: Resolve healthcheck URI.
	log.Info("resolving healthcheck", "uri", alert.Healthcheck)
	resolved, err := healthcheck.Resolve(alert.Healthcheck, healthcheck.ResolveOpts{
		HealthchecksDir: r.cfg.Options.HealthchecksDir,
		CacheDir:        r.cfg.Options.CacheDir,
		SHA256:          alert.SHA256,
	})
	if err != nil {
		result.Err = err
		result.ErrStage = "resolve"
		result.Duration = time.Since(start)
		log.Error("resolve failed", "error", err)
		return result
	}
	result.HealthcheckPath = resolved.Path
	log.Debug("healthcheck resolved", "path", resolved.Path, "scheme", resolved.Scheme)

	// Stage 2: Execute healthcheck.
	timeout, _ := time.ParseDuration(alert.Timeout)
	log.Info("executing healthcheck", "path", resolved.Path, "timeout", timeout)

	execResult, err := healthcheck.Exec(ctx, healthcheck.ExecOpts{
		Path:        resolved.Path,
		Timeout:     timeout,
		TriggerType: detectTriggerType(alert.Trigger),
		Args:        alert.Args,
	})
	if err != nil {
		result.Err = err
		result.ErrStage = "exec"
		result.Duration = time.Since(start)
		if execResult != nil {
			result.Stderr = execResult.Stderr
		}
		log.Error("exec failed", "error", err)
		return result
	}
	result.Stderr = execResult.Stderr
	result.Env = execResult.Env
	log.Debug("healthcheck executed", "exit_code", execResult.ExitCode, "duration", execResult.Duration, "stderr", execResult.Stderr)

	// Stage 3: Parse output.
	log.Info("parsing output")
	parsed, err := healthcheck.Parse(execResult.Stdout)
	if err != nil {
		result.Err = err
		result.ErrStage = "parse"
		result.Duration = time.Since(start)
		log.Error("parse failed", "error", err, "stdout", execResult.Stdout)
		return result
	}
	result.Status = parsed.Status
	result.Output = parsed.Lines
	result.Fields = parsed.Fields
	log.Debug("output parsed", "status", parsed.Status, "fields", parsed.Fields)

	// Stage 4: Build template data and resolve targets.
	log.Info("rendering templates")
	tmplData := notify.BuildTemplateData(
		r.cfg.Globals,
		alert.Name,
		parsed.Fields,
		alert.Args,
	)

	refs := mapNotifyRefs(alert.Notify)
	svcs := mapServiceDefs(r.cfg.Services)

	targets, err := notify.ResolveTargets(refs, svcs, alert.Template, tmplData)
	if err != nil {
		result.Err = err
		result.ErrStage = "template"
		result.Duration = time.Since(start)
		log.Error("template failed", "error", err)
		return result
	}

	result.Rendered = make(map[string]string, len(targets))
	for _, t := range targets {
		result.Rendered[t.ServiceName] = t.Message
	}
	log.Debug("templates rendered", "targets", len(targets))

	// Stage 5: Send notifications (or validate dry-run).
	if cd != nil {
		shouldNotify, isRecovery := cd.Check(parsed.Status)
		result.Suppressed = !shouldNotify
		result.IsRecovery = isRecovery
		if !shouldNotify {
			result.Duration = time.Since(start)
			log.Info("notification suppressed by cooldown", "status", parsed.Status)
			return result
		}
		// isRecovery=true: fall through to send even when status is "ok"
	} else {
		// Original behaviour: ok never notifies.
		if parsed.Status == "ok" {
			result.Duration = time.Since(start)
			log.Info("status ok, skipping notifications")
			return result
		}
	}

	for _, t := range targets {
		if dryRun {
			if err := notify.Validate(t); err != nil {
				result.Err = err
				result.ErrStage = "notify"
				result.Duration = time.Since(start)
				log.Error("notify validation failed (dry-run)", "service", t.ServiceName, "error", err)
				return result
			}
			result.Notified = append(result.Notified, t.ServiceName)
			log.Debug("would notify (dry-run)", "service", t.ServiceName, "message", t.Message)
			continue
		}

		log.Info("sending notification", "service", t.ServiceName)
		if err := notify.Send(t); err != nil {
			result.Err = err
			result.ErrStage = "notify"
			result.Duration = time.Since(start)
			log.Error("notify failed", "service", t.ServiceName, "error", err)
			return result
		}
		result.Notified = append(result.Notified, t.ServiceName)
		log.Debug("notification sent", "service", t.ServiceName)
	}

	result.Duration = time.Since(start)
	log.Info("alert completed", "status", result.Status, "duration", result.Duration)
	return result
}

func detectTriggerType(t config.Trigger) string {
	switch {
	case t.Watch != "":
		return "watch"
	case t.Cron != "":
		return "cron"
	default:
		return "interval"
	}
}

func mapNotifyRefs(targets []config.NotifyTarget) []notify.NotifyRef {
	refs := make([]notify.NotifyRef, len(targets))
	for i, t := range targets {
		refs[i] = notify.NotifyRef{
			ServiceName: t.Service,
			Template:    t.Template,
			Params:      t.Params,
		}
	}
	return refs
}

func mapServiceDefs(services map[string]config.Service) map[string]notify.ServiceDef {
	defs := make(map[string]notify.ServiceDef, len(services))
	for name, svc := range services {
		defs[name] = notify.ServiceDef{
			URL:    svc.URL,
			Params: svc.Params,
		}
	}
	return defs
}
