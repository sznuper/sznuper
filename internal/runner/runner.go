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
			for res := range r.RunAlertOpts(ctx, &r.cfg.Alerts[i], RunOpts{DryRun: dryRun}) {
				out <- res
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// RunOpts holds optional parameters for RunAlert.
type RunOpts struct {
	DryRun        bool
	Cooldown      *cooldown.State
	Stdin         []byte
	BuiltinParams map[string]string // params for builtin:// healthchecks
}

// RunAlert executes a single alert through the full pipeline asynchronously.
// It returns a channel that yields one Result per healthcheck record, then closes.
// Single-record healthchecks yield exactly one result (same as before).
// Multi-record healthchecks (using "--- records" / "--- record" separators) yield one result per record.
// Pass a non-nil cd to enforce cooldown; nil preserves the original behaviour (ok skips notify).
// Pass non-nil stdin to pipe bytes to the healthcheck process via stdin.
func (r *Runner) RunAlert(ctx context.Context, alert *config.Alert, dryRun bool, cd *cooldown.State, stdin []byte) <-chan Result {
	return r.RunAlertOpts(ctx, alert, RunOpts{DryRun: dryRun, Cooldown: cd, Stdin: stdin})
}

// RunAlertOpts is like RunAlert but accepts RunOpts for extended options.
func (r *Runner) RunAlertOpts(ctx context.Context, alert *config.Alert, opts RunOpts) <-chan Result {
	ch := make(chan Result, 8)
	go func() {
		defer close(ch)
		r.runAlert(ctx, alert, opts, ch)
	}()
	return ch
}

func (r *Runner) runAlert(ctx context.Context, alert *config.Alert, opts RunOpts, out chan<- Result) {
	log := r.logger.With("alert", alert.Name)
	start := time.Now()

	dryRun := opts.DryRun
	cd := opts.Cooldown

	base := Result{
		AlertName:      alert.Name,
		HealthcheckURI: alert.Healthcheck,
		DryRun:         dryRun,
	}

	sendErr := func(r Result) {
		r.Duration = time.Since(start)
		out <- r
	}

	// Stage 1: Resolve healthcheck URI.
	log.Info("resolving healthcheck", "uri", alert.Healthcheck)
	resolved, err := healthcheck.Resolve(alert.Healthcheck, healthcheck.ResolveOpts{
		HealthchecksDir: r.cfg.Options.HealthchecksDir,
		CacheDir:        r.cfg.Options.CacheDir,
		SHA256:          alert.SHA256,
	})
	if err != nil {
		base.Err = err
		base.ErrStage = "resolve"
		log.Error("resolve failed", "error", err)
		sendErr(base)
		return
	}
	base.HealthcheckPath = resolved.Path
	log.Debug("healthcheck resolved", "path", resolved.Path, "scheme", resolved.Scheme)

	// Stage 2: Execute healthcheck.
	var execResult *healthcheck.ExecResult
	if resolved.Scheme == "builtin" {
		log.Info("executing builtin healthcheck", "name", resolved.Path)
		execResult, err = healthcheck.ExecBuiltin(resolved.Path, opts.BuiltinParams)
	} else {
		timeout, _ := time.ParseDuration(alert.Timeout)
		log.Info("executing healthcheck", "path", resolved.Path, "timeout", timeout)
		execResult, err = healthcheck.Exec(ctx, healthcheck.ExecOpts{
			Path:        resolved.Path,
			Timeout:     timeout,
			TriggerType: detectTriggerType(alert.Trigger),
			Args:        alert.Args,
			Stdin:       opts.Stdin,
		})
	}
	if err != nil {
		base.Err = err
		base.ErrStage = "exec"
		if execResult != nil {
			base.Stderr = execResult.Stderr
		}
		log.Error("exec failed", "error", err)
		sendErr(base)
		return
	}
	base.Stderr = execResult.Stderr
	base.Env = execResult.Env
	log.Debug("healthcheck executed", "exit_code", execResult.ExitCode, "duration", execResult.Duration, "stderr", execResult.Stderr)

	// Stage 3: Parse output (multi-record aware).
	log.Info("parsing output")
	multi, err := healthcheck.ParseMulti(execResult.Stdout)
	if err != nil {
		base.Err = err
		base.ErrStage = "parse"
		log.Error("parse failed", "error", err, "stdout", execResult.Stdout)
		sendErr(base)
		return
	}
	log.Debug("output parsed", "records", len(multi.Records))

	refs := mapNotifyRefs(alert.Notify)
	svcs := mapServiceDefs(r.cfg.Services)

	// Stages 4-5: Per-record template + notify.
	for _, record := range multi.Records {
		result := base
		result.Status = record.Status
		result.Output = record.Lines
		result.Fields = record.Fields
		result.Arrays = record.Arrays

		// Global fields are base; record fields override on collision.
		mergedFields := mergeFields(multi.GlobalFields, record.Fields)
		mergedArrays := mergeArrays(multi.GlobalArrays, record.Arrays)

		log.Info("rendering templates", "status", record.Status)
		tmplData := notify.BuildTemplateData(
			r.cfg.Globals,
			alert.Name,
			mergedFields,
			mergedArrays,
			alert.Args,
		)

		targets, err := notify.ResolveTargets(refs, svcs, alert.Template, tmplData)
		if err != nil {
			result.Err = err
			result.ErrStage = "template"
			log.Error("template failed", "error", err)
			sendErr(result)
			return
		}

		result.Rendered = make(map[string]string, len(targets))
		for _, t := range targets {
			result.Rendered[t.ServiceName] = t.Message
		}
		log.Debug("templates rendered", "targets", len(targets))

		if cd != nil {
			shouldNotify, isRecovery := cd.Check(record.Status)
			result.Suppressed = !shouldNotify
			result.IsRecovery = isRecovery
			if !shouldNotify {
				log.Info("notification suppressed by cooldown", "status", record.Status)
				result.Duration = time.Since(start)
				out <- result
				continue
			}
		} else {
			if record.Status == "ok" {
				log.Info("status ok, skipping notifications")
				result.Duration = time.Since(start)
				out <- result
				continue
			}
		}

		for _, t := range targets {
			if dryRun {
				if err := notify.Validate(t); err != nil {
					result.Err = err
					result.ErrStage = "notify"
					log.Error("notify validation failed (dry-run)", "service", t.ServiceName, "error", err)
					sendErr(result)
					return
				}
				result.Notified = append(result.Notified, t.ServiceName)
				log.Debug("would notify (dry-run)", "service", t.ServiceName, "message", t.Message)
				continue
			}

			log.Info("sending notification", "service", t.ServiceName)
			if err := notify.Send(t); err != nil {
				result.Err = err
				result.ErrStage = "notify"
				log.Error("notify failed", "service", t.ServiceName, "error", err)
				sendErr(result)
				return
			}
			result.Notified = append(result.Notified, t.ServiceName)
			log.Debug("notification sent", "service", t.ServiceName)
		}

		result.Duration = time.Since(start)
		log.Info("alert completed", "status", result.Status, "duration", result.Duration)
		out <- result
	}
}

func mergeFields(global, record map[string]string) map[string]string {
	merged := make(map[string]string, len(global)+len(record))
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range record {
		merged[k] = v
	}
	return merged
}

func mergeArrays(global, record map[string]any) map[string]any {
	merged := make(map[string]any, len(global)+len(record))
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range record {
		merged[k] = v
	}
	return merged
}

func detectTriggerType(t config.Trigger) string {
	switch {
	case t.Lifecycle:
		return "lifecycle"
	case t.Pipe != "":
		return "pipe"
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
