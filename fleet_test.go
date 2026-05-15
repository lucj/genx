package main

import (
	"context"
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
	runRealtime(context.Background(), fns, sink, devices, 2, 1) // 2 points per device, 1 s step

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

// Verify that cancelling the context stops realtime emission early.
func TestRunRealtimeCancellation(t *testing.T) {
	sink := &captureSink{}
	ctx, cancel := context.WithCancel(context.Background())

	fns := []func(float64) float64{func(x float64) float64 { return 1.0 }}
	devices := []string{"sensor-0"}

	// Cancel before the first tick (step = 10 s, so the goroutine will block on the ticker).
	cancel()
	runRealtime(ctx, fns, sink, devices, 100, 10)

	// No points should have been sent since the context was already cancelled.
	if len(sink.points) != 0 {
		t.Errorf("expected 0 points after immediate cancellation, got %d", len(sink.points))
	}
}

func TestRunRealtimeMultiEmitsPoints(t *testing.T) {
	sink := &captureSink{}
	fieldFns := map[string]func(float64) float64{
		"temperature": func(x float64) float64 { return 22.0 },
		"humidity":    func(x float64) float64 { return 60.0 },
	}
	scales := []float64{1.0, 1.0}
	devices := []string{"sensor-0", "sensor-1"}

	runRealtimeMulti(context.Background(), fieldFns, scales, 0, sink, devices, 2, 1)

	if len(sink.points) != 4 {
		t.Fatalf("expected 4 points (2 devices × 2 steps), got %d", len(sink.points))
	}
	for _, dp := range sink.points {
		if dp.Fields == nil {
			t.Fatalf("expected Fields to be populated, got nil for device %s", dp.Device)
		}
		if dp.Fields["temperature"] != 22.0 {
			t.Errorf("expected temperature 22.0, got %v", dp.Fields["temperature"])
		}
		if dp.Fields["humidity"] != 60.0 {
			t.Errorf("expected humidity 60.0, got %v", dp.Fields["humidity"])
		}
	}
}

func TestRunRealtimeMultiCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sink := &captureSink{}
	fieldFns := map[string]func(float64) float64{
		"temperature": func(x float64) float64 { return 22.0 },
	}
	runRealtimeMulti(ctx, fieldFns, []float64{1.0}, 0, sink, []string{"sensor-0"}, 100, 10)

	if len(sink.points) != 0 {
		t.Errorf("expected 0 points after immediate cancellation, got %d", len(sink.points))
	}
}

func TestEvalFields(t *testing.T) {
	fieldFns := map[string]func(float64) float64{
		"temp":  func(x float64) float64 { return 20.0 },
		"humid": func(x float64) float64 { return 50.0 },
	}

	fields := evalFields(fieldFns, 1.0, 0, 0)
	if fields["temp"] != 20.0 {
		t.Errorf("expected temp=20.0, got %v", fields["temp"])
	}
	if fields["humid"] != 50.0 {
		t.Errorf("expected humid=50.0, got %v", fields["humid"])
	}

	// Scale applies multiplicatively.
	fields = evalFields(fieldFns, 2.0, 0, 0)
	if fields["temp"] != 40.0 {
		t.Errorf("expected temp=40.0 with scale=2, got %v", fields["temp"])
	}

	// Noise causes values to deviate from the deterministic result.
	seen := map[float64]bool{}
	for i := 0; i < 20; i++ {
		f := evalFields(fieldFns, 1.0, 0.1, 0)
		seen[f["temp"]] = true
	}
	if len(seen) == 1 {
		t.Error("noise should produce varying values, but all were identical")
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
