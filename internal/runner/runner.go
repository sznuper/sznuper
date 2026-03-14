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

// Runner orchestrates the healthcheck -> parse -> template -> notify pipeline.
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

// AlertState tracks the healthy/unhealthy binary state for an alert.
// Used only when events.healthy is configured.
type AlertState struct {
	Healthy bool
}

// RunOpts holds optional parameters for RunAlertOpts.
type RunOpts struct {
	DryRun        bool
	Cooldown      *cooldown.State
	State         *AlertState // state machine (nil = no state tracking)
	Stdin         []byte
	BuiltinParams map[string]string // params for builtin:// healthchecks
}

// RunAlert executes a single alert through the full pipeline asynchronously.
// It returns a channel that yields one Result per event, then closes.
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

	// Stage 3: Parse events.
	log.Info("parsing output")
	events, err := healthcheck.ParseEvents(execResult.Stdout)
	if err != nil {
		base.Err = err
		base.ErrStage = "parse"
		log.Error("parse failed", "error", err, "stdout", execResult.Stdout)
		sendErr(base)
		return
	}
	log.Debug("output parsed", "events", len(events))

	svcs := mapServiceDefs(r.cfg.Services)

	// Stage 4: Process each event.
	for _, ev := range events {
		result := base
		result.EventType = ev.Type
		result.Fields = ev.Fields
		result.Arrays = ev.Arrays

		// a. Resolve config: find matching override or apply on_unmatched rule.
		var override *config.EventOverride
		if alert.Events != nil {
			if o, ok := alert.Events.Override[ev.Type]; ok {
				override = &o
			}
		}

		dropped := override == nil && alert.Events != nil && alert.Events.OnUnmatched == "drop"

		// b. State machine.
		skipNotify := false
		if opts.State != nil {
			isHealthyEv := isHealthyEvent(alert, ev.Type)
			if isHealthyEv {
				if opts.State.Healthy {
					// healthy -> healthy: no notification
					log.Info("healthy event in healthy state, skipping", "type", ev.Type)
					skipNotify = true
				} else {
					// unhealthy -> healthy: recovery
					opts.State.Healthy = true
					result.IsRecovery = true
					if opts.Cooldown != nil {
						opts.Cooldown.ResetAll()
					}
					log.Info("recovery transition", "type", ev.Type)
					if dropped {
						log.Info("recovery event dropped by on_unmatched", "type", ev.Type)
						skipNotify = true
					}
				}
			} else {
				// unhealthy event
				if opts.State.Healthy {
					opts.State.Healthy = false
					log.Info("unhealthy transition", "type", ev.Type)
				}
				if dropped {
					log.Info("event dropped by on_unmatched", "type", ev.Type)
					skipNotify = true
				}
			}
		} else if dropped {
			log.Info("event dropped by on_unmatched", "type", ev.Type)
			skipNotify = true
		}

		if skipNotify {
			result.Dropped = dropped
			result.Duration = time.Since(start)
			out <- result
			continue
		}

		// c. Cooldown.
		effectiveDuration := resolveEffectiveCooldown(alert, override)
		if opts.Cooldown != nil {
			if !opts.Cooldown.Check(ev.Type, effectiveDuration) {
				log.Info("notification suppressed by cooldown", "type", ev.Type)
				result.Suppressed = true
				result.Duration = time.Since(start)
				out <- result
				continue
			}
		}

		// d. Template.
		effectiveTemplate := alert.Template
		if override != nil && override.Template != "" {
			effectiveTemplate = override.Template
		}

		log.Info("rendering templates", "type", ev.Type)
		tmplData := notify.BuildTemplateData(
			r.cfg.Globals,
			alert.Name,
			ev.Fields,
			ev.Arrays,
			alert.Args,
		)

		effectiveNotify := alert.Notify
		if override != nil && len(override.Notify) > 0 {
			effectiveNotify = override.Notify
		}
		refs := mapNotifyRefs(effectiveNotify)

		targets, err := notify.ResolveTargets(refs, svcs, effectiveTemplate, tmplData)
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

		// e. Notify.
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
		log.Info("event processed", "type", result.EventType, "duration", result.Duration)
		out <- result
	}
}

// isHealthyEvent returns true if the event type is in the alert's healthy list.
func isHealthyEvent(alert *config.Alert, eventType string) bool {
	if alert.Events == nil {
		return false
	}
	for _, h := range alert.Events.Healthy {
		if h == eventType {
			return true
		}
	}
	return false
}

// resolveEffectiveCooldown returns the cooldown duration for an event type.
// Override cooldown takes precedence over alert-level cooldown.
func resolveEffectiveCooldown(alert *config.Alert, override *config.EventOverride) time.Duration {
	cd := alert.Cooldown
	if override != nil && override.Cooldown != "" {
		cd = override.Cooldown
	}
	return parseCooldownValue(cd)
}

func parseCooldownValue(s string) time.Duration {
	if s == "" {
		return 0
	}
	if s == "inf" {
		return cooldown.Infinite
	}
	d, _ := time.ParseDuration(s)
	return d
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
