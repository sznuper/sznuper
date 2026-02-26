package notify

import (
	"testing"
)

func TestResolveTargets_Basic(t *testing.T) {
	services := map[string]ServiceDef{
		"telegram": {URL: "telegram://token@telegram", Options: map[string]string{"chats": "123"}},
	}
	refs := []NotifyRef{
		{ServiceName: "telegram"},
	}
	data := BuildTemplateData("vps-01", "disk_check",
		map[string]string{"status": "warning", "usage": "84"},
		map[string]any{"mount": "/"},
	)

	targets, err := ResolveTargets(refs, services, `{{check.status | upper}} {{globals.hostname}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	if targets[0].Message != "WARNING vps-01" {
		t.Errorf("message = %q, want %q", targets[0].Message, "WARNING vps-01")
	}
	if targets[0].Options["chats"] != "123" {
		t.Errorf("chats option = %q, want %q", targets[0].Options["chats"], "123")
	}
}

func TestResolveTargets_TemplateOverride(t *testing.T) {
	services := map[string]ServiceDef{
		"telegram": {URL: "telegram://token@telegram"},
	}
	refs := []NotifyRef{
		{ServiceName: "telegram", Template: `CUSTOM: {{check.status}}`},
	}
	data := BuildTemplateData("host", "alert",
		map[string]string{"status": "ok"}, nil)

	targets, err := ResolveTargets(refs, services, `DEFAULT: {{check.status}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targets[0].Message != "CUSTOM: ok" {
		t.Errorf("message = %q, want %q", targets[0].Message, "CUSTOM: ok")
	}
}

func TestResolveTargets_OptionMerge(t *testing.T) {
	services := map[string]ServiceDef{
		"telegram": {
			URL:     "telegram://token@telegram",
			Options: map[string]string{"chats": "123", "parsemode": "HTML"},
		},
	}
	refs := []NotifyRef{
		{
			ServiceName: "telegram",
			Options:     map[string]string{"parsemode": "MarkdownV2"},
		},
	}
	data := BuildTemplateData("host", "alert",
		map[string]string{"status": "ok"}, nil)

	targets, err := ResolveTargets(refs, services, `test`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targets[0].Options["chats"] != "123" {
		t.Errorf("chats = %q, want %q", targets[0].Options["chats"], "123")
	}
	if targets[0].Options["parsemode"] != "MarkdownV2" {
		t.Errorf("parsemode = %q, want %q", targets[0].Options["parsemode"], "MarkdownV2")
	}
}

func TestResolveTargets_TemplateInOptions(t *testing.T) {
	services := map[string]ServiceDef{
		"email": {URL: "smtp://user:pass@host"},
	}
	refs := []NotifyRef{
		{
			ServiceName: "email",
			Options:     map[string]string{"subject": `[{{check.status | upper}}] {{globals.hostname}}`},
		},
	}
	data := BuildTemplateData("vps-01", "alert",
		map[string]string{"status": "critical"}, nil)

	targets, err := ResolveTargets(refs, services, `body`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targets[0].Options["subject"] != "[CRITICAL] vps-01" {
		t.Errorf("subject = %q, want %q", targets[0].Options["subject"], "[CRITICAL] vps-01")
	}
}

func TestResolveTargets_UnknownService(t *testing.T) {
	services := map[string]ServiceDef{}
	refs := []NotifyRef{{ServiceName: "nonexistent"}}
	data := BuildTemplateData("host", "alert",
		map[string]string{"status": "ok"}, nil)

	_, err := ResolveTargets(refs, services, `test`, data)
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestResolveTargets_MultipleTargets(t *testing.T) {
	services := map[string]ServiceDef{
		"telegram": {URL: "telegram://token@telegram"},
		"slack":    {URL: "slack://token-a/token-b/token-c"},
	}
	refs := []NotifyRef{
		{ServiceName: "telegram"},
		{ServiceName: "slack"},
	}
	data := BuildTemplateData("host", "alert",
		map[string]string{"status": "ok"}, nil)

	targets, err := ResolveTargets(refs, services, `msg`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
}
