package main

import (
	"testing"
)

func TestCompileTopicPlain(t *testing.T) {
	fn, err := compileTopic("sensors/temp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dp := DataPoint{Device: "sensor-1", Timestamp: 1000}
	got := fn(dp)
	if got != "sensors/temp" {
		t.Errorf("got %q, want %q", got, "sensors/temp")
	}
	// Same function should return the same value for different DataPoints.
	got2 := fn(DataPoint{Device: "other"})
	if got2 != "sensors/temp" {
		t.Errorf("plain topic changed with different DataPoint: %q", got2)
	}
}

func TestCompileTopicDevice(t *testing.T) {
	fn, err := compileTopic("sensors/{{.Device}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cases := []struct {
		device string
		want   string
	}{
		{"sensor-0", "sensors/sensor-0"},
		{"sensor-1", "sensors/sensor-1"},
		{"paris", "sensors/paris"},
	}
	for _, c := range cases {
		got := fn(DataPoint{Device: c.device})
		if got != c.want {
			t.Errorf("device %q: got %q, want %q", c.device, got, c.want)
		}
	}
}

func TestCompileTopicTimestamp(t *testing.T) {
	fn, err := compileTopic("data/{{.Timestamp}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := fn(DataPoint{Device: "x", Timestamp: 1715000000})
	if got != "data/1715000000" {
		t.Errorf("got %q, want %q", got, "data/1715000000")
	}
}

func TestCompileTopicDotSeparatedSubject(t *testing.T) {
	// NATS-style dot-separated subjects
	fn, err := compileTopic("sensors.{{.Device}}.temp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := fn(DataPoint{Device: "zone-a"})
	if got != "sensors.zone-a.temp" {
		t.Errorf("got %q, want %q", got, "sensors.zone-a.temp")
	}
}

func TestCompileTopicInvalidTemplate(t *testing.T) {
	_, err := compileTopic("sensors/{{.Unclosed")
	if err == nil {
		t.Error("expected error for invalid template syntax, got nil")
	}
}

func TestCompileTopicEmpty(t *testing.T) {
	fn, err := compileTopic("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := fn(DataPoint{Device: "any"})
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
