package main

import (
	"sync"
	"testing"
	"time"
)

type captureSink struct {
	mu     sync.Mutex
	points []DataPoint
}

func (s *captureSink) Send(dp DataPoint) error {
	s.mu.Lock()
	s.points = append(s.points, dp)
	s.mu.Unlock()
	return nil
}

func (s *captureSink) Close() error { return nil }

func TestRunBatchSingleDevice(t *testing.T) {
	sink := &captureSink{}
	fn := func(x float64) float64 { return x }
	runBatch([]func(float64) float64{fn}, sink, []string{"dev"}, time.Now().Unix(), 3, 60)

	if len(sink.points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(sink.points))
	}
	for _, dp := range sink.points {
		if dp.Device != "dev" {
			t.Errorf("expected device %q, got %q", "dev", dp.Device)
		}
	}
}

func TestRunBatchFleet(t *testing.T) {
	sink := &captureSink{}
	devices := []string{"sensor-0", "sensor-1", "sensor-2"}
	fns := []func(float64) float64{
		func(x float64) float64 { return x },
		func(x float64) float64 { return x * 2 },
		func(x float64) float64 { return x * 3 },
	}
	runBatch(fns, sink, devices, time.Now().Unix(), 4, 60)

	if len(sink.points) != 12 {
		t.Fatalf("expected 12 points (3 devices × 4 steps), got %d", len(sink.points))
	}
	counts := map[string]int{}
	for _, dp := range sink.points {
		counts[dp.Device]++
	}
	for _, dev := range devices {
		if counts[dev] != 4 {
			t.Errorf("device %q: expected 4 points, got %d", dev, counts[dev])
		}
	}
}

func TestRunRealtimeFleet(t *testing.T) {
	sink := &captureSink{}
	devices := []string{"sensor-0", "sensor-1"}
	fns := []func(float64) float64{
		func(x float64) float64 { return 1.0 },
		func(x float64) float64 { return 2.0 },
	}
	runRealtime(fns, sink, devices, 2, 1) // 2 points per device, 1 s step

	if len(sink.points) != 4 {
		t.Fatalf("expected 4 points (2 devices × 2 steps), got %d", len(sink.points))
	}
	counts := map[string]int{}
	for _, dp := range sink.points {
		counts[dp.Device]++
	}
	for _, dev := range devices {
		if counts[dev] != 2 {
			t.Errorf("device %q: expected 2 points, got %d", dev, counts[dev])
		}
	}
}

// Verify that values from different devices differ when spread > 0.
func TestSpreadProducesDifferentValues(t *testing.T) {
	sink := &captureSink{}
	base := 100.0
	fns := []func(float64) float64{
		func(x float64) float64 { return base * 0.9 },
		func(x float64) float64 { return base * 1.1 },
	}
	devices := []string{"sensor-0", "sensor-1"}
	runBatch(fns, sink, devices, time.Now().Unix(), 1, 60)

	if *sink.points[0].Value == *sink.points[1].Value {
		t.Error("expected different values for devices with different scale factors")
	}
}
