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
	// Config file flag
	configPtr := flag.String("config", "", "path to YAML config file (CLI flags take precedence)")

	// Curve flags
	typePtr := flag.String("type", "cos", "type of curve: cos, linear, log, exp")
	durationPtr := flag.String("duration", "1d", "total duration (e.g. 2d, 6h, 30m)")
	stepPtr := flag.String("step", "1h", "sampling interval (e.g. 1h, 5m, 10s)")
	devicePtr := flag.String("device", "device", "device/sensor name (or prefix when --devices > 1)")
	devicesPtr := flag.Int("devices", 1, "number of devices to simulate simultaneously")
	spreadPtr := flag.Float64("spread", 0.0, "per-device value spread as a ratio, e.g. 0.1 = ±10%")
	noisePtr := flag.Float64("noise", 0.0, "random noise added to every sample as a ratio, e.g. 0.05 = ±5%")
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
	webhookToken := flag.String("webhook-token", "", "bearer token for webhook Authorization header")

	// NATS flags
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	natsSubject := flag.String("nats-subject", "genx", "NATS subject to publish to")
	natsUser := flag.String("nats-user", "", "NATS username")
	natsPassword := flag.String("nats-password", "", "NATS password")

	// MQTT flags
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	mqttTopic := flag.String("mqtt-topic", "genx", "MQTT topic to publish to")
	mqttQoS := flag.Int("mqtt-qos", 0, "MQTT QoS level (0, 1, or 2)")
	mqttClientID := flag.String("mqtt-client-id", fmt.Sprintf("genx-%d", os.Getpid()), "MQTT client ID")
	mqttUser := flag.String("mqtt-user", "", "MQTT username")
	mqttPassword := flag.String("mqtt-password", "", "MQTT password")

	flag.Parse()

	// Apply config file values for any flag not explicitly set on the CLI.
	if *configPtr != "" {
		cfg, err := LoadConfig(*configPtr)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		set := map[string]bool{}
		flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

		if cfg.Type != "" && !set["type"]               { *typePtr = cfg.Type }
		if cfg.Duration != "" && !set["duration"]        { *durationPtr = cfg.Duration }
		if cfg.Step != "" && !set["step"]                { *stepPtr = cfg.Step }
		if cfg.Device != "" && !set["device"]            { *devicePtr = cfg.Device }
		if cfg.Devices != nil && !set["devices"]         { *devicesPtr = *cfg.Devices }
		if cfg.Spread != nil && !set["spread"]           { *spreadPtr = *cfg.Spread }
		if cfg.Realtime != nil && !set["realtime"]       { *realtimePtr = *cfg.Realtime }

		if cfg.First != nil && !set["first"]             { *linearFirst = *cfg.First }
		if cfg.Last != nil && !set["last"]               { *linearLast = *cfg.Last }
		if cfg.Min != nil && !set["min"]                 { *cosMin = *cfg.Min }
		if cfg.Max != nil && !set["max"]                 { *cosMax = *cfg.Max }
		if cfg.Period != "" && !set["period"]            { *cosPeriod = cfg.Period }

		if cfg.Output != "" && !set["output"]                   { *outputPtr = cfg.Output }
		if cfg.WebhookURL != "" && !set["webhook-url"]          { *webhookURL = cfg.WebhookURL }
		if cfg.WebhookToken != "" && !set["webhook-token"]      { *webhookToken = cfg.WebhookToken }
		if cfg.NatsURL != "" && !set["nats-url"]                { *natsURL = cfg.NatsURL }
		if cfg.NatsSubject != "" && !set["nats-subject"]        { *natsSubject = cfg.NatsSubject }
		if cfg.NatsUser != "" && !set["nats-user"]              { *natsUser = cfg.NatsUser }
		if cfg.NatsPassword != "" && !set["nats-password"]      { *natsPassword = cfg.NatsPassword }
		if cfg.Noise != nil && !set["noise"]                     { *noisePtr = *cfg.Noise }
		if cfg.MqttBroker != "" && !set["mqtt-broker"]          { *mqttBroker = cfg.MqttBroker }
		if cfg.MqttTopic != "" && !set["mqtt-topic"]            { *mqttTopic = cfg.MqttTopic }
		if cfg.MqttQoS != nil && !set["mqtt-qos"]               { *mqttQoS = *cfg.MqttQoS }
		if cfg.MqttClientID != "" && !set["mqtt-client-id"]     { *mqttClientID = cfg.MqttClientID }
		if cfg.MqttUser != "" && !set["mqtt-user"]              { *mqttUser = cfg.MqttUser }
		if cfg.MqttPassword != "" && !set["mqtt-password"]      { *mqttPassword = cfg.MqttPassword }
	}

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
		fns[i] = WithNoise(func(x float64) float64 { return fn(x) * s }, *noisePtr)
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
		sink = NewWebhookSink(*webhookURL, *webhookToken)
	case "nats":
		sink, err = NewNatsSink(*natsURL, *natsSubject, *natsUser, *natsPassword)
		if err != nil {
			log.Fatalf("NATS connection failed: %v", err)
		}
	case "mqtt":
		sink, err = NewMqttSink(*mqttBroker, *mqttTopic, *mqttClientID, *mqttQoS, *mqttUser, *mqttPassword)
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
