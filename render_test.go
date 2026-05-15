package main

import (
	"encoding/json"
	"testing"
	"text/template"
)

func ptrF(v float64) *float64 { return &v }

func TestJSONRenderer(t *testing.T) {
	dp := DataPoint{Device: "dev1", Timestamp: 1000, Value: ptrF(42.0)}
	b, err := JSONRenderer(dp)
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

func TestTemplateRenderer(t *testing.T) {
	tmpl := template.Must(template.New("p").Parse(`{"sensor":"{{.Device}}","ts":{{.Timestamp}},"val":{{.Value}}}`))
	render := NewTemplateRenderer(tmpl)

	dp := DataPoint{Device: "sensor-a", Timestamp: 9999, Value: ptrF(7.5)}
	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
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

func TestTemplateRendererWithFields(t *testing.T) {
	tmpl := template.Must(template.New("p").Parse(`{"device":"{{.Device}}","temp":{{.Fields.temperature}}}`))
	render := NewTemplateRenderer(tmpl)

	dp := DataPoint{
		Device:    "room1",
		Timestamp: 5000,
		Fields:    map[string]float64{"temperature": 21.3},
	}
	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v\n%s", err, b)
	}
	if m["device"] != "room1" {
		t.Errorf("expected room1, got %v", m["device"])
	}
}

func TestTemplateRendererNonJSON(t *testing.T) {
	tmpl := template.Must(template.New("p").Parse(`not json at all {{.Device}}`))
	render := NewTemplateRenderer(tmpl)
	dp := DataPoint{Device: "x", Timestamp: 1}
	if _, err := render(dp); err == nil {
		t.Fatal("expected error for non-JSON template output")
	}
}

func TestTemplateRendererNilValue(t *testing.T) {
	tmpl := template.Must(template.New("p").Parse(`{"device":"{{.Device}}","val":{{.Value}}}`))
	render := NewTemplateRenderer(tmpl)

	// Value is nil — template should see 0.0, not <nil>
	dp := DataPoint{Device: "dev", Timestamp: 1, Value: nil}
	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v\n%s", err, b)
	}
	if m["val"] != float64(0) {
		t.Errorf("expected val=0, got %v", m["val"])
	}
}
