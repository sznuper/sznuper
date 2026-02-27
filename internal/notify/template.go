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
	Healthcheck map[string]string
	Args    map[string]string
}

// BuildTemplateData constructs template data from healthcheck output and config.
func BuildTemplateData(globals map[string]any, alertName string, healthcheckFields map[string]string, args map[string]any) TemplateData {
	alert := map[string]string{
		"name": alertName,
	}

	// Copy healthcheck fields and add derived status_emoji.
	hc := make(map[string]string, len(healthcheckFields)+1)
	for k, v := range healthcheckFields {
		hc[k] = v
	}
	hc["status_emoji"] = statusEmoji(hc["status"])

	// Convert args to string map.
	argsStr := make(map[string]string, len(args))
	for k, v := range args {
		argsStr[k] = fmt.Sprint(v)
	}

	return TemplateData{
		Globals:     globals,
		Alert:       alert,
		Healthcheck: hc,
		Args:        argsStr,
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
// custom accessor functions (healthcheck, globals, alert, args).
func Render(tmplStr string, data TemplateData) (string, error) {
	funcMap := sprig.TxtFuncMap()

	// Register accessor functions so {{healthcheck.status}} works:
	// "healthcheck" returns the healthcheck map, then ".status" accesses a key.
	funcMap["healthcheck"] = func() map[string]string { return data.Healthcheck }
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
