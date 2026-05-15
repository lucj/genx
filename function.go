package main

import "math"

func GetLinear(first float64, last float64, start int64, durationSeconds int) func(x float64) float64 {
    // Build function : y = A.(x-B)+C
    fn := func(x float64) float64 {
        A := (last - first) / float64(durationSeconds)
        B := float64(start)
        C := first
        return A * (x - B) + C
    }
    return fn
}

func GetCosinus(min float64, max float64, periodSeconds int) func(x float64) float64 {
    // Build function : y = A.cos(B(x-C))+D
    fn := func(x float64) float64 {
        A := (max - min) / 2
        B := float64(2 * math.Pi) / float64(periodSeconds)
        C := 0.0
        D := min + A
        return A * math.Cos(B * (x - C)) + D
    }
    return fn
}

func GetLog(start int64) func(x float64) float64 {
    // Build function : y = A.ln(x-B)+C
    fn := func(x float64) float64 {
        A := 1.0
        B := float64(start)
        C := 0.0
        return A * math.Log(x-B) + C
    }
    return fn
}

// GetRandomWalk returns a stateful closure that drifts by a random delta each call.
// stepSize controls the magnitude of each random step, bias adds a fixed directional
// drift per call (negative = downward trend), and min/max clamp the output when max > min.
// The timestamp argument is ignored — the value depends only on call history.
func GetRandomWalk(start, stepSize, bias, min, max float64) func(float64) float64 {
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
// WithNoise(fn, 0) returns fn unchanged.
func WithNoise(fn func(float64) float64, noise float64) func(float64) float64 {
	if noise <= 0 {
		return fn
	}
	return func(x float64) float64 {
		return fn(x) * (1 + noise*(2*rng.Float64()-1))
	}
}

func GetExp(start int64, durationSeconds int) func(x float64) float64 {
    // Build function : y = A.exp(B.(x-C)+D
    fn := func(x float64) float64 {
        A := 1.0
        B := 1 / float64(durationSeconds)
        C := float64(start)
        D := 0.0
        return A * math.Exp(B * (x - C)) + D
    }
    return fn
}
