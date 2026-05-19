package main

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

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

func runBatch(rng *rand.Rand, fns []func(float64) float64, sink Sink, devices []string, start int64, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	ctx := context.Background()
	for d, device := range devices {
		for i := 0; i < count; i++ {
			if dropoutRate > 0 && rng.Float64() < dropoutRate {
				continue
			}
			ts := start + int64(i*stepSeconds)
			v := fns[d](float64(ts))
			dp := DataPoint{Device: device, Timestamp: ts, Value: &v}
			if !waitRate(ctx, rateC) {
				return
			}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtime(ctx context.Context, rng *rand.Rand, fns []func(float64) float64, sink Sink, devices []string, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		go func(device string, fn func(float64) float64) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				if dropoutRate > 0 && rng.Float64() < dropoutRate {
					sent++
					if sent >= count {
						return
					}
					continue
				}
				if !waitRate(ctx, rateC) {
					return
				}
				ts := time.Now().Unix()
				v := fn(float64(ts))
				dp := DataPoint{Device: device, Timestamp: ts, Value: &v}
				if err := sink.Send(dp); err != nil {
					log.Printf("send error: %v", err)
				}
				sent++
				if sent >= count {
					return
				}
			}
		}(device, fns[d])
	}
	wg.Wait()
}

func runBatchMulti(rng *rand.Rand, fieldFns map[string]func(float64) float64, scales []float64, noise, anomalyRate, anomalyFactor, dropoutRate float64, sink Sink, devices []string, start int64, count, stepSeconds int, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	ctx := context.Background()
	for d, device := range devices {
		scale := scales[d]
		for i := 0; i < count; i++ {
			if dropoutRate > 0 && rng.Float64() < dropoutRate {
				continue
			}
			ts := start + int64(i*stepSeconds)
			dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(rng, fieldFns, scale, noise, anomalyRate, anomalyFactor, float64(ts))}
			if !waitRate(ctx, rateC) {
				return
			}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtimeMulti(ctx context.Context, rng *rand.Rand, fieldFns map[string]func(float64) float64, scales []float64, noise, anomalyRate, anomalyFactor, dropoutRate float64, sink Sink, devices []string, count, stepSeconds int, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		go func(device string, scale float64) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				if dropoutRate > 0 && rng.Float64() < dropoutRate {
					sent++
					if sent >= count {
						return
					}
					continue
				}
				if !waitRate(ctx, rateC) {
					return
				}
				ts := time.Now().Unix()
				dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(rng, fieldFns, scale, noise, anomalyRate, anomalyFactor, float64(ts))}
				if err := sink.Send(dp); err != nil {
					log.Printf("send error: %v", err)
				}
				sent++
				if sent >= count {
					return
				}
			}
		}(device, scales[d])
	}
	wg.Wait()
}

func runBatchGeo(rng *rand.Rand, walkers []*GeoWalker, sink Sink, devices []string, start int64, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	ctx := context.Background()
	for d, device := range devices {
		for i := 0; i < count; i++ {
			if dropoutRate > 0 && rng.Float64() < dropoutRate {
				continue
			}
			ts := start + int64(i*stepSeconds)
			lat, lon := walkers[d].Step(rng, stepSeconds)
			dp := DataPoint{
				Device:    device,
				Timestamp: ts,
				Fields:    map[string]float64{"lat": lat, "lon": lon},
			}
			if !waitRate(ctx, rateC) {
				return
			}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtimeGeo(ctx context.Context, rng *rand.Rand, walkers []*GeoWalker, sink Sink, devices []string, count, stepSeconds int, dropoutRate, rate float64) {
	rateC, stop := rateTicker(rate)
	defer stop()
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		go func(device string, walker *GeoWalker) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				if dropoutRate > 0 && rng.Float64() < dropoutRate {
					sent++
					if sent >= count {
						return
					}
					continue
				}
				if !waitRate(ctx, rateC) {
					return
				}
				ts := time.Now().Unix()
				lat, lon := walker.Step(rng, stepSeconds)
				dp := DataPoint{
					Device:    device,
					Timestamp: ts,
					Fields:    map[string]float64{"lat": lat, "lon": lon},
				}
				if err := sink.Send(dp); err != nil {
					log.Printf("send error: %v", err)
				}
				sent++
				if sent >= count {
					return
				}
			}
		}(device, walkers[d])
	}
	wg.Wait()
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
