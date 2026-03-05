package notify

import (
	"testing"
)

func TestRender_Basic(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "vps-01"}, "disk_check",
		map[string]string{"status": "warning", "usage": "84"},
		nil,
		map[string]any{"mount": "/"},
	)

	result, err := Render(`{{healthcheck.status | upper}} {{globals.hostname}}: Disk at {{healthcheck.usage}}%`, data)
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
			map[string]string{"status": tt.status}, nil, nil)
		result, err := Render(`{{healthcheck.status_emoji}}`, data)
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
		nil,
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
		map[string]string{"status": "ok"}, nil, nil)

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
		map[string]string{"status": "ok", "msg": "hello"}, nil, nil)

	result, err := Render(`{{healthcheck.msg | upper | repeat 2}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "HELLOHELLO" {
		t.Errorf("result = %q, want %q", result, "HELLOHELLO")
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil, nil)

	_, err := Render(`{{healthcheck.status | nonexistent}}`, data)
	if err == nil {
		t.Fatal("expected error for invalid template function")
	}
}

func TestBuildTemplateData_NilArgs(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil, nil)

	if data.Args == nil {
		t.Error("Args should be non-nil empty map")
	}
}

func TestRender_DefaultSprigFunc(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"status": "ok"}, nil, nil)

	result, err := Render(`{{args.mount | default "/"}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/" {
		t.Errorf("result = %q, want %q", result, "/")
	}
}

func TestRender_ArrayJoin(t *testing.T) {
	arrays := map[string]any{
		"hosts": []string{"1.2.3.4", "5.6.7.8"},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.hosts | arrayJoin ", "}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1.2.3.4, 5.6.7.8" {
		t.Errorf("result = %q, want %q", result, "1.2.3.4, 5.6.7.8")
	}
}

func TestRender_ArrayMax(t *testing.T) {
	arrays := map[string]any{
		"counts": []int64{3, 1, 4, 1, 5, 9},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.counts | arrayMax}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "9" {
		t.Errorf("result = %q, want %q", result, "9")
	}
}

func TestRender_ArrayMin(t *testing.T) {
	arrays := map[string]any{
		"counts": []int64{3, 1, 4, 1, 5, 9},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.counts | arrayMin}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1" {
		t.Errorf("result = %q, want %q", result, "1")
	}
}

func TestRender_ArraySum(t *testing.T) {
	arrays := map[string]any{
		"counts": []int64{1, 2, 3},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.counts | arraySum}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "6" {
		t.Errorf("result = %q, want %q", result, "6")
	}
}

func TestRender_ArrayFirst(t *testing.T) {
	arrays := map[string]any{
		"hosts": []string{"first.host", "second.host"},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.hosts | arrayFirst}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "first.host" {
		t.Errorf("result = %q, want %q", result, "first.host")
	}
}

func TestRender_ArrayLast(t *testing.T) {
	arrays := map[string]any{
		"hosts": []string{"first.host", "last.host"},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.hosts | arrayLast}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "last.host" {
		t.Errorf("result = %q, want %q", result, "last.host")
	}
}

func TestRender_ArrayContains_True(t *testing.T) {
	arrays := map[string]any{
		"hosts": []string{"1.2.3.4", "5.6.7.8"},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{if healthcheck.hosts | arrayContains "1.2.3.4"}}yes{{else}}no{{end}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "yes" {
		t.Errorf("result = %q, want %q", result, "yes")
	}
}

func TestRender_ArrayContains_False(t *testing.T) {
	arrays := map[string]any{
		"hosts": []string{"1.2.3.4", "5.6.7.8"},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{if healthcheck.hosts | arrayContains "9.9.9.9"}}yes{{else}}no{{end}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "no" {
		t.Errorf("result = %q, want %q", result, "no")
	}
}

func TestRender_ArrayJoin_IntArray(t *testing.T) {
	arrays := map[string]any{
		"counts": []int64{1, 2, 3},
	}
	data := BuildTemplateData(map[string]any{}, "alert",
		map[string]string{"status": "ok"}, arrays, nil)

	result, err := Render(`{{healthcheck.counts | arrayJoin "-"}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1-2-3" {
		t.Errorf("result = %q, want %q", result, "1-2-3")
	}
}
