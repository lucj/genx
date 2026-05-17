package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

// sinkConfig carries the resolved parameters needed to construct a Sink.
type sinkConfig struct {
	output       string
	webhookURL   string
	webhookToken string
	natsURL      string
	natsSubject  string
	natsUser     string
	natsPassword string
	natsToken    string
	mqttBroker      string
	mqttTopic       string
	mqttClientID    string
	mqttQoS         int
	mqttUser        string
	mqttPassword    string
	mqttCACert      string
	mqttCert        string
	mqttKey         string
	mqttTLSInsecure bool
	mqttDeviceCerts map[string]MqttDeviceCert
	renderer        Renderer
}

func buildSink(cfg sinkConfig) (Sink, error) {
	switch cfg.output {
	case "stdout":
		return NewStdoutSink(cfg.renderer), nil
	case "webhook":
		if cfg.webhookURL == "" {
			return nil, fmt.Errorf("--webhook-url is required when --output is webhook")
		}
		return NewWebhookSink(cfg.webhookURL, cfg.webhookToken, cfg.renderer), nil
	case "nats":
		return NewNatsSink(cfg.natsURL, cfg.natsSubject, cfg.natsUser, cfg.natsPassword, cfg.natsToken, cfg.renderer)
	case "mqtt":
		return NewMqttSink(cfg.mqttBroker, cfg.mqttTopic, cfg.mqttClientID, cfg.mqttQoS, cfg.mqttUser, cfg.mqttPassword, cfg.mqttCACert, cfg.mqttCert, cfg.mqttKey, cfg.mqttTLSInsecure, cfg.mqttDeviceCerts, cfg.renderer)
	default:
		return nil, fmt.Errorf("unknown output %q (use stdout, webhook, nats, mqtt)", cfg.output)
	}
}

func main() {
	// Config file flag
	configPtr := flag.String("config", "", "path to YAML config file (CLI flags take precedence)")

	// Curve flags
	typePtr := flag.String("type", "cos", "type of curve: cos, linear, log, exp, walk")
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

	// Cosine curve flags
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
	mqttCACert := flag.String("mqtt-ca-cert", "", "path to CA certificate file for verifying the broker's TLS certificate")
	mqttCert := flag.String("mqtt-cert", "", "path to client certificate file for mTLS authentication")
	mqttKey := flag.String("mqtt-key", "", "path to client private key file for mTLS authentication")
	mqttTLSInsecure := flag.Bool("mqtt-tls-insecure", false, "skip broker TLS certificate verification (use only for testing)")
	var mqttDeviceCerts map[string]MqttDeviceCert

	// Payload template flags
	payloadTemplateStr := flag.String("payload-template", "", "Go text/template string for JSON payload")
	payloadTemplateFile := flag.String("payload-template-file", "", "path to a Go text/template file for JSON payload")

	// Utility flags
	generateConfig := flag.Bool("generate-config", false, "print a sample YAML config file and exit")

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	if *generateConfig {
		printSampleConfig()
		return
	}

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

		if cfg.Type != "" && !set["type"]                      { *typePtr = cfg.Type }
		if cfg.Duration != "" && !set["duration"]              { *durationPtr = cfg.Duration }
		if cfg.Step != "" && !set["step"]                      { *stepPtr = cfg.Step }
		if cfg.Device != "" && !set["device"]                  { *devicePtr = cfg.Device }
		if cfg.Devices != nil && !set["devices"]               { *devicesPtr = *cfg.Devices }
		if cfg.Spread != nil && !set["spread"]                 { *spreadPtr = *cfg.Spread }
		if cfg.Noise != nil && !set["noise"]                   { *noisePtr = *cfg.Noise }
		if cfg.Realtime != nil && !set["realtime"]             { *realtimePtr = *cfg.Realtime }
		if cfg.Seed != nil && !set["seed"]                     { *seedPtr = *cfg.Seed }
		if cfg.ReplayFile != "" && !set["replay-file"]         { *replayFilePtr = cfg.ReplayFile }

		if cfg.First != nil && !set["first"]                   { *linearFirst = *cfg.First }
		if cfg.Last != nil && !set["last"]                     { *linearLast = *cfg.Last }
		if cfg.Min != nil && !set["min"]                       { *cosMin = *cfg.Min }
		if cfg.Max != nil && !set["max"]                       { *cosMax = *cfg.Max }
		if cfg.Period != "" && !set["period"]                  { *cosPeriod = cfg.Period }
		if cfg.WalkStart != nil && !set["walk-start"]          { *walkStart = *cfg.WalkStart }
		if cfg.WalkStep != nil && !set["walk-step"]            { *walkStep = *cfg.WalkStep }
		if cfg.WalkBias != nil && !set["walk-bias"]            { *walkBias = *cfg.WalkBias }
		if cfg.WalkMin != nil && !set["walk-min"]              { *walkMin = *cfg.WalkMin }
		if cfg.WalkMax != nil && !set["walk-max"]              { *walkMax = *cfg.WalkMax }

		if cfg.Output != "" && !set["output"]                  { *outputPtr = cfg.Output }
		if cfg.WebhookURL != "" && !set["webhook-url"]         { *webhookURL = cfg.WebhookURL }
		if cfg.WebhookToken != "" && !set["webhook-token"]     { *webhookToken = cfg.WebhookToken }
		if cfg.NatsURL != "" && !set["nats-url"]               { *natsURL = cfg.NatsURL }
		if cfg.NatsSubject != "" && !set["nats-subject"]       { *natsSubject = cfg.NatsSubject }
		if cfg.NatsUser != "" && !set["nats-user"]             { *natsUser = cfg.NatsUser }
		if cfg.NatsPassword != "" && !set["nats-password"]     { *natsPassword = cfg.NatsPassword }
		if cfg.NatsToken != "" && !set["nats-token"]           { *natsToken = cfg.NatsToken }
		if cfg.MqttBroker != "" && !set["mqtt-broker"]         { *mqttBroker = cfg.MqttBroker }
		if cfg.MqttTopic != "" && !set["mqtt-topic"]           { *mqttTopic = cfg.MqttTopic }
		if cfg.MqttQoS != nil && !set["mqtt-qos"]              { *mqttQoS = *cfg.MqttQoS }
		if cfg.MqttClientID != "" && !set["mqtt-client-id"]    { *mqttClientID = cfg.MqttClientID }
		if cfg.MqttUser != "" && !set["mqtt-user"]                   { *mqttUser = cfg.MqttUser }
		if cfg.MqttPassword != "" && !set["mqtt-password"]           { *mqttPassword = cfg.MqttPassword }
		if cfg.MqttCACert != "" && !set["mqtt-ca-cert"]              { *mqttCACert = cfg.MqttCACert }
		if cfg.MqttCert != "" && !set["mqtt-cert"]                   { *mqttCert = cfg.MqttCert }
		if cfg.MqttKey != "" && !set["mqtt-key"]                     { *mqttKey = cfg.MqttKey }
		if cfg.MqttTLSInsecure != nil && !set["mqtt-tls-insecure"]   { *mqttTLSInsecure = *cfg.MqttTLSInsecure }
		mqttDeviceCerts = cfg.MqttDeviceCerts
		if cfg.PayloadTemplate != "" && !set["payload-template"]          { *payloadTemplateStr = cfg.PayloadTemplate }
		if cfg.PayloadTemplateFile != "" && !set["payload-template-file"] { *payloadTemplateFile = cfg.PayloadTemplateFile }
	}

	// Build the renderer (file takes precedence over inline string).
	renderer := Renderer(JSONRenderer)
	if *payloadTemplateFile != "" {
		raw, err := os.ReadFile(*payloadTemplateFile)
		if err != nil {
			log.Fatalf("cannot read payload-template-file: %v", err)
		}
		tmpl, err := template.New("payload").Parse(string(raw))
		if err != nil {
			log.Fatalf("invalid payload template: %v", err)
		}
		renderer = NewTemplateRenderer(tmpl)
	} else if *payloadTemplateStr != "" {
		tmpl, err := template.New("payload").Parse(*payloadTemplateStr)
		if err != nil {
			log.Fatalf("invalid payload template: %v", err)
		}
		renderer = NewTemplateRenderer(tmpl)
	}

	// Initialise RNG (seeded before any random values are consumed).
	rng := newRand()
	if *seedPtr != 0 {
		rng = seededRand(uint64(*seedPtr))
	}

	// Build the output sink.
	sink, err := buildSink(sinkConfig{
		output:       *outputPtr,
		webhookURL:   *webhookURL,
		webhookToken: *webhookToken,
		natsURL:      *natsURL,
		natsSubject:  *natsSubject,
		natsUser:     *natsUser,
		natsPassword: *natsPassword,
		natsToken:    *natsToken,
		mqttBroker:   *mqttBroker,
		mqttTopic:    *mqttTopic,
		mqttClientID: *mqttClientID,
		mqttQoS:      *mqttQoS,
		mqttUser:        *mqttUser,
		mqttPassword:    *mqttPassword,
		mqttCACert:      *mqttCACert,
		mqttCert:        *mqttCert,
		mqttKey:         *mqttKey,
		mqttTLSInsecure: *mqttTLSInsecure,
		mqttDeviceCerts: mqttDeviceCerts,
		renderer:        renderer,
	})
	if err != nil {
		log.Fatalf("sink: %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			log.Printf("sink close error: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Replay mode: send a recorded JSON-lines file through the sink.
	if *replayFilePtr != "" {
		stepSeconds, err := GetSeconds(*stepPtr)
		if err != nil {
			log.Fatalf("invalid --step: %v", err)
		}
		if err := runReplay(ctx, *replayFilePtr, sink, *realtimePtr, stepSeconds); err != nil {
			log.Fatalf("%v", err)
		}
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
			fn, err := buildFieldFn(rng, fc, start, durationSeconds)
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
			runRealtimeMulti(ctx, rng, fieldFns, scales, *noisePtr, sink, devices, itemCount, stepSeconds)
		} else {
			runBatchMulti(rng, fieldFns, scales, *noisePtr, sink, devices, start, itemCount, stepSeconds)
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
		periodSeconds, err := GetSeconds(*cosPeriod)
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
			fns[i] = WithNoise(rng, GetRandomWalk(rng, *walkStart*scale, *walkStep, *walkBias, *walkMin, *walkMax), *noisePtr)
		} else {
			fn := baseFn
			s := scale
			fns[i] = WithNoise(rng, func(x float64) float64 { return fn(x) * s }, *noisePtr)
		}
	}
	if *realtimePtr {
		runRealtime(ctx, fns, sink, devices, itemCount, stepSeconds)
	} else {
		runBatch(fns, sink, devices, start, itemCount, stepSeconds)
	}
}
