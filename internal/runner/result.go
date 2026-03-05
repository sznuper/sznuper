package runner

import "time"

// Result captures the outcome of running a single alert through the pipeline.
// Errors are stored in Err/ErrStage rather than returned, so the caller always
// has something to display.
type Result struct {
	AlertName       string
	HealthcheckURI  string
	HealthcheckPath string
	Status          string            // "ok", "warning", "critical"
	Output          []string          // ordered KEY=VALUE lines
	Fields          map[string]string // parsed scalar pairs
	Arrays          map[string]any    // parsed array fields ([]string, []int64, or []bool)
	Rendered        map[string]string // service name → rendered message
	Notified        []string          // services notified (or would-notify)
	Env             []string
	DryRun          bool
	Suppressed      bool // notification suppressed by cooldown
	IsRecovery      bool // ok-after-alert recovery notification
	Duration        time.Duration
	Err             error
	ErrStage        string // "resolve", "exec", "parse", "template", "notify"
	Stderr          string
}
