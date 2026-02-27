package runner

import "time"

// Result captures the outcome of running a single alert through the pipeline.
// Errors are stored in Err/ErrStage rather than returned, so the caller always
// has something to display.
type Result struct {
	AlertName string
	HealthcheckURI  string
	HealthcheckPath string
	Status    string            // "ok", "warning", "critical"
	Output    []string          // ordered KEY=VALUE lines
	Fields    map[string]string // parsed pairs
	Rendered  map[string]string // service name → rendered message
	Notified  []string          // services notified (or would-notify)
	DryRun    bool
	Duration  time.Duration
	Err       error
	ErrStage  string // "resolve", "exec", "parse", "template", "notify"
	Stderr    string
}
