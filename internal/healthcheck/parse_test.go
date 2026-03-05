package healthcheck

import (
	"reflect"
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

func TestParse_StringArray(t *testing.T) {
	stdout := `status=ok
hosts=["1.2.3.4", "5.6.7.8"]
`
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, inFields := out.Fields["hosts"]; inFields {
		t.Error("array key should not be in Fields")
	}
	got, ok := out.Arrays["hosts"].([]string)
	if !ok {
		t.Fatalf("hosts not []string, got %T", out.Arrays["hosts"])
	}
	want := []string{"1.2.3.4", "5.6.7.8"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("hosts = %v, want %v", got, want)
	}
}

func TestParse_IntArray(t *testing.T) {
	stdout := "status=ok\ncounts=[1, 2, 3]\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := out.Arrays["counts"].([]int64)
	if !ok {
		t.Fatalf("counts not []int64, got %T", out.Arrays["counts"])
	}
	want := []int64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}
}

func TestParse_BoolArray(t *testing.T) {
	stdout := "status=ok\nflags=[true, false, true]\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := out.Arrays["flags"].([]bool)
	if !ok {
		t.Fatalf("flags not []bool, got %T", out.Arrays["flags"])
	}
	want := []bool{true, false, true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flags = %v, want %v", got, want)
	}
}

func TestParse_EmptyArray(t *testing.T) {
	stdout := "status=ok\narr=[]\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := out.Arrays["arr"].([]string)
	if !ok {
		t.Fatalf("arr not []string, got %T", out.Arrays["arr"])
	}
	if len(got) != 0 {
		t.Errorf("arr = %v, want empty", got)
	}
}

func TestParse_SingleElementArray(t *testing.T) {
	stdout := "status=ok\ncounts=[42]\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := out.Arrays["counts"].([]int64)
	if !ok {
		t.Fatalf("counts not []int64, got %T", out.Arrays["counts"])
	}
	if !reflect.DeepEqual(got, []int64{42}) {
		t.Errorf("counts = %v, want [42]", got)
	}
}

func TestParse_ArrayInLines(t *testing.T) {
	stdout := "status=ok\nhosts=[\"a\", \"b\"]\n"
	out, err := Parse(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Array lines are included in Lines with raw value
	found := false
	for _, l := range out.Lines {
		if l == `hosts=["a", "b"]` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("array line not found in Lines: %v", out.Lines)
	}
}
