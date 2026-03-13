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
	b.WriteString("status=warning\n")
	b.WriteString("event=" + event + "\n")
	if alerts, ok := params["alerts"]; ok {
		b.WriteString("alerts=" + alerts + "\n")
	}

	return &ExecResult{Stdout: b.String()}, nil
}
