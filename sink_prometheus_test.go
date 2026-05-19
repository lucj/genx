package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

// freePort returns an available TCP port by letting the OS bind to :0.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func getMetrics(t *testing.T, port int) string {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestPrometheusSinkSingleField(t *testing.T) {
	port := freePort(t)
	sink, err := NewPrometheusSink(port, "genx")
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	defer sink.Close()

	val := 42.5
	if err := sink.Send(DataPoint{Device: "dev-0", Timestamp: 1000, Value: &val}); err != nil {
		t.Fatalf("Send error: %v", err)
	}

	body := getMetrics(t, port)
	if !strings.Contains(body, "# TYPE genx gauge") {
		t.Errorf("missing TYPE comment; got:\n%s", body)
	}
	if !strings.Contains(body, `genx{device="dev-0"} 42.5`) {
		t.Errorf("missing metric line; got:\n%s", body)
	}
}

func TestPrometheusSinkMultiDevice(t *testing.T) {
	port := freePort(t)
	sink, err := NewPrometheusSink(port, "sensor")
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	defer sink.Close()

	for i, v := range []float64{10.0, 20.0, 30.0} {
		val := v
		sink.Send(DataPoint{Device: fmt.Sprintf("dev-%d", i), Timestamp: 1000, Value: &val})
	}

	body := getMetrics(t, port)
	for i := range []int{0, 1, 2} {
		if !strings.Contains(body, fmt.Sprintf(`sensor{device="dev-%d"}`, i)) {
			t.Errorf("missing device dev-%d in output:\n%s", i, body)
		}
	}
}

func TestPrometheusSinkMultiField(t *testing.T) {
	port := freePort(t)
	sink, err := NewPrometheusSink(port, "genx")
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	defer sink.Close()

	sink.Send(DataPoint{
		Device:    "room1",
		Timestamp: 5000,
		Fields:    map[string]float64{"temperature": 22.5, "humidity": 60.0},
	})

	body := getMetrics(t, port)
	if !strings.Contains(body, "# TYPE genx_temperature gauge") {
		t.Errorf("missing temperature metric; got:\n%s", body)
	}
	if !strings.Contains(body, "# TYPE genx_humidity gauge") {
		t.Errorf("missing humidity metric; got:\n%s", body)
	}
}

func TestPrometheusSinkNilValue(t *testing.T) {
	port := freePort(t)
	sink, err := NewPrometheusSink(port, "genx")
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	defer sink.Close()

	// nil Value — Send should be a no-op, /metrics should return no data lines.
	sink.Send(DataPoint{Device: "dev", Timestamp: 1, Value: nil})
	body := getMetrics(t, port)
	if strings.Contains(body, "genx{") {
		t.Errorf("expected no metric line for nil value, got:\n%s", body)
	}
}

func TestPrometheusSinkPortConflict(t *testing.T) {
	port := freePort(t)
	sink1, err := NewPrometheusSink(port, "genx")
	if err != nil {
		t.Fatalf("first sink failed: %v", err)
	}
	defer sink1.Close()

	_, err = NewPrometheusSink(port, "genx")
	if err == nil {
		t.Error("expected error on duplicate port, got nil")
	}
}

func TestPrometheusSinkLabelEscaping(t *testing.T) {
	port := freePort(t)
	sink, err := NewPrometheusSink(port, "genx")
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	defer sink.Close()

	val := 1.0
	sink.Send(DataPoint{Device: `dev"quoted`, Timestamp: 1, Value: &val})
	body := getMetrics(t, port)
	if !strings.Contains(body, `"dev\"quoted"`) {
		t.Errorf("expected escaped label value; got:\n%s", body)
	}
}

func TestPromSanitizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"temperature", "temperature"},
		{"my.field", "my_field"},
		{"field-name", "field_name"},
		{"field name", "field_name"},
	}
	for _, c := range cases {
		if got := promSanitizeName(c.in); got != c.want {
			t.Errorf("sanitize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
