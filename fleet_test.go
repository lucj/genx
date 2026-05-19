package main

import (
	"context"
	"fmt"
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
	runBatch(newRand(), []func(float64) float64{fn}, sink, []string{"dev"}, time.Now().Unix(), 3, 60, 0, 0)

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
	runBatch(newRand(), fns, sink, devices, time.Now().Unix(), 4, 60, 0, 0)

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
	runRealtime(context.Background(), newRand(), fns, sink, devices, 2, 1, 0, 0)

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

func TestRunRealtimeCancellation(t *testing.T) {
	sink := &captureSink{}
	ctx, cancel := context.WithCancel(context.Background())

	fns := []func(float64) float64{func(x float64) float64 { return 1.0 }}
	devices := []string{"sensor-0"}

	cancel()
	runRealtime(ctx, newRand(), fns, sink, devices, 100, 10, 0, 0)

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

	runRealtimeMulti(context.Background(), newRand(), fieldFns, scales, 0, 0, 0, 0, sink, devices, 2, 1, 0)

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
	runRealtimeMulti(ctx, newRand(), fieldFns, []float64{1.0}, 0, 0, 0, 0, sink, []string{"sensor-0"}, 100, 10, 0)

	if len(sink.points) != 0 {
		t.Errorf("expected 0 points after immediate cancellation, got %d", len(sink.points))
	}
}

func TestEvalFields(t *testing.T) {
	fieldFns := map[string]func(float64) float64{
		"temp":  func(x float64) float64 { return 20.0 },
		"humid": func(x float64) float64 { return 50.0 },
	}

	fields := evalFields(newRand(), fieldFns, 1.0, 0, 0, 0, 0)
	if fields["temp"] != 20.0 {
		t.Errorf("expected temp=20.0, got %v", fields["temp"])
	}
	if fields["humid"] != 50.0 {
		t.Errorf("expected humid=50.0, got %v", fields["humid"])
	}

	fields = evalFields(newRand(), fieldFns, 2.0, 0, 0, 0, 0)
	if fields["temp"] != 40.0 {
		t.Errorf("expected temp=40.0 with scale=2, got %v", fields["temp"])
	}

	seen := map[float64]bool{}
	for i := 0; i < 20; i++ {
		f := evalFields(newRand(), fieldFns, 1.0, 0.1, 0, 0, 0)
		seen[f["temp"]] = true
	}
	if len(seen) == 1 {
		t.Error("noise should produce varying values, but all were identical")
	}
}

func TestDropoutRateSkipsPoints(t *testing.T) {
	fn := func(x float64) float64 { return 1.0 }
	devices := []string{"dev"}
	fns := []func(float64) float64{fn}

	sink := &captureSink{}
	runBatch(seededRand(1), fns, sink, devices, time.Now().Unix(), 100, 1, 1.0, 0)
	if len(sink.points) != 0 {
		t.Errorf("dropout-rate=1.0: expected 0 points, got %d", len(sink.points))
	}

	sink = &captureSink{}
	runBatch(seededRand(1), fns, sink, devices, time.Now().Unix(), 100, 1, 0.0, 0)
	if len(sink.points) != 100 {
		t.Errorf("dropout-rate=0: expected 100 points, got %d", len(sink.points))
	}

	sink = &captureSink{}
	runBatch(seededRand(1), fns, sink, devices, time.Now().Unix(), 1000, 1, 0.5, 0)
	if len(sink.points) < 400 || len(sink.points) > 600 {
		t.Errorf("dropout-rate=0.5: expected ~500 points, got %d", len(sink.points))
	}
}

func TestDropoutRateMultiField(t *testing.T) {
	fieldFns := map[string]func(float64) float64{
		"temperature": func(x float64) float64 { return 22.0 },
	}
	scales := []float64{1.0}
	devices := []string{"sensor-0"}

	sink := &captureSink{}
	runBatchMulti(seededRand(1), fieldFns, scales, 0, 0, 0, 1.0, sink, devices, time.Now().Unix(), 100, 1, 0)
	if len(sink.points) != 0 {
		t.Errorf("multi dropout-rate=1.0: expected 0 points, got %d", len(sink.points))
	}
}

func TestSpreadProducesDifferentValues(t *testing.T) {
	sink := &captureSink{}
	base := 100.0
	fns := []func(float64) float64{
		func(x float64) float64 { return base * 0.9 },
		func(x float64) float64 { return base * 1.1 },
	}
	devices := []string{"sensor-0", "sensor-1"}
	runBatch(newRand(), fns, sink, devices, time.Now().Unix(), 1, 60, 0, 0)

	if *sink.points[0].Value == *sink.points[1].Value {
		t.Error("expected different values for devices with different scale factors")
	}
}

// --- statsSink tests ---

func TestStatsSinkCountsSends(t *testing.T) {
	inner := &captureSink{}
	s := &statsSink{inner: inner}

	for i := 0; i < 5; i++ {
		v := 1.0
		s.Send(DataPoint{Device: "dev", Timestamp: int64(i), Value: &v})
	}

	if got := s.sent.Load(); got != 5 {
		t.Errorf("expected 5 sent, got %d", got)
	}
	if got := s.errors.Load(); got != 0 {
		t.Errorf("expected 0 errors, got %d", got)
	}
	if len(inner.points) != 5 {
		t.Errorf("expected 5 points in inner sink, got %d", len(inner.points))
	}
}

type errorSink struct{}

func (e *errorSink) Send(DataPoint) error { return fmt.Errorf("send failed") }
func (e *errorSink) Close() error         { return nil }

func TestStatsSinkCountsErrors(t *testing.T) {
	s := &statsSink{inner: &errorSink{}}
	v := 1.0
	s.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v})
	s.Send(DataPoint{Device: "dev", Timestamp: 2, Value: &v})

	if got := s.sent.Load(); got != 0 {
		t.Errorf("expected 0 sent, got %d", got)
	}
	if got := s.errors.Load(); got != 2 {
		t.Errorf("expected 2 errors, got %d", got)
	}
}

// --- Rate cap tests ---

func TestRateCapBatch(t *testing.T) {
	sink := &captureSink{}
	fn := func(x float64) float64 { return 1.0 }
	fns := []func(float64) float64{fn}

	// rate=20 for 5 points → 4 inter-point delays of 50 ms each = 200 ms minimum.
	start := time.Now()
	runBatch(newRand(), fns, sink, []string{"dev"}, time.Now().Unix(), 5, 1, 0, 20)
	elapsed := time.Since(start)

	if len(sink.points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(sink.points))
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("rate=20 for 5 points should take ≥150 ms, took %v", elapsed)
	}
}

func TestRateCapZeroIsUnlimited(t *testing.T) {
	sink := &captureSink{}
	fn := func(x float64) float64 { return 1.0 }
	fns := []func(float64) float64{fn}

	// With rate=0 (unlimited) 1000 points should complete almost instantly.
	start := time.Now()
	runBatch(newRand(), fns, sink, []string{"dev"}, time.Now().Unix(), 1000, 1, 0, 0)
	elapsed := time.Since(start)

	if len(sink.points) != 1000 {
		t.Fatalf("expected 1000 points, got %d", len(sink.points))
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("rate=0 (unlimited) should be fast, took %v", elapsed)
	}
}

// --- Geo runner tests ---

func TestRunBatchGeoEmitsLatLon(t *testing.T) {
	sink := &captureSink{}
	walker := NewGeoWalker(48.8566, 2.3522, 0, 10, 0)
	runBatchGeo(newRand(), []*GeoWalker{walker}, sink, []string{"truck-0"}, time.Now().Unix(), 5, 60, 0, 0)

	if len(sink.points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(sink.points))
	}
	for _, dp := range sink.points {
		if dp.Fields == nil {
			t.Fatal("expected Fields map, got nil")
		}
		if _, ok := dp.Fields["lat"]; !ok {
			t.Error("missing lat field")
		}
		if _, ok := dp.Fields["lon"]; !ok {
			t.Error("missing lon field")
		}
		if dp.Value != nil {
			t.Error("Value should be nil for geo points")
		}
	}
}

func TestRunBatchGeoFleet(t *testing.T) {
	sink := &captureSink{}
	walkers := []*GeoWalker{
		NewGeoWalker(48.8566, 2.3522, 0, 10, 0),
		NewGeoWalker(51.5074, -0.1278, 90, 5, 0),
	}
	devices := []string{"truck-0", "truck-1"}
	runBatchGeo(newRand(), walkers, sink, devices, time.Now().Unix(), 3, 60, 0, 0)

	if len(sink.points) != 6 {
		t.Fatalf("expected 6 points (2 devices × 3 steps), got %d", len(sink.points))
	}
	counts := map[string]int{}
	for _, dp := range sink.points {
		counts[dp.Device]++
	}
	for _, dev := range devices {
		if counts[dev] != 3 {
			t.Errorf("device %q: expected 3 points, got %d", dev, counts[dev])
		}
	}
}

func TestRunRealtimeGeoCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sink := &captureSink{}
	walker := NewGeoWalker(48.8566, 2.3522, 0, 10, 0)
	runRealtimeGeo(ctx, newRand(), []*GeoWalker{walker}, sink, []string{"truck-0"}, 100, 10, 0, 0)

	if len(sink.points) != 0 {
		t.Errorf("expected 0 points after cancellation, got %d", len(sink.points))
	}
}
