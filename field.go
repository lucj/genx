package main

import (
	"fmt"
	"math/rand/v2"
)

// FieldConfig defines the curve for one field in a multi-field payload.
type FieldConfig struct {
	Type   string   `yaml:"type"`
	First  *float64 `yaml:"first"`
	Last   *float64 `yaml:"last"`
	Min    *float64 `yaml:"min"`
	Max    *float64 `yaml:"max"`
	Period string   `yaml:"period"`
	// Walk-specific parameters
	WalkStart *float64 `yaml:"walk-start"`
	WalkStep  *float64 `yaml:"walk-step"`
	WalkBias  *float64 `yaml:"walk-bias"`
	// Square-wave parameter
	DutyCycle *float64 `yaml:"duty-cycle"`
}

// resolvePeriod returns the parsed duration of periodStr, or fallback if empty.
func resolvePeriod(periodStr string, fallback int) (int, error) {
	if periodStr == "" {
		return fallback, nil
	}
	p, err := GetSeconds(periodStr)
	if err != nil {
		return 0, fmt.Errorf("invalid period: %w", err)
	}
	return p, nil
}

// ptrOr dereferences p if non-nil, otherwise returns def.
func ptrOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

// buildFieldFn constructs a curve function from a FieldConfig.
// rng is only consumed when fc.Type is "walk".
func buildFieldFn(rng *rand.Rand, fc FieldConfig, start int64, durationSeconds int) (func(float64) float64, error) {
	switch fc.Type {
	case "linear":
		return GetLinear(ptrOr(fc.First, 0.0), ptrOr(fc.Last, 1.0), start, durationSeconds), nil
	case "cos":
		period, err := resolvePeriod(fc.Period, durationSeconds)
		if err != nil {
			return nil, err
		}
		return GetCosinus(ptrOr(fc.Min, 0.0), ptrOr(fc.Max, 1.0), period), nil
	case "log":
		return GetLog(start), nil
	case "exp":
		return GetExp(start, durationSeconds), nil
	case "sawtooth":
		period, err := resolvePeriod(fc.Period, durationSeconds)
		if err != nil {
			return nil, err
		}
		return GetSawtooth(ptrOr(fc.Min, 0.0), ptrOr(fc.Max, 1.0), start, period), nil
	case "square":
		period, err := resolvePeriod(fc.Period, durationSeconds)
		if err != nil {
			return nil, err
		}
		return GetSquare(ptrOr(fc.Min, 0.0), ptrOr(fc.Max, 1.0), start, period, ptrOr(fc.DutyCycle, 0.5)), nil
	case "walk":
		return GetRandomWalk(rng,
			ptrOr(fc.WalkStart, 100.0),
			ptrOr(fc.WalkStep, 1.0),
			ptrOr(fc.WalkBias, 0.0),
			ptrOr(fc.Min, 0.0),
			ptrOr(fc.Max, 0.0),
		), nil
	default:
		return nil, fmt.Errorf("unknown type %q (use cos, linear, log, exp, walk, sawtooth, square)", fc.Type)
	}
}
