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

func TestInfluxRendererSingleField(t *testing.T) {
	render := NewInfluxRenderer("sensors")
	b, err := render(DataPoint{Device: "dev-1", Timestamp: 1000, Value: ptrF(22.5)})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	got := string(b)
	want := "sensors,device=dev-1 value=22.5 1000000000000"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestInfluxRendererMultiField(t *testing.T) {
	render := NewInfluxRenderer("")
	dp := DataPoint{
		Device:    "room1",
		Timestamp: 5000,
		Fields:    map[string]float64{"humidity": 60.0, "temperature": 22.0},
	}
	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	got := string(b)
	// fields must be sorted
	want := "genx,device=room1 humidity=60,temperature=22 5000000000000"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestInfluxRendererNilValue(t *testing.T) {
	render := NewInfluxRenderer("genx")
	b, err := render(DataPoint{Device: "dev", Timestamp: 1, Value: nil})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "value=0") {
		t.Errorf("expected value=0 for nil Value, got %q", got)
	}
}

func TestInfluxRendererEscaping(t *testing.T) {
	render := NewInfluxRenderer("my measurement")
	b, err := render(DataPoint{Device: "dev,1=a", Timestamp: 1, Value: ptrF(1.0)})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	got := string(b)
	// spaces in measurement and commas/equals in tag value must be escaped
	if !strings.HasPrefix(got, `my\ measurement,device=dev\,1\=a`) {
		t.Errorf("expected escaped line, got %q", got)
	}
}

func TestCloudEventRendererStructure(t *testing.T) {
	render := NewCloudEventRenderer("/myapp", "com.example.sensor")
	dp := DataPoint{Device: "sensor-1", Timestamp: 1715000000, Value: ptrF(24.5)}

	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	var ce map[string]any
	if err := json.Unmarshal(b, &ce); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, b)
	}

	checks := map[string]string{
		"specversion":     "1.0",
		"type":            "com.example.sensor",
		"datacontenttype": "application/json",
	}
	for field, want := range checks {
		if got, ok := ce[field].(string); !ok || got != want {
			t.Errorf("ce.%s: want %q, got %v", field, want, ce[field])
		}
	}
	if source, _ := ce["source"].(string); source != "/myapp/sensor-1" {
		t.Errorf("ce.source: want /myapp/sensor-1, got %v", source)
	}
	if id, _ := ce["id"].(string); len(id) == 0 {
		t.Error("ce.id should not be empty")
	}
	if _, ok := ce["time"].(string); !ok {
		t.Error("ce.time should be a string")
	}
}

func TestCloudEventRendererDataPayload(t *testing.T) {
	render := NewCloudEventRenderer("", "")
	dp := DataPoint{Device: "dev", Timestamp: 1000, Value: ptrF(7.0)}

	b, _ := render(dp)
	var ce map[string]any
	json.Unmarshal(b, &ce)

	data, ok := ce["data"].(map[string]any)
	if !ok {
		t.Fatalf("ce.data should be a JSON object, got %T", ce["data"])
	}
	if data["device"] != "dev" {
		t.Errorf("ce.data.device: want dev, got %v", data["device"])
	}
	if data["value"] != 7.0 {
		t.Errorf("ce.data.value: want 7.0, got %v", data["value"])
	}
}

func TestCloudEventRendererMultiField(t *testing.T) {
	render := NewCloudEventRenderer("/fleet", "io.genx.measurement")
	dp := DataPoint{
		Device:    "truck-0",
		Timestamp: 2000,
		Fields:    map[string]float64{"lat": 48.86, "lon": 2.35},
	}

	b, err := render(dp)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	var ce map[string]any
	json.Unmarshal(b, &ce)

	data, ok := ce["data"].(map[string]any)
	if !ok {
		t.Fatalf("ce.data should be a JSON object")
	}
	fields, ok := data["fields"].(map[string]any)
	if !ok {
		t.Fatalf("ce.data.fields should be a JSON object")
	}
	if fields["lat"] != 48.86 {
		t.Errorf("ce.data.fields.lat: want 48.86, got %v", fields["lat"])
	}
}

func TestCloudEventRendererDefaults(t *testing.T) {
	render := NewCloudEventRenderer("", "")
	dp := DataPoint{Device: "x", Timestamp: 1}
	b, _ := render(dp)

	var ce map[string]any
	json.Unmarshal(b, &ce)

	if src := ce["source"]; src != "/genx/x" {
		t.Errorf("default source should be /genx/<device>, got %v", src)
	}
	if typ := ce["type"]; typ != "io.genx.measurement" {
		t.Errorf("default type should be io.genx.measurement, got %v", typ)
	}
}

func TestCloudEventRendererUniqueIDs(t *testing.T) {
	render := NewCloudEventRenderer("", "")
	dp := DataPoint{Device: "dev", Timestamp: 1, Value: ptrF(1.0)}
	ids := map[string]bool{}
	for i := 0; i < 20; i++ {
		b, _ := render(dp)
		var ce map[string]any
		json.Unmarshal(b, &ce)
		id := ce["id"].(string)
		if ids[id] {
			t.Errorf("duplicate event ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestCloudEventRendererTimestamp(t *testing.T) {
	render := NewCloudEventRenderer("", "")
	dp := DataPoint{Device: "dev", Timestamp: 0, Value: ptrF(1.0)} // Unix epoch
	b, _ := render(dp)

	var ce map[string]any
	json.Unmarshal(b, &ce)

	if ce["time"] != "1970-01-01T00:00:00Z" {
		t.Errorf("expected epoch timestamp, got %v", ce["time"])
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
