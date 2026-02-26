package check

import (
	"fmt"
	"strings"
)

// ParsedOutput holds the parsed key-value output from a check.
type ParsedOutput struct {
	Status string
	Fields map[string]string
	Lines  []string
}

// Parse parses KEY=VALUE lines from check stdout.
// Lines without '=' are ignored. The "status" key is required.
func Parse(stdout string) (*ParsedOutput, error) {
	out := &ParsedOutput{
		Fields: make(map[string]string),
	}

	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		out.Fields[key] = value
		out.Lines = append(out.Lines, key+"="+value)
	}

	status, ok := out.Fields["status"]
	if !ok {
		return nil, fmt.Errorf("check output missing required 'status' key")
	}
	out.Status = status

	return out, nil
}
