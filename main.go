package main

import (
	"flag"
	"fmt"
	"log"
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
	seedPtr := flag.Int64("seed", 0, "random seed for reproducible output (0 = random); batch mode only")

	// Replay flag
	replayFilePtr := flag.String("replay-file", "", "path to a JSON-lines file to replay through the configured sink")

	// Linear curve flags
	linearFirst := flag.Float64("first", 0, "first value (linear)")
	linearLast := flag.Float64("last", 1, "last value (linear)")

	// Cosinus curve flags
	cosMin := flag.Float64("min", 10, "minimum value (cos)")
	cosMax := flag.Float64("max", 25, "maximum value (cos)")
	cosPeriod := flag.String("period", "1d", "period (cos), e.g. 1d, 12h")

	// Random walk flags
	walkStart := flag.Float64("walk-start", 100.0, "starting value (walk)")
	walkStep := flag.Float64("walk-step", 1.0, "max delta magnitude per sample (walk)")
	walkBias := flag.Float64("walk-bias", 0.0, "per-step directional drift (walk); negative = downward")
	walkMin := flag.Float64("walk-min", 0.0, "lower clamp bound (walk); clamping disabled when walk-min == walk-max")
	walkMax := flag.Float64("walk-max", 0.0, "upper clamp bound (walk); clamping disabled when walk-min == walk-max")

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
	natsToken := flag.String("nats-token", "", "NATS token (used when --nats-user is not set)")

	// MQTT flags
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	mqttTopic := flag.String("mqtt-topic", "genx", "MQTT topic to publish to")
	mqttQoS := flag.Int("mqtt-qos", 0, "MQTT QoS level (0, 1, or 2)")
	mqttClientID := flag.String("mqtt-client-id", fmt.Sprintf("genx-%d", os.Getpid()), "MQTT client ID")
	mqttUser := flag.String("mqtt-user", "", "MQTT username")
	mqttPassword := flag.String("mqtt-password", "", "MQTT password")

	// Payload template flags
	payloadTemplate := flag.String("payload-template", "", "Go text/template string for JSON payload")
	payloadTemplateFile := flag.String("payload-template-file", "", "path to a Go text/template file for JSON payload")

	flag.Parse()

	// Load config and apply values for flags not explicitly set on the CLI.
	var cfg *Config
	if *configPtr != "" {
		c, err := LoadConfig(*configPtr)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		cfg = c
		set := map[string]bool{}
		flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

		if cfg.Type != "" && !set["type"]                       { *typePtr = cfg.Type }
		if cfg.Duration != "" && !set["duration"]               { *durationPtr = cfg.Duration }
		if cfg.Step != "" && !set["step"]                       { *stepPtr = cfg.Step }
		if cfg.Device != "" && !set["device"]                   { *devicePtr = cfg.Device }
		if cfg.Devices != nil && !set["devices"]                { *devicesPtr = *cfg.Devices }
		if cfg.Spread != nil && !set["spread"]                  { *spreadPtr = *cfg.Spread }
		if cfg.Noise != nil && !set["noise"]                    { *noisePtr = *cfg.Noise }
		if cfg.Realtime != nil && !set["realtime"]              { *realtimePtr = *cfg.Realtime }
		if cfg.Seed != nil && !set["seed"]                      { *seedPtr = *cfg.Seed }
		if cfg.ReplayFile != "" && !set["replay-file"]          { *replayFilePtr = cfg.ReplayFile }

		if cfg.First != nil && !set["first"]                    { *linearFirst = *cfg.First }
		if cfg.Last != nil && !set["last"]                      { *linearLast = *cfg.Last }
		if cfg.Min != nil && !set["min"]                        { *cosMin = *cfg.Min }
		if cfg.Max != nil && !set["max"]                        { *cosMax = *cfg.Max }
		if cfg.Period != "" && !set["period"]                   { *cosPeriod = cfg.Period }
		if cfg.WalkStart != nil && !set["walk-start"]           { *walkStart = *cfg.WalkStart }
		if cfg.WalkStep != nil && !set["walk-step"]             { *walkStep = *cfg.WalkStep }
		if cfg.WalkBias != nil && !set["walk-bias"]             { *walkBias = *cfg.WalkBias }
		if cfg.WalkMin != nil && !set["walk-min"]               { *walkMin = *cfg.WalkMin }
		if cfg.WalkMax != nil && !set["walk-max"]               { *walkMax = *cfg.WalkMax }

		if cfg.Output != "" && !set["output"]                   { *outputPtr = cfg.Output }
		if cfg.WebhookURL != "" && !set["webhook-url"]          { *webhookURL = cfg.WebhookURL }
		if cfg.WebhookToken != "" && !set["webhook-token"]      { *webhookToken = cfg.WebhookToken }
		if cfg.NatsURL != "" && !set["nats-url"]                { *natsURL = cfg.NatsURL }
		if cfg.NatsSubject != "" && !set["nats-subject"]        { *natsSubject = cfg.NatsSubject }
		if cfg.NatsUser != "" && !set["nats-user"]              { *natsUser = cfg.NatsUser }
		if cfg.NatsPassword != "" && !set["nats-password"]      { *natsPassword = cfg.NatsPassword }
		if cfg.NatsToken != "" && !set["nats-token"]            { *natsToken = cfg.NatsToken }
		if cfg.MqttBroker != "" && !set["mqtt-broker"]          { *mqttBroker = cfg.MqttBroker }
		if cfg.MqttTopic != "" && !set["mqtt-topic"]            { *mqttTopic = cfg.MqttTopic }
		if cfg.MqttQoS != nil && !set["mqtt-qos"]               { *mqttQoS = *cfg.MqttQoS }
		if cfg.MqttClientID != "" && !set["mqtt-client-id"]     { *mqttClientID = cfg.MqttClientID }
		if cfg.MqttUser != "" && !set["mqtt-user"]              { *mqttUser = cfg.MqttUser }
		if cfg.MqttPassword != "" && !set["mqtt-password"]      { *mqttPassword = cfg.MqttPassword }
		if cfg.PayloadTemplate != "" && !set["payload-template"]           { *payloadTemplate = cfg.PayloadTemplate }
		if cfg.PayloadTemplateFile != "" && !set["payload-template-file"]  { *payloadTemplateFile = cfg.PayloadTemplateFile }
	}

	// Initialize payload template (file takes precedence over inline string).
	if *payloadTemplateFile != "" {
		raw, err := os.ReadFile(*payloadTemplateFile)
		if err != nil {
			log.Fatalf("cannot read payload-template-file: %v", err)
		}
		if err := initTemplate(string(raw)); err != nil {
			log.Fatalf("invalid payload template: %v", err)
		}
	} else if *payloadTemplate != "" {
		if err := initTemplate(*payloadTemplate); err != nil {
			log.Fatalf("invalid payload template: %v", err)
		}
	}

	// Reseed before any random values are consumed.
	if *seedPtr != 0 {
		initRand(uint64(*seedPtr))
	}

	// Build the output sink.
	var err error
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
		sink, err = NewNatsSink(*natsURL, *natsSubject, *natsUser, *natsPassword, *natsToken)
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

	// Replay mode: send a recorded JSON-lines file through the sink.
	if *replayFilePtr != "" {
		stepSeconds, err := GetSeconds(*stepPtr)
		if err != nil {
			log.Fatalf("invalid --step: %v", err)
		}
		runReplay(*replayFilePtr, sink, *realtimePtr, stepSeconds)
		return
	}

	if *devicesPtr < 1 {
		log.Fatal("--devices must be at least 1")
	}

	// Parse durations.
	durationSeconds, err := GetSeconds(*durationPtr)
	if err != nil {
		log.Fatalf("invalid --duration: %v", err)
	}
	stepSeconds, err := GetSeconds(*stepPtr)
	if err != nil {
		log.Fatalf("invalid --step: %v", err)
	}

	start := time.Now().Unix()

	// Build device names.
	devices := make([]string, *devicesPtr)
	for i := range devices {
		if *devicesPtr == 1 {
			devices[i] = *devicePtr
		} else {
			devices[i] = fmt.Sprintf("%s-%d", *devicePtr, i)
		}
	}

	itemCount := durationSeconds / stepSeconds

	// Multi-field mode: only available via config file.
	if cfg != nil && len(cfg.Fields) > 0 {
		fieldFns := make(map[string]func(float64) float64, len(cfg.Fields))
		for name, fc := range cfg.Fields {
			fn, err := buildFieldFn(fc, start, durationSeconds)
			if err != nil {
				log.Fatalf("field %q: %v", name, err)
			}
			fieldFns[name] = fn
		}
		scales := make([]float64, *devicesPtr)
		for i := range scales {
			scales[i] = 1.0
			if *spreadPtr > 0 {
				scales[i] = 1.0 + *spreadPtr*(2*rng.Float64()-1)
			}
		}
		if *realtimePtr {
			runRealtimeMulti(fieldFns, scales, *noisePtr, sink, devices, itemCount, stepSeconds)
		} else {
			runBatchMulti(fieldFns, scales, *noisePtr, sink, devices, start, itemCount, stepSeconds)
		}
		return
	}

	// Single-field mode.
	// Walk is handled per-device (stateful closure); all other types share a pure baseFn.
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
	case "walk":
		// baseFn intentionally left nil; each device gets its own closure below.
	default:
		log.Fatalf("unknown curve type %q (use cos, linear, log, exp, walk)", *typePtr)
	}

	fns := make([]func(float64) float64, *devicesPtr)
	for i := range fns {
		scale := 1.0
		if *spreadPtr > 0 {
			scale = 1.0 + *spreadPtr*(2*rng.Float64()-1)
		}
		if *typePtr == "walk" {
			// Spread varies the starting value so devices begin at different levels.
			fns[i] = WithNoise(GetRandomWalk(*walkStart*scale, *walkStep, *walkBias, *walkMin, *walkMax), *noisePtr)
		} else {
			fn := baseFn
			s := scale
			fns[i] = WithNoise(func(x float64) float64 { return fn(x) * s }, *noisePtr)
		}
	}

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
			v := fns[d](float64(ts))
			dp := DataPoint{Device: device, Timestamp: ts, Value: &v}
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

// runBatchMulti generates multi-field data points for each device sequentially.
func runBatchMulti(fieldFns map[string]func(float64) float64, scales []float64, noise float64, sink Sink, devices []string, start int64, count, stepSeconds int) {
	for d, device := range devices {
		scale := scales[d]
		for i := 0; i < count; i++ {
			ts := start + int64(i*stepSeconds)
			dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(fieldFns, scale, noise, float64(ts))}
			if err := sink.Send(dp); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

// runRealtimeMulti emits multi-field data points per step interval for each device concurrently.
func runRealtimeMulti(fieldFns map[string]func(float64) float64, scales []float64, noise float64, sink Sink, devices []string, count, stepSeconds int) {
	var wg sync.WaitGroup
	for d, device := range devices {
		wg.Add(1)
		go func(device string, scale float64) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(stepSeconds) * time.Second)
			defer ticker.Stop()
			sent := 0
			for range ticker.C {
				ts := time.Now().Unix()
				dp := DataPoint{Device: device, Timestamp: ts, Fields: evalFields(fieldFns, scale, noise, float64(ts))}
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

// evalFields evaluates all field functions at timestamp x, applying scale and noise.
func evalFields(fieldFns map[string]func(float64) float64, scale, noise, x float64) map[string]float64 {
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
