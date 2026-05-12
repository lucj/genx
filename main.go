package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Curve flags
	typePtr := flag.String("type", "cos", "type of curve: cos, linear, log, exp")
	durationPtr := flag.String("duration", "1d", "total duration (e.g. 2d, 6h, 30m)")
	stepPtr := flag.String("step", "1h", "sampling interval (e.g. 1h, 5m, 10s)")
	devicePtr := flag.String("device", "device", "device/sensor name included in each data point")
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

	// Parse durations
	durationSeconds, err := GetSeconds(*durationPtr)
	if err != nil {
		log.Fatalf("invalid --duration: %v", err)
	}
	stepSeconds, err := GetSeconds(*stepPtr)
	if err != nil {
		log.Fatalf("invalid --step: %v", err)
	}

	// Build the mathematical function
	start := time.Now().Unix()

	var fn func(float64) float64
	switch *typePtr {
	case "linear":
		fn = GetLinear(*linearFirst, *linearLast, start, durationSeconds)
	case "cos":
		var periodSeconds int
		periodSeconds, err = GetSeconds(*cosPeriod)
		if err != nil {
			log.Fatalf("invalid --period: %v", err)
		}
		fn = GetCosinus(*cosMin, *cosMax, periodSeconds)
	case "log":
		fn = GetLog(start)
	case "exp":
		fn = GetExp(start, durationSeconds)
	default:
		log.Fatalf("unknown curve type %q (use cos, linear, log, exp)", *typePtr)
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
	defer func() {
		if err := sink.Close(); err != nil {
			log.Printf("sink close error: %v", err)
		}
	}()

	itemCount := durationSeconds / stepSeconds

	if *realtimePtr {
		runRealtime(fn, sink, *devicePtr, itemCount, stepSeconds)
	} else {
		runBatch(fn, sink, *devicePtr, start, itemCount, stepSeconds)
	}
}

// runBatch generates all data points immediately, spacing timestamps by stepSeconds.
func runBatch(fn func(float64) float64, sink Sink, device string, start int64, count, stepSeconds int) {
	for i := 0; i < count; i++ {
		ts := start + int64(i*stepSeconds)
		dp := DataPoint{Device: device, Timestamp: ts, Value: fn(float64(ts))}
		if err := sink.Send(dp); err != nil {
			log.Printf("send error: %v", err)
		}
	}
}

// runRealtime emits one data point per step interval using the actual current time.
func runRealtime(fn func(float64) float64, sink Sink, device string, count, stepSeconds int) {
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
}
