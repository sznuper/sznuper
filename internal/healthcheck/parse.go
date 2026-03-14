package healthcheck

import (
	"fmt"
	"strconv"
	"strings"
)

// Event holds a single parsed event from healthcheck output.
type Event struct {
	Type   string
	Fields map[string]string
	Arrays map[string]any // typed slices: []string, []int64, or []bool
}

// ParseEvents parses healthcheck stdout into a list of events.
//
// Each event block starts with "--- event" on its own line, followed by
// key=value pairs. The "type" field is required in every event.
// Lines before the first "--- event" are ignored.
// Empty output (no "--- event" markers) returns zero events.
func ParseEvents(stdout string) ([]Event, error) {
	const tok = "--- event"

	lines := strings.Split(stdout, "\n")

	// Split into event blocks.
	var blocks [][]string
	var cur []string
	inEvent := false

	for _, l := range lines {
		if strings.TrimSpace(l) == tok {
			if inEvent {
				blocks = append(blocks, cur)
			}
			cur = nil
			inEvent = true
			continue
		}
		if inEvent {
			cur = append(cur, l)
		}
	}
	if inEvent {
		blocks = append(blocks, cur)
	}

	events := make([]Event, 0, len(blocks))
	for i, block := range blocks {
		ev := Event{
			Fields: make(map[string]string),
			Arrays: make(map[string]any),
		}
		parseKeyValues(strings.Join(block, "\n"), ev.Fields, ev.Arrays)

		typ, ok := ev.Fields["type"]
		if !ok {
			return nil, fmt.Errorf("event %d: missing required 'type' field", i)
		}
		ev.Type = typ

		events = append(events, ev)
	}

	return events, nil
}

// parseKeyValues parses KEY=VALUE lines into fields and arrays.
func parseKeyValues(text string, fields map[string]string, arrays map[string]any) {
	for _, line := range strings.Split(text, "\n") {
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

		if len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']' {
			arrays[key] = parseArrayValue(value)
		} else {
			fields[key] = value
		}
	}
}

func parseArrayValue(raw string) any {
	inner := strings.TrimSpace(raw[1 : len(raw)-1])
	if inner == "" {
		return []string{}
	}
	switch {
	case inner[0] == '"':
		return parseStringArray(inner)
	case strings.HasPrefix(inner, "true") || strings.HasPrefix(inner, "false"):
		return parseBoolArray(inner)
	default:
		return parseIntArray(inner)
	}
}

// splitCommaTokens splits a comma-separated string into trimmed tokens,
// stripping surrounding double-quotes from each token.
func splitCommaTokens(s string) []string {
	parts := strings.Split(s, ",")
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
			p = p[1 : len(p)-1]
		}
		tokens = append(tokens, p)
	}
	return tokens
}

func parseStringArray(inner string) []string {
	return splitCommaTokens(inner)
}

func parseBoolArray(inner string) []bool {
	tokens := splitCommaTokens(inner)
	result := make([]bool, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, t == "true")
	}
	return result
}

func parseIntArray(inner string) []int64 {
	tokens := splitCommaTokens(inner)
	result := make([]int64, 0, len(tokens))
	for _, t := range tokens {
		v, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			result = append(result, v)
		}
	}
	return result
}
