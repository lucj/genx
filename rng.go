package main

import "math/rand/v2"

// rng is the shared random source used for noise, spread, and seed calculations.
// It is safe for concurrent use (PCG source uses atomic operations).
var rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

// initRand reseeds rng with a fixed value for reproducible output.
// Must be called before any random values are consumed.
func initRand(seed uint64) {
	rng = rand.New(rand.NewPCG(seed, 0))
}
