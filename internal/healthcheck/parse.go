package healthcheck

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsedOutput holds the parsed key-value output from a healthcheck.
type ParsedOutput struct {
	Status string
	Fields map[string]string
	Arrays map[string]any // typed slices: []string, []int64, or []bool
	Lines  []string
}

// MultiOutput holds the result of ParseMulti.
// GlobalFields/GlobalArrays are shared context emitted before "--- records".
// Records holds one ParsedOutput per "--- record" block.
// For single-record output (no separators), Global is empty and Records has one entry.
type MultiOutput struct {
	GlobalFields map[string]string
	GlobalArrays map[string]any
	Records      []*ParsedOutput
}

// Parse parses KEY=VALUE lines from healthcheck stdout.
// Lines without '=' are ignored. The "status" key is required.
// Values of the form [...] are parsed as typed arrays and stored in Arrays.
func Parse(stdout string) (*ParsedOutput, error) {
	out := &ParsedOutput{
		Fields: make(map[string]string),
		Arrays: make(map[string]any),
	}
	out.Lines = parseKeyValues(stdout, out.Fields, out.Arrays)

	status, ok := out.Fields["status"]
	if !ok {
		return nil, fmt.Errorf("healthcheck output missing required 'status' key")
	}
	out.Status = status

	return out, nil
}

// ParseMulti parses healthcheck stdout supporting the multi-record format.
//
// Structural tokens (exact line match after trimming):
//
//	"--- records" — ends the global props section, starts the records array
//	"--- record"  — starts the next record within the array
//
// Rules:
//   - No separators: single-record output. Equivalent to Parse; Records has one entry.
//   - "--- records" present: everything before it is global props; everything after
//     is records split by "--- record".
func ParseMulti(stdout string) (*MultiOutput, error) {
	const tokRecords = "--- records"
	const tokRecord = "--- record"

	lines := strings.Split(stdout, "\n")

	hasRecords := false
	for _, l := range lines {
		if strings.TrimSpace(l) == tokRecords {
			hasRecords = true
			break
		}
	}

	if !hasRecords {
		parsed, err := Parse(stdout)
		if err != nil {
			return nil, err
		}
		return &MultiOutput{
			GlobalFields: make(map[string]string),
			GlobalArrays: make(map[string]any),
			Records:      []*ParsedOutput{parsed},
		}, nil
	}

	out := &MultiOutput{
		GlobalFields: make(map[string]string),
		GlobalArrays: make(map[string]any),
	}

	var globalLines []string
	var recordBlocks [][]string
	var cur []string
	inRecords := false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		switch {
		case trimmed == tokRecords:
			inRecords = true
			cur = nil
		case trimmed == tokRecord:
			if inRecords {
				recordBlocks = append(recordBlocks, cur)
				cur = nil
			}
		default:
			if inRecords {
				cur = append(cur, l)
			} else {
				globalLines = append(globalLines, l)
			}
		}
	}
	if inRecords {
		recordBlocks = append(recordBlocks, cur)
	}

	parseKeyValues(strings.Join(globalLines, "\n"), out.GlobalFields, out.GlobalArrays)

	for i, block := range recordBlocks {
		parsed, err := Parse(strings.Join(block, "\n"))
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i, err)
		}
		out.Records = append(out.Records, parsed)
	}

	return out, nil
}

// parseKeyValues parses KEY=VALUE lines into fields and arrays.
// Returns the ordered Lines slice of "key=value" strings.
func parseKeyValues(text string, fields map[string]string, arrays map[string]any) []string {
	var lines []string
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

		lines = append(lines, key+"="+value)

		if len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']' {
			arrays[key] = parseArrayValue(value)
		} else {
			fields[key] = value
		}
	}
	return lines
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
