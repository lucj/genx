package main

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

func runBatch(fns []func(float64) float64, sink Sink, devices []string, start int64, count, stepSeconds int) {
	for d, device := range devices {
		for i := 0; i < count; i++ {
			ts := start + int64(i*stepSeconds)
			v := fns[d](float64(ts))
			dp := DataPoint{Device: device, Timestamp: ts, Value: &v}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtime(ctx context.Context, fns []func(float64) float64, sink Sink, devices []string, count, stepSeconds int) {
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

func runBatchMulti(rng *rand.Rand, fieldFns map[string]func(float64) float64, scales []float64, noise float64, sink Sink, devices []string, start int64, count, stepSeconds int) {
	for d, device := range devices {
		scale := scales[d]
		for i := 0; i < count; i++ {
			ts := start + int64(i*stepSeconds)
			dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(rng, fieldFns, scale, noise, float64(ts))}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func runRealtimeMulti(ctx context.Context, rng *rand.Rand, fieldFns map[string]func(float64) float64, scales []float64, noise float64, sink Sink, devices []string, count, stepSeconds int) {
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
				ts := time.Now().Unix()
				dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(rng, fieldFns, scale, noise, float64(ts))}
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

func evalFields(rng *rand.Rand, fieldFns map[string]func(float64) float64, scale, noise, x float64) map[string]float64 {
	fields := make(map[string]float64, len(fieldFns))
	for name, fn := range fieldFns {
		v := fn(x) * scale
		if noise > 0 {
			v *= 1 + noise*(2*rng.Float64()-1)
		}
		fields[name] = v
	}
	return fields
}
