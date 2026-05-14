package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"sync"
	"time"
)

func main() {
	// Curve flags
	typePtr := flag.String("type", "cos", "type of curve: cos, linear, log, exp")
	durationPtr := flag.String("duration", "1d", "total duration (e.g. 2d, 6h, 30m)")
	stepPtr := flag.String("step", "1h", "sampling interval (e.g. 1h, 5m, 10s)")
	devicePtr := flag.String("device", "device", "device/sensor name (or prefix when --devices > 1)")
	devicesPtr := flag.Int("devices", 1, "number of devices to simulate simultaneously")
	spreadPtr := flag.Float64("spread", 0.0, "per-device value spread as a ratio, e.g. 0.1 = ±10%")
	realtimePtr := flag.Bool("realtime", false, "real-time mode: emit one point per step interval")

	// Linear curve flags
	linearFirst := flag.Float64("first", 0, "first value (linear)")
	linearLast := flag.Float64("last", 1, "last value (linear)")

	// Cosinus curve flags
	cosMin := flag.Float64("min", 10, "minimum value (cos)")
	cosMax := flag.Float64("max", 25, "maximum value (cos)")
	cosPeriod := flag.String("period", "1d", "period (cos), e.g. 1d, 12h")

	// Output sink flags
	outputPtr := flag.String("output", "stdout", "output backend: stdout, webhook, nats, mqtt")

	// Webhook flags
	webhookURL := flag.String("webhook-url", "", "webhook URL (required for --output webhook)")

	// NATS flags
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	natsSubject := flag.String("nats-subject", "genx", "NATS subject to publish to")

	// MQTT flags
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	mqttTopic := flag.String("mqtt-topic", "genx", "MQTT topic to publish to")
	mqttQoS := flag.Int("mqtt-qos", 0, "MQTT QoS level (0, 1, or 2)")
	mqttClientID := flag.String("mqtt-client-id", fmt.Sprintf("genx-%d", os.Getpid()), "MQTT client ID")

	flag.Parse()

	if *devicesPtr < 1 {
		log.Fatal("--devices must be at least 1")
	}

	// Parse durations
	durationSeconds, err := GetSeconds(*durationPtr)
	if err != nil {
		log.Fatalf("invalid --duration: %v", err)
	}
	stepSeconds, err := GetSeconds(*stepPtr)
	if err != nil {
		log.Fatalf("invalid --step: %v", err)
	}

	// Build the base mathematical function
	start := time.Now().Unix()

	var baseFn func(float64) float64
	switch *typePtr {
	case "linear":
		baseFn = GetLinear(*linearFirst, *linearLast, start, durationSeconds)
	case "cos":
		var periodSeconds int
		periodSeconds, err = GetSeconds(*cosPeriod)
		if err != nil {
			log.Fatalf("invalid --period: %v", err)
		}
		baseFn = GetCosinus(*cosMin, *cosMax, periodSeconds)
	case "log":
		baseFn = GetLog(start)
	case "exp":
		baseFn = GetExp(start, durationSeconds)
	default:
		log.Fatalf("unknown curve type %q (use cos, linear, log, exp)", *typePtr)
	}

	// Build device names and per-device functions
	devices := make([]string, *devicesPtr)
	fns := make([]func(float64) float64, *devicesPtr)
	for i := range devices {
		if *devicesPtr == 1 {
			devices[i] = *devicePtr
		} else {
			devices[i] = fmt.Sprintf("%s-%d", *devicePtr, i)
		}
		scale := 1.0
		if *spreadPtr > 0 {
			scale = 1.0 + *spreadPtr*(2*rand.Float64()-1)
		}
		fn := baseFn
		s := scale
		fns[i] = func(x float64) float64 { return fn(x) * s }
	}

	// Build the output sink
	var sink Sink
	switch *outputPtr {
	case "stdout":
		sink = NewStdoutSink()
	case "webhook":
		if *webhookURL == "" {
			log.Fatal("--webhook-url is required when --output is webhook")
		}
		sink = NewWebhookSink(*webhookURL)
	case "nats":
		sink, err = NewNatsSink(*natsURL, *natsSubject)
		if err != nil {
			log.Fatalf("NATS connection failed: %v", err)
		}
	case "mqtt":
		sink, err = NewMqttSink(*mqttBroker, *mqttTopic, *mqttClientID, *mqttQoS)
		if err != nil {
			log.Fatalf("MQTT connection failed: %v", err)
		}
	default:
		log.Fatalf("unknown output %q (use stdout, webhook, nats, mqtt)", *outputPtr)
	}
	defer sink.Close()

	itemCount := durationSeconds / stepSeconds

	if *realtimePtr {
		runRealtime(fns, sink, devices, itemCount, stepSeconds)
	} else {
		runBatch(fns, sink, devices, start, itemCount, stepSeconds)
	}
}

// runBatch generates all data points immediately for each device, spacing timestamps by stepSeconds.
func runBatch(fns []func(float64) float64, sink Sink, devices []string, start int64, count, stepSeconds int) {
	for d, device := range devices {
		for i := 0; i < count; i++ {
			ts := start + int64(i*stepSeconds)
			dp := DataPoint{Device: device, Timestamp: ts, Value: fns[d](float64(ts))}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

// runRealtime emits one data point per step interval for each device concurrently.
func runRealtime(fns []func(float64) float64, sink Sink, devices []string, count, stepSeconds int) {
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		go func(device string, fn func(float64) float64) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for range ticker.C {
				ts := time.Now().Unix()
				dp := DataPoint{Device: device, Timestamp: ts, Value: fn(float64(ts))}
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
