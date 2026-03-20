package healthcheck

import (
	"fmt"
	"strings"
)

// ExecBuiltin returns synthetic ExecResult output for built-in healthchecks
// without spawning a process.
func ExecBuiltin(name string, params map[string]string) (*ExecResult, error) {
	switch name {
	case "lifecycle":
		return execLifecycle(params)
	case "ok":
		return &ExecResult{Stdout: "--- event\ntype=ok\n"}, nil
	default:
		return nil, fmt.Errorf("unknown builtin healthcheck: %s", name)
	}
}

func execLifecycle(params map[string]string) (*ExecResult, error) {
	event := params["event"]
	if event == "" {
		return nil, fmt.Errorf("builtin lifecycle: missing event param")
	}

	var b strings.Builder
	b.WriteString("--- event\n")
	b.WriteString("type=" + event + "\n")
	if alerts, ok := params["alerts"]; ok {
		b.WriteString("alerts=" + alerts + "\n")
	}

	return &ExecResult{Stdout: b.String()}, nil
}
