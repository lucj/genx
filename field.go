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

// buildFieldFn constructs a curve function from a FieldConfig.
// rng is only consumed when fc.Type is "walk".
func buildFieldFn(rng *rand.Rand, fc FieldConfig, start int64, durationSeconds int) (func(float64) float64, error) {
	switch fc.Type {
	case "linear":
		first := 0.0
		if fc.First != nil {
			first = *fc.First
		}
		last := 1.0
		if fc.Last != nil {
			last = *fc.Last
		}
		return GetLinear(first, last, start, durationSeconds), nil
	case "cos":
		min := 0.0
		if fc.Min != nil {
			min = *fc.Min
		}
		max := 1.0
		if fc.Max != nil {
			max = *fc.Max
		}
		period := durationSeconds
		if fc.Period != "" {
			var err error
			period, err = GetSeconds(fc.Period)
			if err != nil {
				return nil, fmt.Errorf("invalid period: %w", err)
			}
		}
		return GetCosinus(min, max, period), nil
	case "log":
		return GetLog(start), nil
	case "exp":
		return GetExp(start, durationSeconds), nil
	case "sawtooth":
		min := 0.0
		if fc.Min != nil {
			min = *fc.Min
		}
		max := 1.0
		if fc.Max != nil {
			max = *fc.Max
		}
		period := durationSeconds
		if fc.Period != "" {
			var err error
			period, err = GetSeconds(fc.Period)
			if err != nil {
				return nil, fmt.Errorf("invalid period: %w", err)
			}
		}
		return GetSawtooth(min, max, start, period), nil
	case "square":
		min := 0.0
		if fc.Min != nil {
			min = *fc.Min
		}
		max := 1.0
		if fc.Max != nil {
			max = *fc.Max
		}
		period := durationSeconds
		if fc.Period != "" {
			var err error
			period, err = GetSeconds(fc.Period)
			if err != nil {
				return nil, fmt.Errorf("invalid period: %w", err)
			}
		}
		dutyCycle := 0.5
		if fc.DutyCycle != nil {
			dutyCycle = *fc.DutyCycle
		}
		return GetSquare(min, max, start, period, dutyCycle), nil
	case "walk":
		walkStart := 100.0
		if fc.WalkStart != nil {
			walkStart = *fc.WalkStart
		}
		step := 1.0
		if fc.WalkStep != nil {
			step = *fc.WalkStep
		}
		bias := 0.0
		if fc.WalkBias != nil {
			bias = *fc.WalkBias
		}
		wmin, wmax := 0.0, 0.0
		if fc.Min != nil {
			wmin = *fc.Min
		}
		if fc.Max != nil {
			wmax = *fc.Max
		}
		return GetRandomWalk(rng, walkStart, step, bias, wmin, wmax), nil
	default:
		return nil, fmt.Errorf("unknown type %q (use cos, linear, log, exp, walk, sawtooth, square)", fc.Type)
	}
}
