package notify

import (
	"testing"
)

func TestRender_Basic(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "vps-01"}, "disk_check",
		map[string]string{"type": "high_usage", "usage": "84"},
		nil,
		map[string]any{"mount": "/"},
	)

	result, err := Render(`{{event.type | upper}} {{globals.hostname}}: Disk at {{event.usage}}%`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "HIGH_USAGE vps-01: Disk at 84%" {
		t.Errorf("result = %q, want %q", result, "HIGH_USAGE vps-01: Disk at 84%")
	}
}

func TestRender_ArgsAccess(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"},
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
		map[string]string{"type": "ok"}, nil, nil)

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
		map[string]string{"type": "ok", "msg": "hello"}, nil, nil)

	result, err := Render(`{{event.msg | upper | repeat 2}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "HELLOHELLO" {
		t.Errorf("result = %q, want %q", result, "HELLOHELLO")
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil, nil)

	_, err := Render(`{{event.type | nonexistent}}`, data)
	if err == nil {
		t.Fatal("expected error for invalid template function")
	}
}

func TestBuildTemplateData_NilArgs(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil, nil)

	if data.Args == nil {
		t.Error("Args should be non-nil empty map")
	}
}

func TestRender_DefaultSprigFunc(t *testing.T) {
	data := BuildTemplateData(map[string]any{"hostname": "host"}, "alert",
		map[string]string{"type": "ok"}, nil, nil)

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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.hosts | arrayJoin ", "}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.counts | arrayMax}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.counts | arrayMin}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.counts | arraySum}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.hosts | arrayFirst}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.hosts | arrayLast}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{if event.hosts | arrayContains "1.2.3.4"}}yes{{else}}no{{end}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{if event.hosts | arrayContains "9.9.9.9"}}yes{{else}}no{{end}}`, data)
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
		map[string]string{"type": "ok"}, arrays, nil)

	result, err := Render(`{{event.counts | arrayJoin "-"}}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1-2-3" {
		t.Errorf("result = %q, want %q", result, "1-2-3")
	}
}
