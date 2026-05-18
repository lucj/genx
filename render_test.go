package main

import (
	"encoding/json"
	"strings"
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
	render := NewTemplateRenderer(tmpl, false)

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
	render := NewTemplateRenderer(tmpl, false)

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
	render := NewTemplateRenderer(tmpl, false)
	dp := DataPoint{Device: "x", Timestamp: 1}
	if _, err := render(dp); err == nil {
		t.Fatal("expected error for non-JSON template output")
	}
}

func TestCSVRendererSingleFieldHeader(t *testing.T) {
	render := NewCSVRenderer(false)
	b, err := render(DataPoint{Device: "dev", Timestamp: 1000, Value: ptrF(22.5)})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines on first call (header + row), got %d: %q", len(lines), b)
	}
	if lines[0] != "device,timestamp,value" {
		t.Errorf("unexpected header: %q", lines[0])
	}
	if lines[1] != "dev,1000,22.5" {
		t.Errorf("unexpected row: %q", lines[1])
	}
}

func TestCSVRendererSingleFieldNoHeaderOnSubsequentCalls(t *testing.T) {
	render := NewCSVRenderer(false)
	render(DataPoint{Device: "dev", Timestamp: 1000, Value: ptrF(1.0)}) // first call — header
	b, err := render(DataPoint{Device: "dev", Timestamp: 1001, Value: ptrF(2.0)})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if strings.Contains(string(b), "device") {
		t.Errorf("second call should not contain header, got: %q", b)
	}
	if string(b) != "dev,1001,2" {
		t.Errorf("unexpected row: %q", b)
	}
}

func TestCSVRendererMultiFieldSortedColumns(t *testing.T) {
	render := NewCSVRenderer(false)
	dp := DataPoint{
		Device:    "sensor",
		Timestamp: 5000,
		Fields:    map[string]float64{"humidity": 60.0, "temperature": 22.0, "pressure": 1013.0},
	}
	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), b)
	}
	if lines[0] != "device,timestamp,humidity,pressure,temperature" {
		t.Errorf("unexpected header (want sorted fields): %q", lines[0])
	}
	if lines[1] != "sensor,5000,60,1013,22" {
		t.Errorf("unexpected row: %q", lines[1])
	}
}

func TestCSVRendererISOTime(t *testing.T) {
	render := NewCSVRenderer(true)
	b, err := render(DataPoint{Device: "dev", Timestamp: 0, Value: ptrF(1.0)})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(string(b), "1970-01-01T00:00:00Z") {
		t.Errorf("expected ISO timestamp, got: %q", b)
	}
}

func TestCSVRendererNilValue(t *testing.T) {
	render := NewCSVRenderer(false)
	b, err := render(DataPoint{Device: "dev", Timestamp: 1, Value: nil})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	lines := strings.Split(string(b), "\n")
	// row should have empty value field
	if !strings.HasSuffix(lines[len(lines)-1], ",") {
		t.Errorf("expected empty value column, got: %q", lines[len(lines)-1])
	}
}

func TestTemplateRendererNilValue(t *testing.T) {
	tmpl := template.Must(template.New("p").Parse(`{"device":"{{.Device}}","val":{{.Value}}}`))
	render := NewTemplateRenderer(tmpl, false)

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
