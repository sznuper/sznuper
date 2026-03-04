package cooldown

import (
	"testing"
	"time"
)

var t0 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// newState returns a State with an injectable clock pointer.
func newState(warn, crit time.Duration, recovery bool) (*State, *time.Time) {
	now := t0
	s := New(warn, crit, recovery, func() time.Time { return now })
	return s, &now
}

func TestCheck_FirstWarningNotifies(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, false)
	notify, isRecovery := s.Check("warning")
	if !notify {
		t.Error("want notify=true")
	}
	if isRecovery {
		t.Error("want isRecovery=false")
	}
}

func TestCheck_SecondWarningWithinCooldownSuppressed(t *testing.T) {
	s, now := newState(5*time.Minute, 10*time.Minute, false)
	s.Check("warning")
	*now = t0.Add(2 * time.Minute) // still within 5m cooldown
	notify, _ := s.Check("warning")
	if notify {
		t.Error("want notify=false (suppressed)")
	}
}

func TestCheck_WarningAfterCooldownExpiresNotifies(t *testing.T) {
	s, now := newState(5*time.Minute, 10*time.Minute, false)
	s.Check("warning")
	*now = t0.Add(6 * time.Minute) // past 5m cooldown
	notify, _ := s.Check("warning")
	if !notify {
		t.Error("want notify=true (cooldown expired)")
	}
}

func TestCheck_CriticalIndependentWhileWarnCooldownActive(t *testing.T) {
	s, now := newState(5*time.Minute, 10*time.Minute, false)
	s.Check("warning") // fires, starts warning timer
	*now = t0.Add(1 * time.Minute)

	// critical has its own timer — should fire
	notifyC, _ := s.Check("critical")
	if !notifyC {
		t.Error("want critical notify=true (independent timer)")
	}

	// warning still suppressed
	notifyW, _ := s.Check("warning")
	if notifyW {
		t.Error("want warning notify=false (still in cooldown)")
	}
}

func TestCheck_OkAfterWarningWithRecoveryNotifies(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, true)
	s.Check("warning")
	notify, isRecovery := s.Check("ok")
	if !notify {
		t.Error("want notify=true")
	}
	if !isRecovery {
		t.Error("want isRecovery=true")
	}
}

func TestCheck_OkAfterWarningWithoutRecoverySuppressed(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, false)
	s.Check("warning")
	notify, isRecovery := s.Check("ok")
	if notify {
		t.Error("want notify=false")
	}
	if isRecovery {
		t.Error("want isRecovery=false")
	}
}

func TestCheck_OkWithNoAlertHistoryNoop(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, true)
	notify, isRecovery := s.Check("ok")
	if notify {
		t.Error("want notify=false (no prior alert)")
	}
	if isRecovery {
		t.Error("want isRecovery=false")
	}
}

func TestCheck_OkAfterOkNoop(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, true)
	s.Check("warning")
	s.Check("ok") // recovery fires, last="ok"
	notify, isRecovery := s.Check("ok")
	if notify {
		t.Error("want notify=false (already recovered)")
	}
	if isRecovery {
		t.Error("want isRecovery=false")
	}
}

func TestCheck_AfterRecoveryNewWarningFiresFresh(t *testing.T) {
	s, _ := newState(5*time.Minute, 10*time.Minute, true)
	s.Check("warning")
	s.Check("ok") // recovery resets timers
	notify, _ := s.Check("warning")
	if !notify {
		t.Error("want notify=true (timers were reset on ok)")
	}
}

func TestCheck_InfiniteWarningNeverExpires(t *testing.T) {
	s, now := newState(Infinite, 10*time.Minute, false)
	s.Check("warning")
	*now = t0.Add(24 * time.Hour)
	notify, _ := s.Check("warning")
	if notify {
		t.Error("want notify=false (infinite cooldown never expires)")
	}
}

func TestCheck_InfiniteResetsOnOk(t *testing.T) {
	s, now := newState(Infinite, 10*time.Minute, true)
	s.Check("warning")
	*now = t0.Add(24 * time.Hour)
	s.Check("ok") // recovery fires, resets infinite timer
	notify, _ := s.Check("warning")
	if !notify {
		t.Error("want notify=true (infinite timer reset by ok)")
	}
}

func TestCheck_SimpleShorthand_BothStatusesUseSameDuration(t *testing.T) {
	// Simulates simple cooldown: same duration for both.
	s, now := newState(5*time.Minute, 5*time.Minute, false)
	s.Check("warning")
	s.Check("critical") // both fire (independent timers)

	*now = t0.Add(2 * time.Minute) // within 5m cooldown for both
	notifyW, _ := s.Check("warning")
	notifyC, _ := s.Check("critical")
	if notifyW {
		t.Error("want warning suppressed")
	}
	if notifyC {
		t.Error("want critical suppressed")
	}

	*now = t0.Add(6 * time.Minute) // past 5m for both
	notifyW, _ = s.Check("warning")
	notifyC, _ = s.Check("critical")
	if !notifyW {
		t.Error("want warning notify after cooldown")
	}
	if !notifyC {
		t.Error("want critical notify after cooldown")
	}
}
