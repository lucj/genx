package main

import "testing"

func TestSeedReproducibility(t *testing.T) {
	r1 := seededRand(42)
	v1 := r1.Float64()

	r2 := seededRand(42)
	v2 := r2.Float64()

	if v1 != v2 {
		t.Errorf("same seed should produce same value: %f != %f", v1, v2)
	}
}

func TestDifferentSeedsDifferentValues(t *testing.T) {
	r1 := seededRand(1)
	v1 := r1.Float64()

	r2 := seededRand(2)
	v2 := r2.Float64()

	if v1 == v2 {
		t.Error("different seeds should produce different values")
	}
}
