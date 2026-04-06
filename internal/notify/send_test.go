package notify

import (
	"testing"
)

func TestResolveTargets_Basic(t *testing.T) {
	channels := map[string]ChannelDef{
		"telegram": {URL: "telegram://token@telegram", Params: map[string]string{"chats": "123"}},
	}
	refs := []NotifyRef{
		{ChannelName: "telegram"},
	}
	data := BuildTemplateData(map[string]any{"hostname": "vps-01"}, "disk_check",
		map[string]string{"type": "high_usage", "usage": "84"},
		map[string]any{"mount": "/"},
	)

	targets, err := ResolveTargets(refs, channels, `{{event.type | upper}} {{globals.hostname}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	if targets[0].Message != "HIGH_USAGE vps-01" {
		t.Errorf("message = %q, want %q", targets[0].Message, "HIGH_USAGE vps-01")
	}
	if targets[0].Params["chats"] != "123" {
		t.Errorf("chats param = %q, want %q", targets[0].Params["chats"], "123")
	}
}

func TestResolveTargets_ParamMerge(t *testing.T) {
	channels := map[string]ChannelDef{
		"telegram": {
			URL:    "telegram://token@telegram",
			Params: map[string]string{"chats": "123", "parsemode": "HTML"},
		},
	}
	refs := []NotifyRef{
		{
			ChannelName: "telegram",
			Params:      map[string]string{"parsemode": "MarkdownV2"},
		},
	}
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil)

	targets, err := ResolveTargets(refs, channels, `test`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targets[0].Params["chats"] != "123" {
		t.Errorf("chats = %q, want %q", targets[0].Params["chats"], "123")
	}
	if targets[0].Params["parsemode"] != "MarkdownV2" {
		t.Errorf("parsemode = %q, want %q", targets[0].Params["parsemode"], "MarkdownV2")
	}
}

func TestResolveTargets_TemplateInParams(t *testing.T) {
	channels := map[string]ChannelDef{
		"email": {URL: "smtp://user:pass@host"},
	}
	refs := []NotifyRef{
		{
			ChannelName: "email",
			Params:      map[string]string{"subject": `[{{event.type | upper}}] {{globals.hostname}}`},
		},
	}
	data := BuildTemplateData(map[string]any{"hostname": "vps-01"}, "alert",
		map[string]string{"type": "critical_usage"}, nil)

	targets, err := ResolveTargets(refs, channels, `body`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targets[0].Params["subject"] != "[CRITICAL_USAGE] vps-01" {
		t.Errorf("subject = %q, want %q", targets[0].Params["subject"], "[CRITICAL_USAGE] vps-01")
	}
}

func TestResolveTargets_UnknownService(t *testing.T) {
	channels := map[string]ChannelDef{}
	refs := []NotifyRef{{ChannelName: "nonexistent"}}
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil)

	_, err := ResolveTargets(refs, channels, `test`, data)
	if err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func TestResolveTargets_MultipleTargets(t *testing.T) {
	channels := map[string]ChannelDef{
		"telegram": {URL: "telegram://token@telegram"},
		"slack":    {URL: "slack://token-a/token-b/token-c"},
	}
	refs := []NotifyRef{
		{ChannelName: "telegram"},
		{ChannelName: "slack"},
	}
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil)

	targets, err := ResolveTargets(refs, channels, `msg`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
}

func TestApplyParams(t *testing.T) {
	got, err := applyParams("telegram://token@telegram", map[string]string{
		"chats": "-100123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "telegram://token@telegram?chats=-100123" {
		t.Errorf("url = %q, want params appended", got)
	}
}

func TestApplyParams_MergesExisting(t *testing.T) {
	got, err := applyParams("telegram://token@telegram?existing=yes", map[string]string{
		"chats": "123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both params should be present
	if got != "telegram://token@telegram?chats=123&existing=yes" {
		t.Errorf("url = %q, want both params", got)
	}
}

func TestApplyParams_Empty(t *testing.T) {
	got, err := applyParams("telegram://token@telegram", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "telegram://token@telegram" {
		t.Errorf("url = %q, want unchanged", got)
	}
}
