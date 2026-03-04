package cooldown

import "time"

// Infinite is a sentinel duration meaning "suppress until ok resets the cycle".
const Infinite time.Duration = -1

// State tracks per-status cooldown for one alert.
// Warning and critical have independent timers.
type State struct {
	warning  timer
	critical timer
	recovery bool
	last     string // "" | "warning" | "critical" | "ok"
	now      func() time.Time
}

// New creates a State. Use Infinite for suppress-until-ok behaviour.
// now may be nil (defaults to time.Now).
func New(warning, critical time.Duration, recovery bool, now func() time.Time) *State {
	if now == nil {
		now = time.Now
	}
	return &State{
		warning:  timer{duration: warning},
		critical: timer{duration: critical},
		recovery: recovery,
		now:      now,
	}
}

// Check evaluates whether to notify and advances internal state.
// Returns (shouldNotify, isRecovery).
//
// For "warning"/"critical": notifies when cooldown is not active, suppresses otherwise.
// For "ok": sends a recovery notification if recovery=true and last status was an alert;
// always resets all timers on ok.
func (s *State) Check(status string) (notify bool, isRecovery bool) {
	now := s.now()
	switch status {
	case "warning", "critical":
		t := s.timerFor(status)
		if t.active(now) {
			return false, false
		}
		t.start(now)
		s.last = status
		return true, false

	case "ok":
		wasAlerted := s.last == "warning" || s.last == "critical"
		s.warning.reset()
		s.critical.reset()
		if !wasAlerted {
			return false, false
		}
		s.last = "ok"
		return s.recovery, s.recovery
	}

	return false, false
}

func (s *State) timerFor(status string) *timer {
	if status == "warning" {
		return &s.warning
	}
	return &s.critical
}

type timer struct {
	duration time.Duration
	started  bool
	expiry   time.Time
}

func (t *timer) active(now time.Time) bool {
	if !t.started {
		return false
	}
	if t.duration == Infinite {
		return true
	}
	return now.Before(t.expiry)
}

func (t *timer) start(now time.Time) {
	t.started = true
	if t.duration != Infinite {
		t.expiry = now.Add(t.duration)
	}
}

func (t *timer) reset() {
	t.started = false
	t.expiry = time.Time{}
}
