package main

import (
	"context"
	"fmt"
	"math/rand/v2"
)

// phaseParams holds the fully resolved (global + phase override) parameters
// for a single scenario phase.
type phaseParams struct {
	curveType       string
	stepSeconds     int
	durationSeconds int
	cosMin          float64
	cosMax          float64
	cosPeriod       string
	dutyCycle       float64
	linearFirst     float64
	linearLast      float64
	walkStart       float64
	walkStep        float64
	walkBias        float64
	walkMin         float64
	walkMax         float64
	noise           float64
	anomalyRate     float64
	anomalyFactor   float64
	dropoutRate     float64
}

// resolvePhase merges global CLI flags with per-phase overrides.
func resolvePhase(v *cliFlags, p PhaseConfig) (phaseParams, error) {
	pp := phaseParams{
		curveType:     v.curveType,
		cosPeriod:     v.cosPeriod,
		dutyCycle:     v.dutyCycle,
		cosMin:        v.cosMin,
		cosMax:        v.cosMax,
		linearFirst:   v.linearFirst,
		linearLast:    v.linearLast,
		walkStart:     v.walkStart,
		walkStep:      v.walkStep,
		walkBias:      v.walkBias,
		walkMin:       v.walkMin,
		walkMax:       v.walkMax,
		noise:         v.noise,
		anomalyRate:   v.anomalyRate,
		anomalyFactor: v.anomalyFactor,
		dropoutRate:   v.dropoutRate,
	}

	if p.Type != ""          { pp.curveType = p.Type }
	if p.Min != nil          { pp.cosMin = *p.Min }
	if p.Max != nil          { pp.cosMax = *p.Max }
	if p.Period != ""        { pp.cosPeriod = p.Period }
	if p.DutyCycle != nil    { pp.dutyCycle = *p.DutyCycle }
	if p.First != nil        { pp.linearFirst = *p.First }
	if p.Last != nil         { pp.linearLast = *p.Last }
	if p.WalkStart != nil    { pp.walkStart = *p.WalkStart }
	if p.WalkStep != nil     { pp.walkStep = *p.WalkStep }
	if p.WalkBias != nil     { pp.walkBias = *p.WalkBias }
	if p.WalkMin != nil      { pp.walkMin = *p.WalkMin }
	if p.WalkMax != nil      { pp.walkMax = *p.WalkMax }
	if p.Noise != nil        { pp.noise = *p.Noise }
	if p.AnomalyRate != nil  { pp.anomalyRate = *p.AnomalyRate }
	if p.AnomalyFactor != nil { pp.anomalyFactor = *p.AnomalyFactor }
	if p.DropoutRate != nil  { pp.dropoutRate = *p.DropoutRate }

	if err := validatePhaseParams(pp); err != nil {
		return pp, err
	}

	if p.Duration == "" {
		return pp, fmt.Errorf("duration is required")
	}
	dur, err := GetSeconds(p.Duration)
	if err != nil {
		return pp, fmt.Errorf("invalid duration %q: %w", p.Duration, err)
	}
	pp.durationSeconds = dur

	stepStr := v.step
	if p.Step != "" {
		stepStr = p.Step
	}
	step, err := GetSeconds(stepStr)
	if err != nil {
		return pp, fmt.Errorf("invalid step %q: %w", stepStr, err)
	}
	pp.stepSeconds = step

	return pp, nil
}

// buildPhaseFns constructs per-device curve functions for a single-field phase.
// phaseStart is the Unix timestamp used as the reference point for relative curves.
func buildPhaseFns(rng *rand.Rand, pp phaseParams, devices int, spread float64, phaseStart int64) ([]func(float64) float64, error) {
	var baseFn func(float64) float64
	switch pp.curveType {
	case "linear":
		baseFn = GetLinear(pp.linearFirst, pp.linearLast, phaseStart, pp.durationSeconds)
	case "cos":
		periodSeconds, err := GetSeconds(pp.cosPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid period: %w", err)
		}
		baseFn = GetCosinus(pp.cosMin, pp.cosMax, periodSeconds)
	case "sawtooth":
		periodSeconds, err := GetSeconds(pp.cosPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid period: %w", err)
		}
		baseFn = GetSawtooth(pp.cosMin, pp.cosMax, phaseStart, periodSeconds)
	case "square":
		periodSeconds, err := GetSeconds(pp.cosPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid period: %w", err)
		}
		if pp.dutyCycle <= 0 || pp.dutyCycle >= 1 {
			return nil, fmt.Errorf("duty-cycle must be between 0 and 1 (exclusive), got %g", pp.dutyCycle)
		}
		baseFn = GetSquare(pp.cosMin, pp.cosMax, phaseStart, periodSeconds, pp.dutyCycle)
	case "log":
		baseFn = GetLog(phaseStart)
	case "exp":
		baseFn = GetExp(phaseStart, pp.durationSeconds)
	case "walk":
		// baseFn intentionally nil; each device gets its own closure below.
	default:
		return nil, fmt.Errorf("unknown curve type %q", pp.curveType)
	}

	fns := make([]func(float64) float64, devices)
	for i := range fns {
		devRng := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		scale := 1.0
		if spread > 0 {
			scale = 1.0 + spread*(2*rng.Float64()-1)
		}
		if pp.curveType == "walk" {
			fns[i] = WithAnomaly(devRng, WithNoise(devRng, GetRandomWalk(devRng, pp.walkStart*scale, pp.walkStep, pp.walkBias, pp.walkMin, pp.walkMax), pp.noise), pp.anomalyRate, pp.anomalyFactor)
		} else {
			fn := baseFn
			s := scale
			fns[i] = WithAnomaly(devRng, WithNoise(devRng, func(x float64) float64 { return fn(x) * s }, pp.noise), pp.anomalyRate, pp.anomalyFactor)
		}
	}
	return fns, nil
}

// runScenario executes scenario phases in sequence, sharing sink and RNG state.
// Geo walkers are created once so position is continuous across geo phases.
// In batch mode, timestamps are continuous across phases.
func runScenario(ctx context.Context, rng *rand.Rand, v *cliFlags, phases []PhaseConfig, sink Sink, deviceNames []string, batchStart int64) error {
	walkers := make([]*GeoWalker, len(deviceNames))
	for i := range walkers {
		walkers[i] = NewGeoWalker(v.geoLat, v.geoLon, v.geoBearing, v.geoSpeed, v.geoDrift)
	}

	batchTs := batchStart
	for i, phase := range phases {
		if ctx.Err() != nil {
			return nil
		}
		pp, err := resolvePhase(v, phase)
		if err != nil {
			return fmt.Errorf("scenario phase %d: %w", i+1, err)
		}
		itemCount := pp.durationSeconds / pp.stepSeconds

		// phaseStart anchors curve functions and timestamps for this phase.
		// Both batch and realtime modes use the same reference so --from is
		// respected regardless of pacing.
		phaseStart := batchTs

		switch {
		case pp.curveType == "geo":
			makers := geoMakers(rng, walkers, pp.stepSeconds)
			dispatchRun(ctx, v.realtime, rng, makers, sink, deviceNames, phaseStart, itemCount, pp.stepSeconds, pp.dropoutRate, v.rate)
		case len(phase.Fields) > 0:
			fieldFns := make(map[string]func(float64) float64, len(phase.Fields))
			for name, fc := range phase.Fields {
				fn, err := buildFieldFn(rng, fc, phaseStart, pp.durationSeconds)
				if err != nil {
					return fmt.Errorf("scenario phase %d field %q: %w", i+1, name, err)
				}
				fieldFns[name] = fn
			}
			scales := make([]float64, len(deviceNames))
			for j := range scales {
				scales[j] = 1.0
				if v.spread > 0 {
					scales[j] = 1.0 + v.spread*(2*rng.Float64()-1)
				}
			}
			makers := multiFieldMakers(rng, fieldFns, scales, pp.noise, pp.anomalyRate, pp.anomalyFactor)
			dispatchRun(ctx, v.realtime, rng, makers, sink, deviceNames, phaseStart, itemCount, pp.stepSeconds, pp.dropoutRate, v.rate)
		default:
			fns, err := buildPhaseFns(rng, pp, len(deviceNames), v.spread, phaseStart)
			if err != nil {
				return fmt.Errorf("scenario phase %d: %w", i+1, err)
			}
			dispatchRun(ctx, v.realtime, rng, singleFieldMakers(fns), sink, deviceNames, phaseStart, itemCount, pp.stepSeconds, pp.dropoutRate, v.rate)
		}

		batchTs += int64(pp.durationSeconds)
	}
	return nil
}
