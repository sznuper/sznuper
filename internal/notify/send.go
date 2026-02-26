package notify

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
)

// Target holds a fully resolved notification target ready to send.
type Target struct {
	ServiceName string
	URL         string
	Message     string
	Options     map[string]string
}

// ResolveTargets builds the list of notification targets from an alert's
// notify list, service definitions, and template data. It renders the message
// template and option value templates for each target.
func ResolveTargets(
	notifyList []NotifyRef,
	services map[string]ServiceDef,
	defaultTemplate string,
	data TemplateData,
) ([]Target, error) {
	var targets []Target

	for _, ref := range notifyList {
		svc, ok := services[ref.ServiceName]
		if !ok {
			return nil, fmt.Errorf("unknown service %q", ref.ServiceName)
		}

		// Pick template: per-target override → alert default.
		tmplStr := defaultTemplate
		if ref.Template != "" {
			tmplStr = ref.Template
		}

		msg, err := Render(tmplStr, data)
		if err != nil {
			return nil, fmt.Errorf("rendering template for %s: %w", ref.ServiceName, err)
		}

		// Merge options: service base ← per-target override.
		merged := make(map[string]string)
		for k, v := range svc.Options {
			merged[k] = v
		}
		for k, v := range ref.Options {
			merged[k] = v
		}

		// Render template vars inside option values.
		for k, v := range merged {
			rendered, err := Render(v, data)
			if err != nil {
				return nil, fmt.Errorf("rendering option %q for %s: %w", k, ref.ServiceName, err)
			}
			merged[k] = rendered
		}

		targets = append(targets, Target{
			ServiceName: ref.ServiceName,
			URL:         svc.URL,
			Message:     msg,
			Options:     merged,
		})
	}

	return targets, nil
}

// Send delivers a notification to a single target via Shoutrrr.
func Send(t Target) error {
	sender, err := shoutrrr.CreateSender(t.URL)
	if err != nil {
		return fmt.Errorf("creating sender for %s: %w", t.ServiceName, err)
	}

	params := types.Params(t.Options)
	errs := sender.Send(t.Message, &params)
	for _, e := range errs {
		if e != nil {
			return fmt.Errorf("sending to %s: %w", t.ServiceName, e)
		}
	}

	return nil
}

// NotifyRef is a simplified notify target reference used by ResolveTargets.
type NotifyRef struct {
	ServiceName string
	Template    string
	Options     map[string]string
}

// ServiceDef is a simplified service definition used by ResolveTargets.
type ServiceDef struct {
	URL     string
	Options map[string]string
}
