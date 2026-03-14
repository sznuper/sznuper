package notify

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TemplateData holds all data available to notification templates.
type TemplateData struct {
	Globals map[string]any
	Alert   map[string]string
	Event   map[string]any // scalar fields (string) and array fields ([]string, []int64, []bool)
	Args    map[string]string
}

// BuildTemplateData constructs template data from event output and config.
func BuildTemplateData(globals map[string]any, alertName string, eventFields map[string]string, eventArrays map[string]any, args map[string]any) TemplateData {
	alert := map[string]string{
		"name": alertName,
	}

	// Merge scalar fields and array fields into a single map[string]any.
	ev := make(map[string]any, len(eventFields)+len(eventArrays))
	for k, v := range eventFields {
		ev[k] = v
	}
	for k, v := range eventArrays {
		ev[k] = v
	}

	// Convert args to string map.
	argsStr := make(map[string]string, len(args))
	for k, v := range args {
		argsStr[k] = fmt.Sprint(v)
	}

	return TemplateData{
		Globals: globals,
		Alert:   alert,
		Event:   ev,
		Args:    argsStr,
	}
}

// Render executes a Go text/template string with Sprig functions and the
// custom accessor functions (event, globals, alert, args).
func Render(tmplStr string, data TemplateData) (string, error) {
	funcMap := sprig.TxtFuncMap()

	// Register accessor functions so {{event.type}} works:
	// "event" returns the event map, then ".type" accesses a key.
	funcMap["event"] = func() map[string]any { return data.Event }
	funcMap["globals"] = func() map[string]any { return data.Globals }
	funcMap["alert"] = func() map[string]string { return data.Alert }
	funcMap["args"] = func() map[string]string { return data.Args }

	// Array functions. Piped value is the last argument, so usage looks like:
	//   {{event.hosts | arrayJoin ", "}}
	//   {{event.counts | arrayMax}}
	//   {{event.hosts | arrayContains "1.2.3.4"}}
	funcMap["arrayJoin"] = func(sep string, arr any) (string, error) {
		switch v := arr.(type) {
		case []string:
			return strings.Join(v, sep), nil
		case []int64:
			strs := make([]string, len(v))
			for i, n := range v {
				strs[i] = strconv.FormatInt(n, 10)
			}
			return strings.Join(strs, sep), nil
		case []bool:
			strs := make([]string, len(v))
			for i, b := range v {
				if b {
					strs[i] = "true"
				} else {
					strs[i] = "false"
				}
			}
			return strings.Join(strs, sep), nil
		default:
			return "", fmt.Errorf("arrayJoin: unsupported type %T", arr)
		}
	}

	funcMap["arrayMax"] = func(arr any) (int64, error) {
		v, ok := arr.([]int64)
		if !ok {
			return 0, fmt.Errorf("arrayMax: requires []int64, got %T", arr)
		}
		if len(v) == 0 {
			return 0, fmt.Errorf("arrayMax: empty array")
		}
		max := v[0]
		for _, n := range v[1:] {
			if n > max {
				max = n
			}
		}
		return max, nil
	}

	funcMap["arrayMin"] = func(arr any) (int64, error) {
		v, ok := arr.([]int64)
		if !ok {
			return 0, fmt.Errorf("arrayMin: requires []int64, got %T", arr)
		}
		if len(v) == 0 {
			return 0, fmt.Errorf("arrayMin: empty array")
		}
		min := v[0]
		for _, n := range v[1:] {
			if n < min {
				min = n
			}
		}
		return min, nil
	}

	funcMap["arraySum"] = func(arr any) (int64, error) {
		v, ok := arr.([]int64)
		if !ok {
			return 0, fmt.Errorf("arraySum: requires []int64, got %T", arr)
		}
		var sum int64
		for _, n := range v {
			sum += n
		}
		return sum, nil
	}

	funcMap["arrayFirst"] = func(arr any) (any, error) {
		switch v := arr.(type) {
		case []string:
			if len(v) == 0 {
				return "", fmt.Errorf("arrayFirst: empty array")
			}
			return v[0], nil
		case []int64:
			if len(v) == 0 {
				return int64(0), fmt.Errorf("arrayFirst: empty array")
			}
			return v[0], nil
		case []bool:
			if len(v) == 0 {
				return false, fmt.Errorf("arrayFirst: empty array")
			}
			return v[0], nil
		default:
			return nil, fmt.Errorf("arrayFirst: unsupported type %T", arr)
		}
	}

	funcMap["arrayLast"] = func(arr any) (any, error) {
		switch v := arr.(type) {
		case []string:
			if len(v) == 0 {
				return "", fmt.Errorf("arrayLast: empty array")
			}
			return v[len(v)-1], nil
		case []int64:
			if len(v) == 0 {
				return int64(0), fmt.Errorf("arrayLast: empty array")
			}
			return v[len(v)-1], nil
		case []bool:
			if len(v) == 0 {
				return false, fmt.Errorf("arrayLast: empty array")
			}
			return v[len(v)-1], nil
		default:
			return nil, fmt.Errorf("arrayLast: unsupported type %T", arr)
		}
	}

	funcMap["arrayContains"] = func(val any, arr any) (bool, error) {
		switch v := arr.(type) {
		case []string:
			s, ok := val.(string)
			if !ok {
				return false, fmt.Errorf("arrayContains: value must be string for []string array")
			}
			for _, item := range v {
				if item == s {
					return true, nil
				}
			}
			return false, nil
		case []int64:
			var n int64
			switch sv := val.(type) {
			case int64:
				n = sv
			case int:
				n = int64(sv)
			default:
				return false, fmt.Errorf("arrayContains: value must be int64 for []int64 array")
			}
			for _, item := range v {
				if item == n {
					return true, nil
				}
			}
			return false, nil
		case []bool:
			b, ok := val.(bool)
			if !ok {
				return false, fmt.Errorf("arrayContains: value must be bool for []bool array")
			}
			for _, item := range v {
				if item == b {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, fmt.Errorf("arrayContains: unsupported array type %T", arr)
		}
	}

	t, err := template.New("notify").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
