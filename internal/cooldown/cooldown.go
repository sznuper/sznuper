package cooldown

import "time"

// Infinite is a sentinel duration meaning "suppress until cooldowns are reset".
const Infinite time.Duration = -1

// State tracks per-event-type cooldown for one alert.
type State struct {
	timers map[string]*timer
	now    func() time.Time
}

// New creates a State. now may be nil (defaults to time.Now).
func New(now func() time.Time) *State {
	if now == nil {
		now = time.Now
	}
	return &State{
		timers: make(map[string]*timer),
		now:    now,
	}
}

// Check evaluates whether to notify for the given event type and advances
// internal state. Returns true if the notification should fire.
//
// A duration of 0 means no cooldown — always notifies.
// Infinite means suppress until ResetAll is called.
func (s *State) Check(eventType string, duration time.Duration) bool {
	if duration == 0 {
		return true
	}

	now := s.now()
	t, ok := s.timers[eventType]
	if !ok {
		t = &timer{}
		s.timers[eventType] = t
	}

	if t.active(now) {
		return false
	}
	t.start(now, duration)
	return true
}

// ResetAll clears all cooldown timers.
// Called on recovery (unhealthy→healthy transition).
func (s *State) ResetAll() {
	for _, t := range s.timers {
		t.reset()
	}
}

type timer struct {
	started  bool
	duration time.Duration
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

func (t *timer) start(now time.Time, duration time.Duration) {
	t.started = true
	t.duration = duration
	if duration != Infinite {
		t.expiry = now.Add(duration)
	}
}

func (t *timer) reset() {
	t.started = false
	t.duration = 0
	t.expiry = time.Time{}
}
