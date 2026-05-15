package main

import (
	"math"
	"math/rand/v2"
)

func GetLinear(first, last float64, start int64, durationSeconds int) func(float64) float64 {
	A := (last - first) / float64(durationSeconds)
	B := float64(start)
	return func(x float64) float64 {
		return A*(x-B) + first
	}
}

func GetCosinus(min, max float64, periodSeconds int) func(float64) float64 {
	A := (max - min) / 2
	B := 2 * math.Pi / float64(periodSeconds)
	D := min + A
	return func(x float64) float64 {
		return A*math.Cos(B*x) + D
	}
}

func GetLog(start int64) func(float64) float64 {
	B := float64(start)
	return func(x float64) float64 {
		delta := x - B
		if delta <= 0 {
			return 0
		}
		return math.Log(delta)
	}
}

func GetExp(start int64, durationSeconds int) func(float64) float64 {
	B := 1 / float64(durationSeconds)
	C := float64(start)
	return func(x float64) float64 {
		return math.Exp(B * (x - C))
	}
}

// GetRandomWalk returns a stateful closure that drifts by a random delta each call.
// stepSize controls the magnitude, bias adds a fixed directional drift per call
// (negative = downward), and min/max clamp the output when max > min.
// The timestamp argument is ignored — the value depends only on call history.
func GetRandomWalk(rng *rand.Rand, start, stepSize, bias, min, max float64) func(float64) float64 {
	current := start
	clamp := max > min
	return func(_ float64) float64 {
		current += stepSize*(2*rng.Float64()-1) + bias
		if clamp {
			if current < min {
				current = min
			} else if current > max {
				current = max
			}
		}
		return current
	}
}

// WithNoise wraps fn with multiplicative random jitter in ±noise ratio.
// WithNoise(rng, fn, 0) returns fn unchanged.
func WithNoise(rng *rand.Rand, fn func(float64) float64, noise float64) func(float64) float64 {
	if noise <= 0 {
		return fn
	}
	return func(x float64) float64 {
		return fn(x) * (1 + noise*(2*rng.Float64()-1))
	}
}
