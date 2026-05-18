package main

import (
	"math"
	"testing"
	"time"
)

func TestGetLinear(t *testing.T) {
	start := time.Now().Unix()
	duration := 3600
	fn := GetLinear(0, 100, start, duration)

	if got := fn(float64(start)); math.Abs(got) > 1e-9 {
		t.Errorf("at start: got %f, want 0", got)
	}
	if got := fn(float64(start + int64(duration))); math.Abs(got-100) > 1e-9 {
		t.Errorf("at end: got %f, want 100", got)
	}
	if got := fn(float64(start + int64(duration/2))); math.Abs(got-50) > 1e-9 {
		t.Errorf("at midpoint: got %f, want 50", got)
	}
}

func TestGetCosinus(t *testing.T) {
	min, max := 10.0, 25.0
	fn := GetCosinus(min, max, 86400)

	start := time.Now().Unix()
	for i := 0; i < 100; i++ {
		x := float64(start + int64(i*864))
		v := fn(x)
		if v < min-1e-9 || v > max+1e-9 {
			t.Errorf("value %f out of range [%f, %f] at step %d", v, min, max, i)
		}
	}
}

func TestGetLog(t *testing.T) {
	start := time.Now().Unix()
	fn := GetLog(start)

	// At exactly start, delta == 0 — should return 0, not -Inf.
	if v := fn(float64(start)); v != 0 {
		t.Errorf("GetLog at start: expected 0, got %f", v)
	}
	v1 := fn(float64(start + 1))
	v2 := fn(float64(start + 2))
	if math.IsNaN(v1) || math.IsInf(v1, 0) {
		t.Errorf("log at start+1 is not finite: %f", v1)
	}
	if v2 <= v1 {
		t.Errorf("log should be increasing: v1=%f, v2=%f", v1, v2)
	}
}

func TestGetRandomWalkAdvances(t *testing.T) {
	rng := newRand()
	fn := GetRandomWalk(rng, 100, 1.0, 0, 0, 0)
	seen := map[float64]bool{}
	for range 20 {
		seen[fn(0)] = true
	}
	if len(seen) == 1 {
		t.Error("random walk should produce different values across calls")
	}
}

func TestGetRandomWalkDownwardBias(t *testing.T) {
	rng := newRand()
	fn := GetRandomWalk(rng, 100, 0.1, -1.0, 0, 0)
	for range 50 {
		fn(0)
	}
	if fn(0) >= 100 {
		t.Error("downward bias should decrease value over time")
	}
}

func TestGetRandomWalkClampedRange(t *testing.T) {
	rng := newRand()
	fn := GetRandomWalk(rng, 50, 100, -10, 0, 100)
	for range 200 {
		v := fn(0)
		if v < 0 || v > 100 {
			t.Errorf("value %f outside clamped range [0, 100]", v)
		}
	}
}

func TestGetRandomWalkReproducibleWithSeed(t *testing.T) {
	r1 := seededRand(42)
	fn1 := GetRandomWalk(r1, 100, 1.0, -0.1, 0, 100)
	vals := make([]float64, 10)
	for i := range vals {
		vals[i] = fn1(0)
	}

	r2 := seededRand(42)
	fn2 := GetRandomWalk(r2, 100, 1.0, -0.1, 0, 100)
	for i, want := range vals {
		if got := fn2(0); got != want {
			t.Errorf("step %d: seed 42 should be reproducible: got %f, want %f", i, got, want)
		}
	}
}

func TestWithNoiseZero(t *testing.T) {
	rng := newRand()
	base := func(x float64) float64 { return 100.0 }
	fn := WithNoise(rng, base, 0)
	if fn(0) != 100.0 {
		t.Errorf("zero noise should return base value unchanged, got %f", fn(0))
	}
}

func TestWithNoiseStaysInRange(t *testing.T) {
	rng := newRand()
	base := func(x float64) float64 { return 100.0 }
	fn := WithNoise(rng, base, 0.2)
	for i := range 200 {
		v := fn(float64(i))
		if v < 80-1e-9 || v > 120+1e-9 {
			t.Errorf("value %f outside expected range [80, 120] with noise=0.2", v)
		}
	}
}

func TestWithNoiseAddsVariation(t *testing.T) {
	rng := newRand()
	base := func(x float64) float64 { return 100.0 }
	fn := WithNoise(rng, base, 0.5)
	seen := map[float64]bool{}
	for i := range 20 {
		seen[fn(float64(i))] = true
	}
	if len(seen) == 1 {
		t.Error("noise should produce different values across samples")
	}
}

func TestWithAnomalyZeroRate(t *testing.T) {
	rng := seededRand(1)
	base := func(_ float64) float64 { return 10.0 }
	fn := WithAnomaly(rng, base, 0, 5)
	for range 20 {
		if fn(0) != 10.0 {
			t.Error("zero rate: value should always be 10.0")
		}
	}
}

func TestWithAnomalyFactorOne(t *testing.T) {
	rng := seededRand(1)
	base := func(_ float64) float64 { return 10.0 }
	fn := WithAnomaly(rng, base, 1.0, 1.0)
	for range 20 {
		if fn(0) != 10.0 {
			t.Error("factor=1: value should always be 10.0")
		}
	}
}

func TestWithAnomalyAlwaysTriggered(t *testing.T) {
	rng := seededRand(42)
	base := func(_ float64) float64 { return 10.0 }
	fn := WithAnomaly(rng, base, 1.0, 5.0)
	for range 50 {
		v := fn(0)
		if math.Abs(v-50) > 1e-9 && math.Abs(v-2) > 1e-9 {
			t.Errorf("anomaly value %f is neither spike (50) nor drop (2)", v)
		}
	}
}

func TestWithAnomalyProducesOutliers(t *testing.T) {
	rng := seededRand(7)
	base := func(_ float64) float64 { return 10.0 }
	fn := WithAnomaly(rng, base, 0.5, 10.0)
	outliers := 0
	for range 200 {
		v := fn(0)
		if v != 10.0 {
			outliers++
		}
	}
	if outliers == 0 {
		t.Error("expected some anomalies with rate=0.5, got none")
	}
}

func TestGetExp(t *testing.T) {
	start := time.Now().Unix()
	duration := 3600
	fn := GetExp(start, duration)

	v1 := fn(float64(start))
	v2 := fn(float64(start + 1800))
	v3 := fn(float64(start + int64(duration)))
	if v1 >= v2 || v2 >= v3 {
		t.Errorf("exp should be increasing: v1=%f v2=%f v3=%f", v1, v2, v3)
	}
}
