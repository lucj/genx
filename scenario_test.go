package main

import (
	"context"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestRunScenarioTwoPhases(t *testing.T) {
	sink := &captureSink{}
	devices := []string{"sensor-0", "sensor-1"}
	v := &cliFlags{
		curveType:     "cos",
		cosPeriod:     "1d",
		dutyCycle:     0.5,
		cosMin:        10,
		cosMax:        25,
		anomalyFactor: 3,
		step:          "1m",
	}
	phases := []PhaseConfig{
		{Duration: "10m", Type: "cos", Min: ptr(20.0), Max: ptr(25.0)},
		{Duration: "5m", Type: "cos", Min: ptr(30.0), Max: ptr(50.0)},
	}

	err := runScenario(context.Background(), newRand(), v, phases, sink, devices, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Phase 1: 10m / 1m = 10 points per device × 2 = 20
	// Phase 2: 5m / 1m = 5 points per device × 2 = 10
	if len(sink.points) != 30 {
		t.Errorf("expected 30 points (2 phases × 2 devices), got %d", len(sink.points))
	}
	counts := map[string]int{}
	for _, dp := range sink.points {
		counts[dp.Device]++
	}
	for _, dev := range devices {
		if counts[dev] != 15 {
			t.Errorf("device %q: expected 15 points, got %d", dev, counts[dev])
		}
	}
}

func TestRunScenarioCancellation(t *testing.T) {
	sink := &captureSink{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Duration: "1h", Type: "cos"},
		{Duration: "1h", Type: "cos"},
	}

	err := runScenario(ctx, newRand(), v, phases, sink, []string{"dev"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.points) != 0 {
		t.Errorf("expected 0 points after cancellation, got %d", len(sink.points))
	}
}

func TestRunScenarioDropoutPhase(t *testing.T) {
	sink := &captureSink{}
	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Duration: "10m", Type: "cos"},
		{Duration: "5m", DropoutRate: ptr(1.0)}, // total connectivity loss
		{Duration: "10m", Type: "cos"},
	}

	err := runScenario(context.Background(), seededRand(1), v, phases, sink, []string{"sensor-0"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Phase 1: 10 points, phase 2: 0 points, phase 3: 10 points
	if len(sink.points) != 20 {
		t.Errorf("expected 20 points (dropout phase skipped), got %d", len(sink.points))
	}
}

func TestRunScenarioTimestampsContinuous(t *testing.T) {
	sink := &captureSink{}
	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Duration: "3m", Type: "cos"},
		{Duration: "3m", Type: "cos"},
	}
	start := int64(1_000_000)

	err := runScenario(context.Background(), newRand(), v, phases, sink, []string{"dev"}, start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect 6 points: timestamps 1000000, 1000060, 1000120, 1000180, 1000240, 1000300
	if len(sink.points) != 6 {
		t.Fatalf("expected 6 points, got %d", len(sink.points))
	}
	for i, dp := range sink.points {
		expected := start + int64(i*60)
		if dp.Timestamp != expected {
			t.Errorf("point %d: expected timestamp %d, got %d", i, expected, dp.Timestamp)
		}
	}
}

func TestRunScenarioMissingDuration(t *testing.T) {
	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Type: "cos"}, // no duration
	}
	err := runScenario(context.Background(), newRand(), v, phases, &captureSink{}, []string{"dev"}, 0)
	if err == nil {
		t.Error("expected error for missing duration, got nil")
	}
}

func TestRunScenarioPhaseStepOverride(t *testing.T) {
	sink := &captureSink{}
	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Duration: "10m", Step: "2m"}, // 5 points instead of 10
	}

	err := runScenario(context.Background(), newRand(), v, phases, sink, []string{"dev"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.points) != 5 {
		t.Errorf("expected 5 points (step override 2m), got %d", len(sink.points))
	}
}

func TestRunScenarioGeoPhase(t *testing.T) {
	sink := &captureSink{}
	v := &cliFlags{
		curveType:  "geo",
		geoLat:     48.8566,
		geoLon:     2.3522,
		geoBearing: 0,
		geoSpeed:   10,
		geoDrift:   5,
		step:       "1m",
	}
	phases := []PhaseConfig{
		{Duration: "5m", Type: "geo"},
	}

	err := runScenario(context.Background(), newRand(), v, phases, sink, []string{"truck-0"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(sink.points))
	}
	for _, dp := range sink.points {
		if dp.Fields == nil {
			t.Fatal("expected Fields map for geo phase")
		}
		if _, ok := dp.Fields["lat"]; !ok {
			t.Error("missing lat field")
		}
	}
}

func TestRunScenarioMultiFieldPhase(t *testing.T) {
	sink := &captureSink{}
	v := &cliFlags{curveType: "cos", cosPeriod: "1d", dutyCycle: 0.5, cosMin: 10, cosMax: 25, anomalyFactor: 3, step: "1m"}
	phases := []PhaseConfig{
		{Duration: "5m", Type: "cos", Min: ptr(20.0), Max: ptr(25.0)},
		{Duration: "5m", Fields: map[string]FieldConfig{
			"temperature": {Type: "cos", Min: ptr(18.0), Max: ptr(26.0)},
			"humidity":    {Type: "cos", Min: ptr(40.0), Max: ptr(80.0)},
		}},
		{Duration: "5m", Type: "cos", Min: ptr(20.0), Max: ptr(25.0)},
	}

	err := runScenario(context.Background(), newRand(), v, phases, sink, []string{"sensor"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 5 + 5 + 5 = 15 points for 1 device
	if len(sink.points) != 15 {
		t.Fatalf("expected 15 points, got %d", len(sink.points))
	}

	// Points 5–9 (phase 2) must carry Fields; the rest must carry Value.
	for i, dp := range sink.points {
		if i >= 5 && i < 10 {
			if dp.Fields == nil {
				t.Errorf("point %d: expected Fields for multi-field phase, got nil", i)
				continue
			}
			if _, ok := dp.Fields["temperature"]; !ok {
				t.Errorf("point %d: missing temperature field", i)
			}
			if _, ok := dp.Fields["humidity"]; !ok {
				t.Errorf("point %d: missing humidity field", i)
			}
		} else {
			if dp.Value == nil {
				t.Errorf("point %d: expected Value for single-field phase, got nil", i)
			}
		}
	}
}

func TestResolvePhaseInheritsGlobals(t *testing.T) {
	v := &cliFlags{
		curveType:     "walk",
		noise:         0.05,
		anomalyRate:   0.01,
		anomalyFactor: 3.0,
		dropoutRate:   0.0,
		cosPeriod:     "12h",
		step:          "30s",
	}
	phase := PhaseConfig{Duration: "10m", Type: "cos", Min: ptr(20.0), Max: ptr(30.0)}

	pp, err := resolvePhase(v, phase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pp.curveType != "cos" {
		t.Errorf("expected curveType cos, got %s", pp.curveType)
	}
	if pp.cosMin != 20.0 {
		t.Errorf("expected cosMin 20.0, got %f", pp.cosMin)
	}
	if pp.noise != 0.05 {
		t.Errorf("expected noise 0.05 inherited from global, got %f", pp.noise)
	}
	if pp.durationSeconds != 600 {
		t.Errorf("expected 600s duration, got %d", pp.durationSeconds)
	}
	if pp.stepSeconds != 30 {
		t.Errorf("expected 30s step inherited from global, got %d", pp.stepSeconds)
	}
}
