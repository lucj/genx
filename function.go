package main

import (
	"math"
)

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
