package notify

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TemplateData holds all data available to notification templates.
type TemplateData struct {
	Globals map[string]any
	Alert   map[string]string
	Check   map[string]string
	Args    map[string]string
}

// BuildTemplateData constructs template data from check output and config.
func BuildTemplateData(globals map[string]any, alertName string, checkFields map[string]string, args map[string]any) TemplateData {
	alert := map[string]string{
		"name": alertName,
	}

	// Copy check fields and add derived status_emoji.
	check := make(map[string]string, len(checkFields)+1)
	for k, v := range checkFields {
		check[k] = v
	}
	check["status_emoji"] = statusEmoji(check["status"])

	// Convert args to string map.
	argsStr := make(map[string]string, len(args))
	for k, v := range args {
		argsStr[k] = fmt.Sprint(v)
	}

	return TemplateData{
		Globals: globals,
		Alert:   alert,
		Check:   check,
		Args:    argsStr,
	}
}

func statusEmoji(status string) string {
	switch status {
	case "critical":
		return "\U0001f534" // 🔴
	case "warning":
		return "\U0001f7e1" // 🟡
	case "ok":
		return "\U0001f7e2" // 🟢
	default:
		return "\u2753" // ❓
	}
}

// Render executes a Go text/template string with Sprig functions and the
// custom accessor functions (check, globals, alert, args).
func Render(tmplStr string, data TemplateData) (string, error) {
	funcMap := sprig.TxtFuncMap()

	// Register accessor functions so {{check.status}} works:
	// "check" returns the check map, then ".status" accesses a key.
	funcMap["check"] = func() map[string]string { return data.Check }
	funcMap["globals"] = func() map[string]any { return data.Globals }
	funcMap["alert"] = func() map[string]string { return data.Alert }
	funcMap["args"] = func() map[string]string { return data.Args }

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
