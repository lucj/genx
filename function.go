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

// GetCosinus returns a cosine wave anchored to start: the curve begins at max
// at t=start, reaches min at t=start+period/2, and returns to max at t=start+period.
func GetCosinus(min, max float64, start int64, periodSeconds int) func(float64) float64 {
	A := (max - min) / 2
	B := 2 * math.Pi / float64(periodSeconds)
	D := min + A
	S := float64(start)
	return func(x float64) float64 {
		return A*math.Cos(B*(x-S)) + D
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

// GetSawtooth returns a function that ramps linearly from min to max over one
// period, then resets — producing a /|/|/| waveform. Phase is anchored to
// start so the wave always begins at min at the start of the run.
func GetSawtooth(min, max float64, start int64, periodSeconds int) func(float64) float64 {
	period := float64(periodSeconds)
	startF := float64(start)
	return func(x float64) float64 {
		phase := math.Mod(x-startF, period) / period
		if phase < 0 {
			phase += 1
		}
		return min + (max-min)*phase
	}
}

// GetSquare returns a function that outputs max during the high portion of each
// period (phase < dutyCycle) and min during the low portion. dutyCycle must be
// in (0, 1); 0.5 gives a symmetric on/off wave.
func GetSquare(min, max float64, start int64, periodSeconds int, dutyCycle float64) func(float64) float64 {
	period := float64(periodSeconds)
	startF := float64(start)
	return func(x float64) float64 {
		phase := math.Mod(x-startF, period) / period
		if phase < 0 {
			phase += 1
		}
		if phase < dutyCycle {
			return max
		}
		return min
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

// WithAnomaly wraps fn so that each call has a `rate` probability of injecting
// an anomaly: a spike (value × factor) or a drop (value / factor), chosen at random.
// WithAnomaly(rng, fn, 0, _) returns fn unchanged.
func WithAnomaly(rng *rand.Rand, fn func(float64) float64, rate, factor float64) func(float64) float64 {
	if rate <= 0 || factor <= 1 {
		return fn
	}
	return func(x float64) float64 {
		v := fn(x)
		if rng.Float64() < rate {
			if rng.Float64() < 0.5 {
				return v * factor
			}
			return v / factor
		}
		return v
	}
}
