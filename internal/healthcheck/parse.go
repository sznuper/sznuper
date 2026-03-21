package healthcheck

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
)

// Event holds a single parsed event from healthcheck output.
type Event struct {
	Type   string
	Fields map[string]string
	Raw    string // original block text (without "--- event" delimiter)
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
		raw := strings.Join(block, "\n")

		// Filter to only KEY=VALUE lines — godotenv rejects non-KV lines.
		var kvLines []string
		for _, line := range block {
			if strings.TrimSpace(line) != "" && strings.Contains(line, "=") {
				kvLines = append(kvLines, line)
			}
		}

		parsed, err := godotenv.Unmarshal(strings.Join(kvLines, "\n"))
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}

		fields := make(map[string]string, len(parsed))
		for k, v := range parsed {
			fields[strings.ToLower(k)] = v
		}

		ev := Event{
			Fields: fields,
			Raw:    raw,
		}

		typ, ok := ev.Fields["type"]
		if !ok {
			return nil, fmt.Errorf("event %d: missing required 'type' field", i)
		}
		ev.Type = typ

		events = append(events, ev)
	}

	return events, nil
}
