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

func TestParseMulti_SingleRecord(t *testing.T) {
	stdout := "status=warning\nusage=84\n"
	out, err := ParseMulti(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(out.Records))
	}
	if out.Records[0].Status != "warning" {
		t.Errorf("status = %q, want %q", out.Records[0].Status, "warning")
	}
	if len(out.GlobalFields) != 0 {
		t.Errorf("GlobalFields should be empty for single-record output")
	}
}

func TestParseMulti_MultiRecord(t *testing.T) {
	stdout := "event_count=2\nfailure_count=1\nlogin_count=1\n--- records\nstatus=warning\nevent=failure\nuser=root\nhost=1.2.3.4\n--- record\nstatus=ok\nevent=login\nuser=bob\nhost=5.6.7.8\n"
	out, err := ParseMulti(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Records) != 2 {
		t.Fatalf("records = %d, want 2", len(out.Records))
	}
	if out.GlobalFields["event_count"] != "2" {
		t.Errorf("GlobalFields[event_count] = %q, want %q", out.GlobalFields["event_count"], "2")
	}
	if out.Records[0].Status != "warning" {
		t.Errorf("record[0] status = %q, want %q", out.Records[0].Status, "warning")
	}
	if out.Records[0].Fields["user"] != "root" {
		t.Errorf("record[0] user = %q, want %q", out.Records[0].Fields["user"], "root")
	}
	if out.Records[1].Status != "ok" {
		t.Errorf("record[1] status = %q, want %q", out.Records[1].Status, "ok")
	}
	if out.Records[1].Fields["host"] != "5.6.7.8" {
		t.Errorf("record[1] host = %q, want %q", out.Records[1].Fields["host"], "5.6.7.8")
	}
}

func TestParseMulti_EmptyRecords(t *testing.T) {
	// "--- records" with nothing after it → 1 empty block, parse fails on missing status
	// But with just global props and no events, the C code won't emit "--- records" at all.
	// Test: no "--- records" and no fields → error on missing status from Parse.
	stdout := "status=ok\nevent_count=0\n"
	out, err := ParseMulti(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(out.Records))
	}
	if out.Records[0].Status != "ok" {
		t.Errorf("status = %q, want %q", out.Records[0].Status, "ok")
	}
}

func TestParseMulti_GlobalFieldsNotInRecords(t *testing.T) {
	stdout := "batch=42\n--- records\nstatus=warning\nevent=failure\n"
	out, err := ParseMulti(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.GlobalFields["batch"] != "42" {
		t.Errorf("GlobalFields[batch] = %q, want %q", out.GlobalFields["batch"], "42")
	}
	if _, ok := out.Records[0].Fields["batch"]; ok {
		t.Error("record should not contain global field 'batch'")
	}
}

func TestParseMulti_RecordMissingStatus(t *testing.T) {
	stdout := "global=1\n--- records\nevent=failure\nuser=root\n"
	_, err := ParseMulti(stdout)
	if err == nil {
		t.Fatal("expected error for record missing status")
	}
}

func TestParseMulti_OneRecordNoSeparator(t *testing.T) {
	// "--- records" with single block (no "--- record") → exactly one record
	stdout := "ctx=1\n--- records\nstatus=ok\nfoo=bar\n"
	out, err := ParseMulti(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(out.Records))
	}
	if out.Records[0].Fields["foo"] != "bar" {
		t.Errorf("foo = %q, want %q", out.Records[0].Fields["foo"], "bar")
	}
}

func TestParseMulti_ArrayInLines(t *testing.T) {
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
