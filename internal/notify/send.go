package notify

import (
	"fmt"
	"net/url"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
)

// Target holds a fully resolved notification target ready to send.
type Target struct {
	ServiceName string
	URL         string
	Message     string
	Params      map[string]string
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

		// Merge params: service base ← per-target override.
		merged := make(map[string]string)
		for k, v := range svc.Params {
			merged[k] = v
		}
		for k, v := range ref.Params {
			merged[k] = v
		}

		// Render template vars inside param values.
		for k, v := range merged {
			rendered, err := Render(v, data)
			if err != nil {
				return nil, fmt.Errorf("rendering param %q for %s: %w", k, ref.ServiceName, err)
			}
			merged[k] = rendered
		}

		targets = append(targets, Target{
			ServiceName: ref.ServiceName,
			URL:         svc.URL,
			Message:     msg,
			Params:      merged,
		})
	}

	return targets, nil
}

// Validate builds the full URL and creates a Shoutrrr sender to verify
// the service configuration is valid, without actually sending anything.
func Validate(t Target) error {
	_, err := buildSender(t)
	return err
}

// Send delivers a notification to a single target via Shoutrrr.
// Params are merged into the URL as query parameters before creating the sender.
func Send(t Target) error {
	sender, err := buildSender(t)
	if err != nil {
		return err
	}

	errs := sender.Send(t.Message, nil)
	for _, e := range errs {
		if e != nil {
			return fmt.Errorf("sending to %s: %w", t.ServiceName, e)
		}
	}

	return nil
}

func buildSender(t Target) (*router.ServiceRouter, error) {
	fullURL, err := applyParams(t.URL, t.Params)
	if err != nil {
		return nil, fmt.Errorf("building URL for %s: %w", t.ServiceName, err)
	}

	sender, err := shoutrrr.CreateSender(fullURL)
	if err != nil {
		return nil, fmt.Errorf("creating sender for %s: %w", t.ServiceName, err)
	}

	return sender, nil
}

// applyParams merges params into the URL as query parameters.
func applyParams(rawURL string, params map[string]string) (string, error) {
	if len(params) == 0 {
		return rawURL, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// NotifyRef is a simplified notify target reference used by ResolveTargets.
type NotifyRef struct {
	ServiceName string
	Template    string
	Params      map[string]string
}

// ServiceDef is a simplified service definition used by ResolveTargets.
type ServiceDef struct {
	URL    string
	Params map[string]string
}
