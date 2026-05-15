package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildFieldFnLinear(t *testing.T) {
	first, last := 0.0, 100.0
	fc := FieldConfig{Type: "linear", First: &first, Last: &last}
	start := time.Now().Unix()
	fn, err := buildFieldFn(fc, start, 3600)
	if err != nil {
		t.Fatalf("buildFieldFn error: %v", err)
	}
	if got := fn(float64(start)); got < -1e-9 || got > 1e-9 {
		t.Errorf("at start: got %f, want ~0", got)
	}
}

func TestBuildFieldFnCos(t *testing.T) {
	min, max := 10.0, 25.0
	fc := FieldConfig{Type: "cos", Min: &min, Max: &max}
	start := time.Now().Unix()
	fn, err := buildFieldFn(fc, start, 86400)
	if err != nil {
		t.Fatalf("buildFieldFn error: %v", err)
	}
	v := fn(float64(start))
	if v < min-1e-9 || v > max+1e-9 {
		t.Errorf("cos value %f out of range [%f, %f]", v, min, max)
	}
}

func TestBuildFieldFnLog(t *testing.T) {
	fc := FieldConfig{Type: "log"}
	start := time.Now().Unix()
	_, err := buildFieldFn(fc, start, 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFieldFnExp(t *testing.T) {
	fc := FieldConfig{Type: "exp"}
	start := time.Now().Unix()
	_, err := buildFieldFn(fc, start, 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFieldFnWalk(t *testing.T) {
	start := 100.0
	fc := FieldConfig{Type: "walk", WalkStart: &start}
	fn, err := buildFieldFn(fc, time.Now().Unix(), 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Walk should produce a value close to start on the first call
	v := fn(0)
	if v < 90 || v > 110 {
		t.Errorf("first walk value %f unexpectedly far from start 100", v)
	}
}

func TestBuildFieldFnUnknownType(t *testing.T) {
	fc := FieldConfig{Type: "unknown"}
	_, err := buildFieldFn(fc, time.Now().Unix(), 3600)
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

func TestBuildFieldFnInvalidPeriod(t *testing.T) {
	min, max := 0.0, 1.0
	fc := FieldConfig{Type: "cos", Min: &min, Max: &max, Period: "5x"}
	_, err := buildFieldFn(fc, time.Now().Unix(), 3600)
	if err == nil {
		t.Error("expected error for invalid period, got nil")
	}
}

func TestRunBatchMultiFieldsPresent(t *testing.T) {
	sink := &captureSink{}
	fieldFns := map[string]func(float64) float64{
		"temperature": func(x float64) float64 { return 20.0 },
		"humidity":    func(x float64) float64 { return 60.0 },
	}
	scales := []float64{1.0, 1.0}
	devices := []string{"sensor-0", "sensor-1"}
	runBatchMulti(fieldFns, scales, 0, sink, devices, time.Now().Unix(), 3, 60)

	if len(sink.points) != 6 {
		t.Fatalf("expected 6 points (2 devices × 3 steps), got %d", len(sink.points))
	}
	for _, dp := range sink.points {
		if dp.Value != nil {
			t.Error("Value should be nil in multi-field mode")
		}
		if len(dp.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(dp.Fields))
		}
		if _, ok := dp.Fields["temperature"]; !ok {
			t.Error("missing temperature field")
		}
		if _, ok := dp.Fields["humidity"]; !ok {
			t.Error("missing humidity field")
		}
	}
}

func TestMultiFieldJSONOutput(t *testing.T) {
	v := 22.5
	dp := DataPoint{
		Device:    "sensor",
		Timestamp: 1000,
		Fields:    map[string]float64{"temperature": v, "humidity": 60.0},
	}
	b, err := json.Marshal(dp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, ok := out["value"]; ok {
		t.Error("value key should be absent in multi-field JSON")
	}
	if _, ok := out["fields"]; !ok {
		t.Error("fields key should be present in multi-field JSON")
	}
}

func TestSingleFieldJSONOutput(t *testing.T) {
	v := 22.5
	dp := DataPoint{Device: "sensor", Timestamp: 1000, Value: &v}
	b, err := json.Marshal(dp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, ok := out["value"]; !ok {
		t.Error("value key should be present in single-field JSON")
	}
	if _, ok := out["fields"]; ok {
		t.Error("fields key should be absent in single-field JSON")
	}
}
