package main

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

// pointMaker produces a DataPoint for a device at a given Unix timestamp.
// Each device must have its own maker so stateful or random operations
// (noise, anomaly, geo walk) are isolated and safe for concurrent use.
type pointMaker func(device string, ts int64) DataPoint

// rateTicker returns a shared channel that fires at the given rate (points/sec).
// Returns nil when rate <= 0 (unlimited).
func rateTicker(rate float64) (<-chan time.Time, func()) {
	if rate <= 0 {
		return nil, func() {}
	}
	t := time.NewTicker(time.Duration(float64(time.Second) / rate))
	return t.C, t.Stop
}

// waitRate blocks until the rate channel fires or ctx is cancelled.
// Returns false if the context was cancelled.
func waitRate(ctx context.Context, rateC <-chan time.Time) bool {
	if rateC == nil {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-rateC:
		return true
	}
}

func runBatch(rng *rand.Rand, makers []pointMaker, sink Sink, devices []string, start int64, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	ctx := context.Background()
	for d, device := range devices {
		for i := 0; i < count; i++ {
			if dropoutRate > 0 && rng.Float64() < dropoutRate {
				continue
			}
			ts := start + int64(i*stepSeconds)
			dp := makers[d](device, ts)
			if !waitRate(ctx, rateC) {
				return
			}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtime(ctx context.Context, rng *rand.Rand, makers []pointMaker, sink Sink, devices []string, start int64, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		// Each goroutine gets its own RNG so dropout decisions don't race across devices.
		dropRng := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		go func(device string, maker pointMaker, dropRng *rand.Rand) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for {
				// First point is emitted immediately; subsequent points wait for
				// the ticker. Always check ctx so a pre-cancelled context is respected.
				if sent > 0 {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}
				} else if ctx.Err() != nil {
					return
				}
				if dropoutRate > 0 && dropRng.Float64() < dropoutRate {
					sent++
					if sent >= count {
						return
					}
					continue
				}
				if !waitRate(ctx, rateC) {
					return
				}
				ts := start + int64(sent*stepSeconds)
				dp := maker(device, ts)
				if err := sink.Send(dp); err != nil {
					log.Printf("send error: %v", err)
				}
				sent++
				if sent >= count {
					return
				}
			}
		}(device, makers[d], dropRng)
	}
	wg.Wait()
}

// singleFieldMakers wraps per-device curve functions as pointMakers.
// Each fn must already capture its own RNG (not shared with other devices).
func singleFieldMakers(fns []func(float64) float64) []pointMaker {
	makers := make([]pointMaker, len(fns))
	for i, fn := range fns {
		fn := fn
		makers[i] = func(device string, ts int64) DataPoint {
			v := fn(float64(ts))
			return DataPoint{Device: device, Timestamp: ts, Value: &v}
		}
	}
	return makers
}

// multiFieldMakers builds one pointMaker per device for multi-field mode.
// Each maker gets its own RNG derived from rng so concurrent calls are safe.
func multiFieldMakers(rng *rand.Rand, fieldFns map[string]func(float64) float64, scales []float64, noise, anomalyRate, anomalyFactor float64) []pointMaker {
	makers := make([]pointMaker, len(scales))
	for i, scale := range scales {
		scale := scale
		devRng := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		makers[i] = func(device string, ts int64) DataPoint {
			return DataPoint{Device: device, Timestamp: ts, Fields: evalFields(devRng, fieldFns, scale, noise, anomalyRate, anomalyFactor, float64(ts))}
		}
	}
	return makers
}

// geoMakers builds one pointMaker per device for geo mode.
// Each maker owns its walker and a private RNG so concurrent calls are safe.
func geoMakers(rng *rand.Rand, walkers []*GeoWalker, stepSeconds int) []pointMaker {
	makers := make([]pointMaker, len(walkers))
	for i, walker := range walkers {
		walker := walker
		devRng := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		makers[i] = func(device string, ts int64) DataPoint {
			lat, lon := walker.Step(devRng, stepSeconds)
			return DataPoint{Device: device, Timestamp: ts, Fields: map[string]float64{"lat": lat, "lon": lon}}
		}
	}
	return makers
}

// dispatchRun calls runRealtime or runBatch depending on the realtime flag.
func dispatchRun(ctx context.Context, realtime bool, rng *rand.Rand, makers []pointMaker, sink Sink, devices []string, start int64, count, stepSeconds int, dropoutRate, rate float64) {
	if realtime {
		runRealtime(ctx, rng, makers, sink, devices, start, count, stepSeconds, dropoutRate, rate)
	} else {
		runBatch(rng, makers, sink, devices, start, count, stepSeconds, dropoutRate, rate)
	}
}

func evalFields(rng *rand.Rand, fieldFns map[string]func(float64) float64, scale, noise, anomalyRate, anomalyFactor, x float64) map[string]float64 {
	fields := make(map[string]float64, len(fieldFns))
	for name, fn := range fieldFns {
		v := fn(x) * scale
		if noise > 0 {
			v *= 1 + noise*(2*rng.Float64()-1)
		}
		if anomalyRate > 0 && anomalyFactor > 1 && rng.Float64() < anomalyRate {
			if rng.Float64() < 0.5 {
				v *= anomalyFactor
			} else {
				v /= anomalyFactor
			}
		}
		fields[name] = v
	}
	return fields
}
