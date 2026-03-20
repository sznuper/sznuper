package healthcheck

import "testing"

func TestExecBuiltin_Ok(t *testing.T) {
	result, err := ExecBuiltin("ok", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "--- event\ntype=ok\n"
	if result.Stdout != want {
		t.Errorf("stdout = %q, want %q", result.Stdout, want)
	}
}

func TestExecBuiltin_Unknown(t *testing.T) {
	_, err := ExecBuiltin("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown builtin")
	}
}
