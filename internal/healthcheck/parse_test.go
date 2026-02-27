package healthcheck

import (
	"testing"
)

func TestParse_Valid(t *testing.T) {
	stdout := "status=warning\nusage=84\navailable=8G\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "warning" {
		t.Errorf("status = %q, want %q", out.Status, "warning")
	}
	if out.Fields["usage"] != "84" {
		t.Errorf("usage = %q, want %q", out.Fields["usage"], "84")
	}
	if out.Fields["available"] != "8G" {
		t.Errorf("available = %q, want %q", out.Fields["available"], "8G")
	}
	if len(out.Lines) != 3 {
		t.Errorf("lines = %d, want 3", len(out.Lines))
	}
}

func TestParse_MissingStatus(t *testing.T) {
	stdout := "usage=84\navailable=8G\n"
	_, err := Parse(stdout)
	if err == nil {
		t.Fatal("expected error for missing status")
	}
}

func TestParse_EmptyLines(t *testing.T) {
	stdout := "\nstatus=ok\n\nusage=10\n\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "ok" {
		t.Errorf("status = %q, want %q", out.Status, "ok")
	}
	if len(out.Lines) != 2 {
		t.Errorf("lines = %d, want 2", len(out.Lines))
	}
}

func TestParse_NoEqualsIgnored(t *testing.T) {
	stdout := "status=ok\nsome random line\nusage=5\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Lines) != 2 {
		t.Errorf("lines = %d, want 2", len(out.Lines))
	}
}

func TestParse_ValueWithEquals(t *testing.T) {
	stdout := "status=ok\nmessage=a=b=c\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Fields["message"] != "a=b=c" {
		t.Errorf("message = %q, want %q", out.Fields["message"], "a=b=c")
	}
}

func TestParse_WhitespaceHandling(t *testing.T) {
	stdout := "  status = ok  \n  usage = 42  \n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "ok" {
		t.Errorf("status = %q, want %q", out.Status, "ok")
	}
	if out.Fields["usage"] != "42" {
		t.Errorf("usage = %q, want %q", out.Fields["usage"], "42")
	}
}
