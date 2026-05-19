package main

import (
	"testing"
)

func TestNewOTLPSinkGRPC(t *testing.T) {
	// gRPC uses lazy connection — constructor succeeds even without a collector.
	sink, err := NewOTLPSink("localhost:4317", false, nil, true, "genx")
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	defer sink.Close() //nolint: errcheck — export will fail, that's expected

	val := 42.0
	if err := sink.Send(DataPoint{Device: "test", Timestamp: 1000, Value: &val}); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

func TestNewOTLPSinkHTTP(t *testing.T) {
	sink, err := NewOTLPSink("localhost:4318", true, nil, true, "genx")
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	defer sink.Close() //nolint: errcheck

	val := 7.5
	if err := sink.Send(DataPoint{Device: "dev-0", Timestamp: 2000, Value: &val}); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

func TestOTLPSinkMultiField(t *testing.T) {
	sink, err := NewOTLPSink("localhost:4317", false, nil, true, "sensors")
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	defer sink.Close() //nolint: errcheck

	dp := DataPoint{
		Device:    "room1",
		Timestamp: 3000,
		Fields:    map[string]float64{"temperature": 22.5, "humidity": 60.0},
	}
	if err := sink.Send(dp); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

func TestOTLPSinkDefaultMetricName(t *testing.T) {
	sink, err := NewOTLPSink("localhost:4317", false, nil, true, "")
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	if sink.metricName != "genx" {
		t.Errorf("expected default metric name 'genx', got %q", sink.metricName)
	}
	sink.Close() //nolint: errcheck
}

func TestOTLPSinkNilValue(t *testing.T) {
	sink, err := NewOTLPSink("localhost:4317", false, nil, true, "genx")
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	defer sink.Close() //nolint: errcheck

	// nil Value — Send should be a no-op, not panic
	if err := sink.Send(DataPoint{Device: "dev", Timestamp: 1, Value: nil}); err != nil {
		t.Errorf("Send with nil Value returned error: %v", err)
	}
}
