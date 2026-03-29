package runner

import "time"

// Result captures the outcome of running a single alert event through the pipeline.
type Result struct {
	AlertName       string
	HealthcheckURI  string
	HealthcheckPath string
	EventType       string            // the event's type field
	Fields          map[string]string // parsed scalar pairs
	Rendered        map[string]string // channel name -> rendered message
	Notified        []string          // channels notified (or would-notify)
	Env             []string
	DryRun          bool
	Suppressed      bool // notification suppressed by cooldown
	IsRecovery      bool // recovery notification (unhealthy->healthy)
	Dropped         bool // event dropped by on_unmatched: drop
	SideEffectsRun  int
	Duration        time.Duration
	Err             error
	ErrStage        string // "resolve", "exec", "parse", "template", "notify"
	Stderr          string
}
