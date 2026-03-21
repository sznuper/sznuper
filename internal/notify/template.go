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
	Event   map[string]any
	Args    map[string]string
}

// BuildTemplateData constructs template data from event output and config.
func BuildTemplateData(globals map[string]any, alertName string, eventFields map[string]string, args map[string]any) TemplateData {
	alert := map[string]string{
		"name": alertName,
	}

	ev := make(map[string]any, len(eventFields))
	for k, v := range eventFields {
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
