package main

import "fmt"

// FieldConfig defines the curve for one field in a multi-field payload.
type FieldConfig struct {
	Type   string   `yaml:"type"`
	First  *float64 `yaml:"first"`
	Last   *float64 `yaml:"last"`
	Min    *float64 `yaml:"min"`
	Max    *float64 `yaml:"max"`
	Period string   `yaml:"period"`
}

// buildFieldFn constructs a curve function from a FieldConfig.
// durationSeconds is used as the default period for cos and as the time range for linear/exp.
func buildFieldFn(fc FieldConfig, start int64, durationSeconds int) (func(float64) float64, error) {
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
	default:
		return nil, fmt.Errorf("unknown type %q (use cos, linear, log, exp)", fc.Type)
	}
}
