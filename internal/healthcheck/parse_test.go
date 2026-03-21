package healthcheck

import (
	"testing"
)

func TestParseEvents_SingleEvent(t *testing.T) {
	stdout := "--- event\ntype=failure\nuser=root\nhost=1.2.3.4\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != "failure" {
		t.Errorf("type = %q, want %q", events[0].Type, "failure")
	}
	if events[0].Fields["user"] != "root" {
		t.Errorf("user = %q, want %q", events[0].Fields["user"], "root")
	}
	if events[0].Fields["host"] != "1.2.3.4" {
		t.Errorf("host = %q, want %q", events[0].Fields["host"], "1.2.3.4")
	}
}

func TestParseEvents_MultipleEvents(t *testing.T) {
	stdout := "--- event\ntype=failure\nuser=root\nhost=1.2.3.4\n--- event\ntype=login\nuser=niar\nhost=10.0.0.1\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].Type != "failure" {
		t.Errorf("event[0] type = %q, want %q", events[0].Type, "failure")
	}
	if events[1].Type != "login" {
		t.Errorf("event[1] type = %q, want %q", events[1].Type, "login")
	}
	if events[1].Fields["user"] != "niar" {
		t.Errorf("event[1] user = %q, want %q", events[1].Fields["user"], "niar")
	}
}

func TestParseEvents_EmptyOutput(t *testing.T) {
	events, err := ParseEvents("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
}

func TestParseEvents_NoEventDelimiters(t *testing.T) {
	stdout := "some random output\nkey=value\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0 (no --- event markers)", len(events))
	}
}

func TestParseEvents_MissingType(t *testing.T) {
	stdout := "--- event\nuser=root\nhost=1.2.3.4\n"
	_, err := ParseEvents(stdout)
	if err == nil {
		t.Fatal("expected error for missing type field")
	}
}

func TestParseEvents_LinesBeforeFirstEvent(t *testing.T) {
	stdout := "some preamble\ndebug output\n--- event\ntype=ok\nusage=42\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != "ok" {
		t.Errorf("type = %q, want %q", events[0].Type, "ok")
	}
}

func TestParseEvents_NonKVLinesIgnored(t *testing.T) {
	stdout := "--- event\ntype=ok\nsome random line\nusage=5\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Fields["usage"] != "5" {
		t.Errorf("usage = %q, want %q", events[0].Fields["usage"], "5")
	}
}

func TestParseEvents_ValueWithEquals(t *testing.T) {
	stdout := "--- event\ntype=ok\nmessage=a=b=c\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].Fields["message"] != "a=b=c" {
		t.Errorf("message = %q, want %q", events[0].Fields["message"], "a=b=c")
	}
}

func TestParseEvents_WhitespaceHandling(t *testing.T) {
	stdout := "  --- event  \ntype=ok\nusage=42\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != "ok" {
		t.Errorf("type = %q, want %q", events[0].Type, "ok")
	}
	if events[0].Fields["usage"] != "42" {
		t.Errorf("usage = %q, want %q", events[0].Fields["usage"], "42")
	}
}

func TestParseEvents_RawPreservesBlockText(t *testing.T) {
	stdout := "--- event\ntype=failure\nuser=root\nhost=1.2.3.4\n--- event\ntype=ok\nusage=42\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	want0 := "type=failure\nuser=root\nhost=1.2.3.4"
	if events[0].Raw != want0 {
		t.Errorf("events[0].Raw = %q, want %q", events[0].Raw, want0)
	}
	want1 := "type=ok\nusage=42\n"
	if events[1].Raw != want1 {
		t.Errorf("events[1].Raw = %q, want %q", events[1].Raw, want1)
	}
}

func TestParseEvents_ArrayValuesAsStrings(t *testing.T) {
	stdout := "--- event\ntype=ok\nhosts='[\"1.2.3.4\", \"5.6.7.8\"]'\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := events[0].Fields["hosts"]
	want := `["1.2.3.4", "5.6.7.8"]`
	if got != want {
		t.Errorf("hosts = %q, want %q", got, want)
	}
}

func TestParseEvents_KeysLowercased(t *testing.T) {
	stdout := "--- event\nTYPE=ok\nUSAGE_PERCENT=84\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].Type != "ok" {
		t.Errorf("type = %q, want %q", events[0].Type, "ok")
	}
	if events[0].Fields["usage_percent"] != "84" {
		t.Errorf("usage_percent = %q, want %q", events[0].Fields["usage_percent"], "84")
	}
}

func TestParseEvents_QuotedValues(t *testing.T) {
	stdout := "--- event\ntype=ok\nmessage=\"hello world\"\n"
	events, err := ParseEvents(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].Fields["message"] != "hello world" {
		t.Errorf("message = %q, want %q", events[0].Fields["message"], "hello world")
	}
}
