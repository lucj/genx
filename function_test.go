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
	// Midpoint should be ~50
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

	v1 := fn(float64(start + 1))
	v2 := fn(float64(start + 2))
	if math.IsNaN(v1) || math.IsInf(v1, 0) {
		t.Errorf("log at start+1 is not finite: %f", v1)
	}
	if v2 <= v1 {
		t.Errorf("log should be increasing: v1=%f, v2=%f", v1, v2)
	}
}

func TestWithNoiseZero(t *testing.T) {
	base := func(x float64) float64 { return 100.0 }
	fn := WithNoise(base, 0)
	if fn(0) != 100.0 {
		t.Errorf("zero noise should return base value unchanged, got %f", fn(0))
	}
}

func TestWithNoiseStaysInRange(t *testing.T) {
	base := func(x float64) float64 { return 100.0 }
	noise := 0.2 // ±20%
	fn := WithNoise(base, noise)
	for i := range 200 {
		v := fn(float64(i))
		if v < 80-1e-9 || v > 120+1e-9 {
			t.Errorf("value %f outside expected range [80, 120] with noise=0.2", v)
		}
	}
}

func TestWithNoiseAddsVariation(t *testing.T) {
	base := func(x float64) float64 { return 100.0 }
	fn := WithNoise(base, 0.5)
	seen := map[float64]bool{}
	for i := range 20 {
		seen[fn(float64(i))] = true
	}
	if len(seen) == 1 {
		t.Error("noise should produce different values across samples")
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
