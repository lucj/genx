package main

import "testing"

func TestSeedReproducibility(t *testing.T) {
	initRand(42)
	v1 := rng.Float64()

	initRand(42)
	v2 := rng.Float64()

	if v1 != v2 {
		t.Errorf("same seed should produce same value: %f != %f", v1, v2)
	}
}

func TestDifferentSeedsDifferentValues(t *testing.T) {
	initRand(1)
	v1 := rng.Float64()

	initRand(2)
	v2 := rng.Float64()

	if v1 == v2 {
		t.Error("different seeds should produce different values")
	}
}
