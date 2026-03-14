package cooldown

import (
	"testing"
	"time"
)

var t0 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// newState returns a State with an injectable clock pointer.
func newState() (*State, *time.Time) {
	now := t0
	s := New(func() time.Time { return now })
	return s, &now
}

func TestCheck_FirstEventNotifies(t *testing.T) {
	s, _ := newState()
	if !s.Check("failure", 5*time.Minute) {
		t.Error("want notify=true")
	}
}

func TestCheck_SecondEventWithinCooldownSuppressed(t *testing.T) {
	s, now := newState()
	s.Check("failure", 5*time.Minute)
	*now = t0.Add(2 * time.Minute) // still within 5m cooldown
	if s.Check("failure", 5*time.Minute) {
		t.Error("want notify=false (suppressed)")
	}
}

func TestCheck_EventAfterCooldownExpiresNotifies(t *testing.T) {
	s, now := newState()
	s.Check("failure", 5*time.Minute)
	*now = t0.Add(6 * time.Minute) // past 5m cooldown
	if !s.Check("failure", 5*time.Minute) {
		t.Error("want notify=true (cooldown expired)")
	}
}

func TestCheck_DifferentTypesIndependent(t *testing.T) {
	s, now := newState()
	s.Check("failure", 5*time.Minute) // fires, starts failure timer
	*now = t0.Add(1 * time.Minute)

	// login has its own timer — should fire
	if !s.Check("login", 10*time.Minute) {
		t.Error("want login notify=true (independent timer)")
	}

	// failure still suppressed
	if s.Check("failure", 5*time.Minute) {
		t.Error("want failure notify=false (still in cooldown)")
	}
}

func TestCheck_ZeroDurationAlwaysNotifies(t *testing.T) {
	s, _ := newState()
	if !s.Check("failure", 0) {
		t.Error("want notify=true (zero duration)")
	}
	if !s.Check("failure", 0) {
		t.Error("want notify=true (zero duration, second call)")
	}
}

func TestCheck_InfiniteNeverExpires(t *testing.T) {
	s, now := newState()
	s.Check("failure", Infinite)
	*now = t0.Add(24 * time.Hour)
	if s.Check("failure", Infinite) {
		t.Error("want notify=false (infinite cooldown never expires)")
	}
}

func TestResetAll_ClearsAllTimers(t *testing.T) {
	s, _ := newState()
	s.Check("failure", 5*time.Minute)
	s.Check("login", 10*time.Minute)
	s.ResetAll()
	// Both should fire again after reset
	if !s.Check("failure", 5*time.Minute) {
		t.Error("want failure notify=true (after reset)")
	}
	if !s.Check("login", 10*time.Minute) {
		t.Error("want login notify=true (after reset)")
	}
}

func TestResetAll_InfiniteResetsOnReset(t *testing.T) {
	s, now := newState()
	s.Check("failure", Infinite)
	*now = t0.Add(24 * time.Hour)
	s.ResetAll()
	if !s.Check("failure", Infinite) {
		t.Error("want notify=true (infinite timer reset)")
	}
}

func TestCheck_SameDurationForAllTypes(t *testing.T) {
	s, now := newState()
	s.Check("failure", 5*time.Minute)
	s.Check("login", 5*time.Minute) // both fire (independent timers)

	*now = t0.Add(2 * time.Minute) // within 5m cooldown for both
	if s.Check("failure", 5*time.Minute) {
		t.Error("want failure suppressed")
	}
	if s.Check("login", 5*time.Minute) {
		t.Error("want login suppressed")
	}

	*now = t0.Add(6 * time.Minute) // past 5m for both
	if !s.Check("failure", 5*time.Minute) {
		t.Error("want failure notify after cooldown")
	}
	if !s.Check("login", 5*time.Minute) {
		t.Error("want login notify after cooldown")
	}
}
