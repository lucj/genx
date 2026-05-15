package main

import (
	"encoding/json"
	"testing"
)

func resetTemplate(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { payloadTmpl = nil })
}

func ptr(v float64) *float64 { return &v }

func TestRenderPayload_NoTemplate(t *testing.T) {
	resetTemplate(t)
	dp := DataPoint{Device: "dev1", Timestamp: 1000, Value: ptr(42.0)}
	b, err := renderPayload(dp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got DataPoint
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got.Device != "dev1" || got.Timestamp != 1000 || *got.Value != 42.0 {
		t.Errorf("unexpected output: %s", b)
	}
}

func TestRenderPayload_Template(t *testing.T) {
	resetTemplate(t)
	tmpl := `{"sensor":"{{.Device}}","ts":{{.Timestamp}},"val":{{.Value}}}`
	if err := initTemplate(tmpl); err != nil {
		t.Fatalf("initTemplate: %v", err)
	}
	dp := DataPoint{Device: "sensor-a", Timestamp: 9999, Value: ptr(7.5)}
	b, err := renderPayload(dp)
	if err != nil {
		t.Fatalf("renderPayload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v\n%s", err, b)
	}
	if m["sensor"] != "sensor-a" {
		t.Errorf("expected sensor-a, got %v", m["sensor"])
	}
	if m["ts"] != float64(9999) {
		t.Errorf("expected ts 9999, got %v", m["ts"])
	}
}

func TestRenderPayload_TemplateWithFields(t *testing.T) {
	resetTemplate(t)
	tmpl := `{"device":"{{.Device}}","temp":{{.Fields.temperature}}}`
	if err := initTemplate(tmpl); err != nil {
		t.Fatalf("initTemplate: %v", err)
	}
	dp := DataPoint{
		Device:    "room1",
		Timestamp: 5000,
		Fields:    map[string]float64{"temperature": 21.3},
	}
	b, err := renderPayload(dp)
	if err != nil {
		t.Fatalf("renderPayload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v\n%s", err, b)
	}
	if m["device"] != "room1" {
		t.Errorf("expected room1, got %v", m["device"])
	}
}

func TestInitTemplate_InvalidSyntax(t *testing.T) {
	resetTemplate(t)
	err := initTemplate(`{"bad": {{.Missing}`)
	if err == nil {
		t.Fatal("expected parse error for invalid template syntax")
	}
}

func TestRenderPayload_NonJSONTemplate(t *testing.T) {
	resetTemplate(t)
	if err := initTemplate(`not json at all {{.Device}}`); err != nil {
		t.Fatalf("initTemplate: %v", err)
	}
	dp := DataPoint{Device: "x", Timestamp: 1}
	_, err := renderPayload(dp)
	if err == nil {
		t.Fatal("expected error for non-JSON template output")
	}
}

func TestRenderPayload_NilValue(t *testing.T) {
	resetTemplate(t)
	tmpl := `{"device":"{{.Device}}","val":{{.Value}}}`
	if err := initTemplate(tmpl); err != nil {
		t.Fatalf("initTemplate: %v", err)
	}
	// Value is nil — template should see 0.0, not <nil>
	dp := DataPoint{Device: "dev", Timestamp: 1, Value: nil}
	b, err := renderPayload(dp)
	if err != nil {
		t.Fatalf("renderPayload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v\n%s", err, b)
	}
	if m["val"] != float64(0) {
		t.Errorf("expected val=0, got %v", m["val"])
	}
}
