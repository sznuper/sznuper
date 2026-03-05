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

// Parse parses KEY=VALUE lines from healthcheck stdout.
// Lines without '=' are ignored. The "status" key is required.
// Values of the form [...] are parsed as typed arrays and stored in Arrays.
func Parse(stdout string) (*ParsedOutput, error) {
	out := &ParsedOutput{
		Fields: make(map[string]string),
		Arrays: make(map[string]any),
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

		out.Lines = append(out.Lines, key+"="+value)

		if len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']' {
			out.Arrays[key] = parseArrayValue(value)
		} else {
			out.Fields[key] = value
		}
	}

	status, ok := out.Fields["status"]
	if !ok {
		return nil, fmt.Errorf("healthcheck output missing required 'status' key")
	}
	out.Status = status

	return out, nil
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
