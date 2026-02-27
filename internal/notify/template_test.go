package notify

import (
	"testing"
)

func TestRender_Basic(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "vps-01"}, "disk_check",
		map[string]string{"status": "warning", "usage": "84"},
		map[string]any{"mount": "/"},
	)

	result, err := Render(`{{check.status | upper}} {{globals.hostname}}: Disk at {{check.usage}}%`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "WARNING vps-01: Disk at 84%" {
		t.Errorf("result = %q, want %q", result, "WARNING vps-01: Disk at 84%")
	}
}

func TestRender_StatusEmoji(t *testing.T) {
	tests := []struct {
		status string
		emoji  string
	}{
		{"ok", "\U0001f7e2"},
		{"warning", "\U0001f7e1"},
		{"critical", "\U0001f534"},
		{"unknown", "\u2753"},
	}
	for _, tt := range tests {
		data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
			map[string]string{"status": tt.status}, nil)
		result, err := Render(`{{check.status_emoji}}`, data)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.status, err)
		}
		if result != tt.emoji {
			t.Errorf("status=%s: emoji = %q, want %q", tt.status, result, tt.emoji)
		}
	}
}

func TestRender_ArgsAccess(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"},
		map[string]any{"mount": "/data", "threshold": 0.8},
	)

	result, err := Render(`mount={{args.mount}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "mount=/data" {
		t.Errorf("result = %q, want %q", result, "mount=/data")
	}
}

func TestRender_AlertName(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "my_alert",
		map[string]string{"status": "ok"}, nil)

	result, err := Render(`alert={{alert.name}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alert=my_alert" {
		t.Errorf("result = %q, want %q", result, "alert=my_alert")
	}
}

func TestRender_SprigFunctions(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok", "msg": "hello"}, nil)

	result, err := Render(`{{check.msg | upper | repeat 2}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "HELLOHELLO" {
		t.Errorf("result = %q, want %q", result, "HELLOHELLO")
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil)

	_, err := Render(`{{check.status | nonexistent}}`, data)
	if err == nil {
		t.Fatal("expected error for invalid template function")
	}
}

func TestBuildTemplateData_NilArgs(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil)

	if data.Args == nil {
		t.Error("Args should be non-nil empty map")
	}
}

func TestRender_DefaultSprigFunc(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil)

	result, err := Render(`{{args.mount | default "/"}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/" {
		t.Errorf("result = %q, want %q", result, "/")
	}
}
