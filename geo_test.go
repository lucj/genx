package main

import (
	"math"
	"testing"
)

func TestGeoWalkerStepNorth(t *testing.T) {
	// Paris, heading north, no drift → should move north only.
	w := NewGeoWalker(48.8566, 2.3522, 0, 10, 0)
	lat0, lon0 := w.lat, w.lon

	lat1, lon1 := w.Step(newRand(), 60) // 60 s → 600 m north

	if lat1 <= lat0 {
		t.Errorf("expected lat to increase going north, got lat0=%v lat1=%v", lat0, lat1)
	}
	if math.Abs(lon1-lon0) > 1e-6 {
		t.Errorf("expected lon to be unchanged going north, got delta=%v", lon1-lon0)
	}

	dist := haversineDistM(lat0, lon0, lat1, lon1)
	if math.Abs(dist-600) > 1 { // within 1 m
		t.Errorf("expected ~600 m step, got %.2f m", dist)
	}
}

func TestGeoWalkerStepEast(t *testing.T) {
	w := NewGeoWalker(0, 0, 90, 10, 0) // equator heading east
	lat0, lon0 := w.lat, w.lon

	lat1, lon1 := w.Step(newRand(), 60)

	if lon1 <= lon0 {
		t.Errorf("expected lon to increase going east, got lon0=%v lon1=%v", lon0, lon1)
	}
	if math.Abs(lat1-lat0) > 1e-6 {
		t.Errorf("expected lat unchanged going east on equator, got delta=%v", lat1-lat0)
	}
}

func TestGeoWalkerDistanceAccuracy(t *testing.T) {
	speeds := []float64{1, 5, 50, 100}
	stepSeconds := 30

	for _, speed := range speeds {
		w := NewGeoWalker(48.8566, 2.3522, 45, speed, 0)
		lat0, lon0 := w.lat, w.lon
		lat1, lon1 := w.Step(newRand(), stepSeconds)

		want := speed * float64(stepSeconds)
		got := haversineDistM(lat0, lon0, lat1, lon1)
		if math.Abs(got-want)/want > 0.001 { // within 0.1%
			t.Errorf("speed=%.0f: expected %.2f m, got %.2f m", speed, want, got)
		}
	}
}

func TestGeoWalkerBearingNormalisation(t *testing.T) {
	// Start with bearing just above 360; after drift it could wrap.
	w := NewGeoWalker(0, 0, 350, 1, 20)
	rng := seededRand(42)
	for i := 0; i < 100; i++ {
		w.Step(rng, 1)
		if w.bearing < 0 || w.bearing >= 360 {
			t.Errorf("bearing out of [0,360): %v after step %d", w.bearing, i)
		}
	}
}

func TestGeoWalkerNoDriftStraightLine(t *testing.T) {
	w := NewGeoWalker(48.8566, 2.3522, 0, 10, 0)
	rng := seededRand(1)

	var lats []float64
	for i := 0; i < 5; i++ {
		lat, _ := w.Step(rng, 60)
		lats = append(lats, lat)
	}

	// Each step should increase latitude by the same amount (straight north).
	delta0 := lats[1] - lats[0]
	for i := 2; i < len(lats); i++ {
		delta := lats[i] - lats[i-1]
		if math.Abs(delta-delta0)/delta0 > 0.001 {
			t.Errorf("step %d: expected uniform delta %.6f, got %.6f", i, delta0, delta)
		}
	}
}

func TestGeoWalkerLongitudeWrap(t *testing.T) {
	// Start near 180° longitude heading east; should wrap to negative.
	w := NewGeoWalker(0, 179.9999, 90, 1000, 0)
	_, lon := w.Step(newRand(), 60)
	if lon > 180 || lon < -180 {
		t.Errorf("longitude out of [-180,180]: %v", lon)
	}
}

func TestGeoWalkerMultipleDevicesIndependent(t *testing.T) {
	w1 := NewGeoWalker(48.8566, 2.3522, 0, 10, 0)
	w2 := NewGeoWalker(48.8566, 2.3522, 90, 10, 0)
	rng := seededRand(1)

	lat1, lon1 := w1.Step(rng, 60)
	lat2, lon2 := w2.Step(rng, 60)

	if lat1 == lat2 && lon1 == lon2 {
		t.Error("walkers with different bearings should produce different positions")
	}
}
