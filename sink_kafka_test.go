package main

import (
	"testing"
)

func TestParseBrokers(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"localhost:9092", []string{"localhost:9092"}},
		{"b1:9092,b2:9092", []string{"b1:9092", "b2:9092"}},
		{"b1:9092, b2:9092 , b3:9092", []string{"b1:9092", "b2:9092", "b3:9092"}},
		{" localhost:9092 ", []string{"localhost:9092"}},
		{"a:9092,,b:9092", []string{"a:9092", "b:9092"}}, // empty segment ignored
	}
	for _, c := range cases {
		got := parseBrokers(c.in)
		if len(got) != len(c.want) {
			t.Errorf("parseBrokers(%q): got %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseBrokers(%q)[%d]: got %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestNewKafkaSinkValidation(t *testing.T) {
	if _, err := NewKafkaSink("", "topic", "", "", false, false, JSONRenderer); err == nil {
		t.Error("expected error for empty brokers, got nil")
	}
	if _, err := NewKafkaSink("localhost:9092", "", "", "", false, false, JSONRenderer); err == nil {
		t.Error("expected error for empty topic, got nil")
	}
	// Invalid template syntax must be rejected at construction time.
	if _, err := NewKafkaSink("localhost:9092", "{{.Unclosed", "", "", false, false, JSONRenderer); err == nil {
		t.Error("expected error for invalid topic template, got nil")
	}
}

func TestNewKafkaSinkTemplateTopic(t *testing.T) {
	sink, err := NewKafkaSink("localhost:9092", "sensors.{{.Device}}", "", "", false, false, JSONRenderer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sink.Close()

	dp0 := DataPoint{Device: "sensor-0", Timestamp: 1000}
	dp1 := DataPoint{Device: "sensor-1", Timestamp: 1001}
	if got := sink.topicFn(dp0); got != "sensors.sensor-0" {
		t.Errorf("topicFn(sensor-0): got %q", got)
	}
	if got := sink.topicFn(dp1); got != "sensors.sensor-1" {
		t.Errorf("topicFn(sensor-1): got %q", got)
	}
}

func TestNewKafkaSinkCreates(t *testing.T) {
	// kafka.Writer is lazy — no connection is made at construction time.
	sink, err := NewKafkaSink("localhost:9092", "genx", "", "", false, false, JSONRenderer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Close on an unconnected writer should succeed.
	if err := sink.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestNewKafkaSinkWithAuth(t *testing.T) {
	sink, err := NewKafkaSink("localhost:9092", "genx", "alice", "secret", false, false, JSONRenderer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sink.Close()
}

func TestNewKafkaSinkWithTLS(t *testing.T) {
	sink, err := NewKafkaSink("localhost:9092", "genx", "", "", true, false, JSONRenderer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sink.Close()
}

func TestNewKafkaSinkTLSInsecure(t *testing.T) {
	sink, err := NewKafkaSink("localhost:9092", "genx", "", "", false, true, JSONRenderer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sink.Close()
}
