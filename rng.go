package main

import "math/rand/v2"

// newRand returns a randomly-seeded RNG for general use.
func newRand() *rand.Rand {
	return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}

// seededRand returns a deterministically-seeded RNG for reproducible output.
func seededRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}
