package runner

import "time"

// Result captures the outcome of running a single alert event through the pipeline.
type Result struct {
	AlertName       string
	HealthcheckURI  string
	HealthcheckPath string
	EventType       string            // the event's type field
	Fields          map[string]string // parsed scalar pairs
	Arrays          map[string]any    // parsed array fields ([]string, []int64, or []bool)
	Rendered        map[string]string // service name -> rendered message
	Notified        []string          // services notified (or would-notify)
	Env             []string
	DryRun          bool
	Suppressed      bool // notification suppressed by cooldown
	IsRecovery      bool // recovery notification (unhealthy->healthy)
	Dropped         bool // event dropped by on_unmatched: drop
	Duration        time.Duration
	Err             error
	ErrStage        string // "resolve", "exec", "parse", "template", "notify"
	Stderr          string
}
