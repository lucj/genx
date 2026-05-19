package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

// sinkConfig carries the resolved parameters needed to construct a Sink.
type sinkConfig struct {
	output       string
	webhookURL         string
	webhookToken       string
	webhookContentType string
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
	filePath        string
	fileMaxBytes    int64
	fileMaxAge      time.Duration
	kafkaBrokers    string
	kafkaTopic      string
	kafkaUsername   string
	kafkaPassword   string
	kafkaTLS        bool
	kafkaTLSInsecure bool
	otlpEndpoint   string
	otlpHTTP       bool
	otlpHeaders    map[string]string
	otlpInsecure   bool
	otlpMetricName string
	prometheusPort   int
	prometheusMetric string
	influxdbURL      string
	influxdbToken    string
	influxdbOrg      string
	influxdbBucket   string
	influxMeasurement string
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
		return NewWebhookSink(cfg.webhookURL, cfg.webhookToken, cfg.webhookContentType, cfg.renderer), nil
	case "nats":
		return NewNatsSink(cfg.natsURL, cfg.natsSubject, cfg.natsUser, cfg.natsPassword, cfg.natsToken, cfg.renderer)
	case "mqtt":
		return NewMqttSink(cfg.mqttBroker, cfg.mqttTopic, cfg.mqttClientID, cfg.mqttQoS, cfg.mqttUser, cfg.mqttPassword, cfg.mqttCACert, cfg.mqttCert, cfg.mqttKey, cfg.mqttTLSInsecure, cfg.mqttDeviceCerts, cfg.renderer)
	case "file":
		if cfg.filePath == "" {
			return nil, fmt.Errorf("--file-path is required when --output is file")
		}
		return NewFileSink(cfg.filePath, cfg.fileMaxBytes, cfg.fileMaxAge, cfg.renderer)
	case "kafka":
		return NewKafkaSink(cfg.kafkaBrokers, cfg.kafkaTopic, cfg.kafkaUsername, cfg.kafkaPassword, cfg.kafkaTLS, cfg.kafkaTLSInsecure, cfg.renderer)
	case "otlp":
		return NewOTLPSink(cfg.otlpEndpoint, cfg.otlpHTTP, cfg.otlpHeaders, cfg.otlpInsecure, cfg.otlpMetricName)
	case "prometheus":
		return NewPrometheusSink(cfg.prometheusPort, cfg.prometheusMetric)
	case "influxdb":
		return NewInfluxDBSink(cfg.influxdbURL, cfg.influxdbToken, cfg.influxdbOrg, cfg.influxdbBucket, cfg.influxMeasurement)
	default:
		return nil, fmt.Errorf("unknown output %q (use stdout, webhook, nats, mqtt, file, kafka, otlp, prometheus, influxdb)", cfg.output)
	}
}

func main() {
	var (
		// General
		configFile  string
		curveType   string
		duration    string
		step        string
		device      string
		devices     int
		spread      float64
		noise       float64
		anomalyRate   float64
		anomalyFactor float64
		dropoutRate   float64
		realtime    bool
		seed        int64
		replayFile  string
		rate        float64
		generateConfig bool

		// Periodic curves (cos, sawtooth, square)
		cosMin    float64
		cosMax    float64
		cosPeriod string
		dutyCycle float64

		// Linear
		linearFirst float64
		linearLast  float64

		// Walk
		walkStart float64
		walkStep  float64
		walkBias  float64
		walkMin   float64
		walkMax   float64

		// Geo
		geoLat     float64
		geoLon     float64
		geoSpeed   float64
		geoBearing float64
		geoDrift   float64

		// Output
		output             string
		format             string
		isoTime            bool
		influxMeasurement  string
		cloudEventSource   string
		cloudEventType     string

		// Payload template
		payloadTemplate     string
		payloadTemplateFile string

		// Webhook
		webhookURL   string
		webhookToken string

		// NATS
		natsURL      string
		natsSubject  string
		natsUser     string
		natsPassword string
		natsToken    string

		// MQTT
		mqttBroker      string
		mqttTopic       string
		mqttQoS         int
		mqttClientID    string
		mqttUser        string
		mqttPassword    string
		mqttCACert      string
		mqttCert        string
		mqttKey         string
		mqttTLSInsecure bool

		// File
		filePath    string
		fileMaxSize string
		fileMaxAge  string

		// Kafka
		kafkaBrokers     string
		kafkaTopic       string
		kafkaUsername    string
		kafkaPassword    string
		kafkaTLS         bool
		kafkaTLSInsecure bool

		// OTLP
		otlpEndpoint   string
		otlpHTTP       bool
		otlpHeaders    []string
		otlpInsecure   bool
		otlpMetricName string

		// Prometheus pull
		prometheusPort   int
		prometheusMetric string

		// InfluxDB
		influxdbURL    string
		influxdbToken  string
		influxdbOrg    string
		influxdbBucket string

		// Named devices
		deviceNameList []string
	)

	var mqttDeviceCerts map[string]MqttDeviceCert

	rootCmd := &cobra.Command{
		Use:   "genx",
		Short: "IoT device simulator — generates synthetic time-series data",
		Long: `genx emits synthetic measurements as JSON to test IoT pipelines, dashboards, and alerting systems.

Use --config to load a YAML config file; CLI flags override any config value.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}

			if generateConfig {
				printSampleConfig()
				return nil
			}

			// Load config and apply values for flags not explicitly set on the CLI.
			var cfg *Config
			if configFile != "" {
				c, err := LoadConfig(configFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cfg = c

				changed := cmd.Flags().Changed

				if cfg.Type != "" && !changed("type")                      { curveType = cfg.Type }
				if cfg.Duration != "" && !changed("duration")              { duration = cfg.Duration }
				if cfg.Step != "" && !changed("step")                      { step = cfg.Step }
				if cfg.Device != "" && !changed("device")                  { device = cfg.Device }
				if cfg.Devices != nil && !changed("devices")               { devices = *cfg.Devices }
				if cfg.Spread != nil && !changed("spread")                 { spread = *cfg.Spread }
				if cfg.Noise != nil && !changed("noise")                   { noise = *cfg.Noise }
				if cfg.AnomalyRate != nil && !changed("anomaly-rate")      { anomalyRate = *cfg.AnomalyRate }
				if cfg.AnomalyFactor != nil && !changed("anomaly-factor")  { anomalyFactor = *cfg.AnomalyFactor }
				if cfg.DropoutRate != nil && !changed("dropout-rate")      { dropoutRate = *cfg.DropoutRate }
				if cfg.Realtime != nil && !changed("realtime")             { realtime = *cfg.Realtime }
				if cfg.Seed != nil && !changed("seed")                     { seed = *cfg.Seed }
				if cfg.ReplayFile != "" && !changed("replay-file")         { replayFile = cfg.ReplayFile }
				if cfg.Rate != nil && !changed("rate")                     { rate = *cfg.Rate }

				if cfg.First != nil && !changed("first")                   { linearFirst = *cfg.First }
				if cfg.Last != nil && !changed("last")                     { linearLast = *cfg.Last }
				if cfg.Min != nil && !changed("min")                       { cosMin = *cfg.Min }
				if cfg.Max != nil && !changed("max")                       { cosMax = *cfg.Max }
				if cfg.Period != "" && !changed("period")                  { cosPeriod = cfg.Period }
				if cfg.DutyCycle != nil && !changed("duty-cycle")          { dutyCycle = *cfg.DutyCycle }
				if cfg.WalkStart != nil && !changed("walk-start")          { walkStart = *cfg.WalkStart }
				if cfg.WalkStep != nil && !changed("walk-step")            { walkStep = *cfg.WalkStep }
				if cfg.WalkBias != nil && !changed("walk-bias")            { walkBias = *cfg.WalkBias }
				if cfg.WalkMin != nil && !changed("walk-min")              { walkMin = *cfg.WalkMin }
				if cfg.WalkMax != nil && !changed("walk-max")              { walkMax = *cfg.WalkMax }
				if cfg.GeoLat != nil && !changed("geo-lat")                { geoLat = *cfg.GeoLat }
				if cfg.GeoLon != nil && !changed("geo-lon")                { geoLon = *cfg.GeoLon }
				if cfg.GeoSpeed != nil && !changed("geo-speed")            { geoSpeed = *cfg.GeoSpeed }
				if cfg.GeoBearing != nil && !changed("geo-bearing")        { geoBearing = *cfg.GeoBearing }
				if cfg.GeoDrift != nil && !changed("geo-drift")            { geoDrift = *cfg.GeoDrift }

				if cfg.Output != "" && !changed("output")                  { output = cfg.Output }
				if cfg.WebhookURL != "" && !changed("webhook-url")         { webhookURL = cfg.WebhookURL }
				if cfg.WebhookToken != "" && !changed("webhook-token")     { webhookToken = cfg.WebhookToken }
				if cfg.NatsURL != "" && !changed("nats-url")               { natsURL = cfg.NatsURL }
				if cfg.NatsSubject != "" && !changed("nats-subject")       { natsSubject = cfg.NatsSubject }
				if cfg.NatsUser != "" && !changed("nats-user")             { natsUser = cfg.NatsUser }
				if cfg.NatsPassword != "" && !changed("nats-password")     { natsPassword = cfg.NatsPassword }
				if cfg.NatsToken != "" && !changed("nats-token")           { natsToken = cfg.NatsToken }
				if cfg.MqttBroker != "" && !changed("mqtt-broker")         { mqttBroker = cfg.MqttBroker }
				if cfg.MqttTopic != "" && !changed("mqtt-topic")           { mqttTopic = cfg.MqttTopic }
				if cfg.MqttQoS != nil && !changed("mqtt-qos")              { mqttQoS = *cfg.MqttQoS }
				if cfg.MqttClientID != "" && !changed("mqtt-client-id")    { mqttClientID = cfg.MqttClientID }
				if cfg.MqttUser != "" && !changed("mqtt-user")             { mqttUser = cfg.MqttUser }
				if cfg.MqttPassword != "" && !changed("mqtt-password")     { mqttPassword = cfg.MqttPassword }
				if cfg.MqttCACert != "" && !changed("mqtt-ca-cert")        { mqttCACert = cfg.MqttCACert }
				if cfg.MqttCert != "" && !changed("mqtt-cert")             { mqttCert = cfg.MqttCert }
				if cfg.MqttKey != "" && !changed("mqtt-key")               { mqttKey = cfg.MqttKey }
				if cfg.MqttTLSInsecure != nil && !changed("mqtt-tls-insecure") { mqttTLSInsecure = *cfg.MqttTLSInsecure }
				mqttDeviceCerts = cfg.MqttDeviceCerts
				if cfg.FilePath != "" && !changed("file-path")           { filePath = cfg.FilePath }
				if cfg.FileMaxSize != "" && !changed("file-max-size")    { fileMaxSize = cfg.FileMaxSize }
				if cfg.FileMaxAge != "" && !changed("file-max-age")      { fileMaxAge = cfg.FileMaxAge }
				if cfg.KafkaBrokers != "" && !changed("kafka-brokers")         { kafkaBrokers = cfg.KafkaBrokers }
				if cfg.KafkaTopic != "" && !changed("kafka-topic")             { kafkaTopic = cfg.KafkaTopic }
				if cfg.KafkaUsername != "" && !changed("kafka-username")       { kafkaUsername = cfg.KafkaUsername }
				if cfg.KafkaPassword != "" && !changed("kafka-password")       { kafkaPassword = cfg.KafkaPassword }
				if cfg.KafkaTLS != nil && !changed("kafka-tls")                { kafkaTLS = *cfg.KafkaTLS }
				if cfg.KafkaTLSInsecure != nil && !changed("kafka-tls-insecure") { kafkaTLSInsecure = *cfg.KafkaTLSInsecure }
				if cfg.PrometheusPort != nil && !changed("prometheus-port")         { prometheusPort = *cfg.PrometheusPort }
				if cfg.PrometheusMetric != "" && !changed("prometheus-metric")     { prometheusMetric = cfg.PrometheusMetric }
				if cfg.OTLPEndpoint != "" && !changed("otlp-endpoint")           { otlpEndpoint = cfg.OTLPEndpoint }
				if cfg.OTLPInsecure != nil && !changed("otlp-insecure")          { otlpInsecure = *cfg.OTLPInsecure }
				if cfg.OTLPHTTP != nil && !changed("otlp-http")                  { otlpHTTP = *cfg.OTLPHTTP }
				if cfg.OTLPMetricName != "" && !changed("otlp-metric")           { otlpMetricName = cfg.OTLPMetricName }
				if len(cfg.OTLPHeaders) > 0 && !changed("otlp-header")           { otlpHeaders = cfg.OTLPHeaders }
				if cfg.Format != "" && !changed("format")                            { format = cfg.Format }
				if cfg.InfluxMeasurement != "" && !changed("influx-measurement")     { influxMeasurement = cfg.InfluxMeasurement }
				if cfg.CloudEventSource != "" && !changed("cloudevent-source")       { cloudEventSource = cfg.CloudEventSource }
				if cfg.CloudEventType != "" && !changed("cloudevent-type")           { cloudEventType = cfg.CloudEventType }
				if cfg.InfluxDBURL != "" && !changed("influxdb-url")                { influxdbURL = cfg.InfluxDBURL }
				if cfg.InfluxDBToken != "" && !changed("influxdb-token")            { influxdbToken = cfg.InfluxDBToken }
				if cfg.InfluxDBOrg != "" && !changed("influxdb-org")                { influxdbOrg = cfg.InfluxDBOrg }
				if cfg.InfluxDBBucket != "" && !changed("influxdb-bucket")          { influxdbBucket = cfg.InfluxDBBucket }
				if len(cfg.DeviceNames) > 0 && !changed("device-names")             { deviceNameList = cfg.DeviceNames }
				if cfg.PayloadTemplate != "" && !changed("payload-template")          { payloadTemplate = cfg.PayloadTemplate }
				if cfg.PayloadTemplateFile != "" && !changed("payload-template-file") { payloadTemplateFile = cfg.PayloadTemplateFile }
			}

			// Infer output sink from sink-specific flags when --output was not set.
			if !cmd.Flags().Changed("output") {
				switch {
				case cmd.Flags().Changed("webhook-url"):
					output = "webhook"
				case cmd.Flags().Changed("nats-url"):
					output = "nats"
				case cmd.Flags().Changed("mqtt-broker"):
					output = "mqtt"
				case cmd.Flags().Changed("file-path"):
					output = "file"
				case cmd.Flags().Changed("kafka-brokers"):
					output = "kafka"
				case cmd.Flags().Changed("otlp-endpoint"):
					output = "otlp"
				case cmd.Flags().Changed("prometheus-port"):
					output = "prometheus"
				case cmd.Flags().Changed("influxdb-url"):
					output = "influxdb"
				}
			}

			// Build the renderer (template takes precedence over format).
			renderer := Renderer(JSONRenderer)
			if isoTime {
				renderer = ISOJSONRenderer
			}
			webhookCT := "application/json"
			switch format {
			case "csv":
				renderer = NewCSVRenderer(isoTime)
			case "influx":
				renderer = NewInfluxRenderer(influxMeasurement)
			case "cloudevent":
				renderer = NewCloudEventRenderer(cloudEventSource, cloudEventType, isoTime)
				webhookCT = "application/cloudevents+json"
			case "json", "":
				// already set above
			default:
				return fmt.Errorf("unknown --format %q (use json, csv, influx, or cloudevent)", format)
			}
			if payloadTemplateFile != "" {
				raw, err := os.ReadFile(payloadTemplateFile)
				if err != nil {
					return fmt.Errorf("cannot read payload-template-file: %w", err)
				}
				tmpl, err := template.New("payload").Parse(string(raw))
				if err != nil {
					return fmt.Errorf("invalid payload template: %w", err)
				}
				renderer = NewTemplateRenderer(tmpl, isoTime)
			} else if payloadTemplate != "" {
				tmpl, err := template.New("payload").Parse(payloadTemplate)
				if err != nil {
					return fmt.Errorf("invalid payload template: %w", err)
				}
				renderer = NewTemplateRenderer(tmpl, isoTime)
			}

			// Initialise RNG.
			rng := newRand()
			if seed != 0 {
				rng = seededRand(uint64(seed))
			}

			// Parse file sink rotation parameters.
			var fileMaxBytes int64
			if fileMaxSize != "" {
				var parseErr error
				fileMaxBytes, parseErr = ParseSize(fileMaxSize)
				if parseErr != nil {
					return fmt.Errorf("invalid --file-max-size: %w", parseErr)
				}
			}
			var fileMaxAgeDur time.Duration
			if fileMaxAge != "" {
				ageSecs, err := GetSeconds(fileMaxAge)
				if err != nil {
					return fmt.Errorf("invalid --file-max-age: %w", err)
				}
				fileMaxAgeDur = time.Duration(ageSecs) * time.Second
			}

			// Parse --otlp-header key=value pairs into a map.
			otlpHeaderMap := make(map[string]string, len(otlpHeaders))
			for _, h := range otlpHeaders {
				k, v, ok := strings.Cut(h, "=")
				if !ok {
					return fmt.Errorf("invalid --otlp-header %q: expected key=value", h)
				}
				otlpHeaderMap[k] = v
			}

			// Build the output sink.
			sink, err := buildSink(sinkConfig{
				output:             output,
				webhookURL:         webhookURL,
				webhookToken:       webhookToken,
				webhookContentType: webhookCT,
				influxdbURL:        influxdbURL,
				influxdbToken:      influxdbToken,
				influxdbOrg:        influxdbOrg,
				influxdbBucket:     influxdbBucket,
				influxMeasurement:  influxMeasurement,
				natsURL:         natsURL,
				natsSubject:     natsSubject,
				natsUser:        natsUser,
				natsPassword:    natsPassword,
				natsToken:       natsToken,
				mqttBroker:      mqttBroker,
				mqttTopic:       mqttTopic,
				mqttClientID:    mqttClientID,
				mqttQoS:         mqttQoS,
				mqttUser:        mqttUser,
				mqttPassword:    mqttPassword,
				mqttCACert:      mqttCACert,
				mqttCert:        mqttCert,
				mqttKey:         mqttKey,
				mqttTLSInsecure: mqttTLSInsecure,
				mqttDeviceCerts: mqttDeviceCerts,
				filePath:        filePath,
				fileMaxBytes:    fileMaxBytes,
				fileMaxAge:      fileMaxAgeDur,
				kafkaBrokers:    kafkaBrokers,
				kafkaTopic:      kafkaTopic,
				kafkaUsername:   kafkaUsername,
				kafkaPassword:   kafkaPassword,
				kafkaTLS:        kafkaTLS,
				kafkaTLSInsecure: kafkaTLSInsecure,
				otlpEndpoint:   otlpEndpoint,
				otlpHTTP:       otlpHTTP,
				otlpHeaders:    otlpHeaderMap,
				otlpInsecure:   otlpInsecure,
				otlpMetricName: otlpMetricName,
				prometheusPort:   prometheusPort,
				prometheusMetric: prometheusMetric,
				renderer:        renderer,
			})
			if err != nil {
				return fmt.Errorf("sink: %w", err)
			}
			defer func() {
				if err := sink.Close(); err != nil {
					log.Printf("sink close error: %v", err)
				}
			}()

			// Wrap sink to count sends and print a summary when the run ends.
			stats := &statsSink{inner: sink}
			sink = stats
			runStart := time.Now()
			defer func() {
				elapsed := time.Since(runStart)
				pts := stats.sent.Load()
				errs := stats.errors.Load()
				rate := 0.0
				if elapsed.Seconds() > 0 {
					rate = float64(pts) / elapsed.Seconds()
				}
				fmt.Fprintf(os.Stderr, "sent %d points in %.1fs (%.1f pts/s, %d errors)\n",
					pts, elapsed.Seconds(), rate, errs)
			}()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			// Replay mode.
			if replayFile != "" {
				stepSeconds, err := GetSeconds(step)
				if err != nil {
					return fmt.Errorf("invalid --step: %w", err)
				}
				return runReplay(ctx, replayFile, sink, realtime, stepSeconds)
			}

			durationSeconds, err := GetSeconds(duration)
			if err != nil {
				return fmt.Errorf("invalid --duration: %w", err)
			}
			stepSeconds, err := GetSeconds(step)
			if err != nil {
				return fmt.Errorf("invalid --step: %w", err)
			}

			start := time.Now().Unix()

			// Resolve device names: explicit list takes precedence over --device / --devices.
			var deviceNames []string
			if len(deviceNameList) > 0 {
				if cmd.Flags().Changed("devices") && devices != len(deviceNameList) {
					return fmt.Errorf("--devices=%d conflicts with --device-names (%d names provided)", devices, len(deviceNameList))
				}
				deviceNames = deviceNameList
				devices = len(deviceNames)
			} else {
				if devices < 1 {
					return fmt.Errorf("--devices must be at least 1")
				}
				deviceNames = make([]string, devices)
				for i := range deviceNames {
					if devices == 1 {
						deviceNames[i] = device
					} else {
						deviceNames[i] = fmt.Sprintf("%s-%d", device, i)
					}
				}
			}

			itemCount := durationSeconds / stepSeconds

			// Multi-field mode: only available via config file.
			if cfg != nil && len(cfg.Fields) > 0 {
				fieldFns := make(map[string]func(float64) float64, len(cfg.Fields))
				for name, fc := range cfg.Fields {
					fn, err := buildFieldFn(rng, fc, start, durationSeconds)
					if err != nil {
						return fmt.Errorf("field %q: %w", name, err)
					}
					fieldFns[name] = fn
				}
				scales := make([]float64, devices)
				for i := range scales {
					scales[i] = 1.0
					if spread > 0 {
						scales[i] = 1.0 + spread*(2*rng.Float64()-1)
					}
				}
				if realtime {
					runRealtimeMulti(ctx, rng, fieldFns, scales, noise, anomalyRate, anomalyFactor, dropoutRate, sink, deviceNames, itemCount, stepSeconds, rate)
				} else {
					runBatchMulti(rng, fieldFns, scales, noise, anomalyRate, anomalyFactor, dropoutRate, sink, deviceNames, start, itemCount, stepSeconds, rate)
				}
				return nil
			}

			// Geo mode.
			if curveType == "geo" {
				walkers := make([]*GeoWalker, devices)
				for i := range walkers {
					walkers[i] = NewGeoWalker(geoLat, geoLon, geoBearing, geoSpeed, geoDrift)
				}
				if realtime {
					runRealtimeGeo(ctx, rng, walkers, sink, deviceNames, itemCount, stepSeconds, dropoutRate, rate)
				} else {
					runBatchGeo(rng, walkers, sink, deviceNames, start, itemCount, stepSeconds, dropoutRate, rate)
				}
				return nil
			}

			// Single-field mode.
			var baseFn func(float64) float64
			switch curveType {
			case "linear":
				baseFn = GetLinear(linearFirst, linearLast, start, durationSeconds)
			case "cos":
				periodSeconds, err := GetSeconds(cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				baseFn = GetCosinus(cosMin, cosMax, periodSeconds)
			case "sawtooth":
				periodSeconds, err := GetSeconds(cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				baseFn = GetSawtooth(cosMin, cosMax, start, periodSeconds)
			case "square":
				periodSeconds, err := GetSeconds(cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				if dutyCycle <= 0 || dutyCycle >= 1 {
					return fmt.Errorf("--duty-cycle must be between 0 and 1 (exclusive), got %g", dutyCycle)
				}
				baseFn = GetSquare(cosMin, cosMax, start, periodSeconds, dutyCycle)
			case "log":
				baseFn = GetLog(start)
			case "exp":
				baseFn = GetExp(start, durationSeconds)
			case "walk":
				// baseFn intentionally left nil; each device gets its own closure below.
			default:
				return fmt.Errorf("unknown curve type %q (use cos, linear, log, exp, walk, sawtooth, square, geo)", curveType)
			}

			fns := make([]func(float64) float64, devices)
			for i := range fns {
				scale := 1.0
				if spread > 0 {
					scale = 1.0 + spread*(2*rng.Float64()-1)
				}
				if curveType == "walk" {
					fns[i] = WithAnomaly(rng, WithNoise(rng, GetRandomWalk(rng, walkStart*scale, walkStep, walkBias, walkMin, walkMax), noise), anomalyRate, anomalyFactor)
				} else {
					fn := baseFn
					s := scale
					fns[i] = WithAnomaly(rng, WithNoise(rng, func(x float64) float64 { return fn(x) * s }, noise), anomalyRate, anomalyFactor)
				}
			}
			if realtime {
				runRealtime(ctx, rng, fns, sink, deviceNames, itemCount, stepSeconds, dropoutRate, rate)
			} else {
				runBatch(rng, fns, sink, deviceNames, start, itemCount, stepSeconds, dropoutRate, rate)
			}
			return nil
		},
	}

	f := rootCmd.Flags()

	// General
	f.StringVar(&configFile, "config", "", "path to YAML config file (CLI flags take precedence)")
	f.BoolVar(&generateConfig, "generate-config", false, "print a sample YAML config file and exit")
	f.StringVar(&curveType, "type", "walk", "curve type: cos, linear, log, exp, walk, sawtooth, square, geo")
	f.StringVar(&duration, "duration", "1d", "total duration (e.g. 2d, 6h, 30m)")
	f.StringVar(&step, "step", "1h", "sampling interval (e.g. 1h, 5m, 10s)")
	f.StringVar(&device, "device", "device", "device/sensor name (or prefix when --devices > 1)")
	f.IntVar(&devices, "devices", 1, "number of devices to simulate simultaneously")
	f.Float64Var(&spread, "spread", 0.0, "per-device value spread as a ratio, e.g. 0.1 = ±10%")
	f.Float64Var(&noise, "noise", 0.0, "random noise per sample as a ratio, e.g. 0.05 = ±5%")
	f.Float64Var(&anomalyRate, "anomaly-rate", 0.0, "probability of injecting an anomaly per point, e.g. 0.02 = 2%")
	f.Float64Var(&anomalyFactor, "anomaly-factor", 3.0, "anomaly magnitude: spike = value × factor, drop = value / factor")
	f.Float64Var(&dropoutRate, "dropout-rate", 0.0, "probability of skipping a point, e.g. 0.05 = 5% dropout")
	f.BoolVar(&realtime, "realtime", false, "emit one point per step interval using wall-clock time")
	f.Int64Var(&seed, "seed", 0, "random seed for reproducible output (0 = random); batch mode only")
	f.StringVar(&replayFile, "replay-file", "", "replay a JSON-lines file through the configured sink")
	f.Float64Var(&rate, "rate", 0, "maximum points per second across all devices (0 = unlimited)")
	f.StringSliceVar(&deviceNameList, "device-names", nil, "explicit device names, comma-separated (overrides --device and --devices)")

	// Periodic curves
	f.Float64Var(&cosMin, "min", 10, "minimum value (cos, sawtooth, square)")
	f.Float64Var(&cosMax, "max", 25, "maximum value (cos, sawtooth, square)")
	f.StringVar(&cosPeriod, "period", "1d", "period (cos, sawtooth, square), e.g. 1d, 12h")
	f.Float64Var(&dutyCycle, "duty-cycle", 0.5, "fraction of period in high state, e.g. 0.3 = 30% on (square)")

	// Linear
	f.Float64Var(&linearFirst, "first", 0, "starting value (linear)")
	f.Float64Var(&linearLast, "last", 1, "ending value (linear)")

	// Walk
	f.Float64Var(&walkStart, "walk-start", 20.0, "starting value (walk)")
	f.Float64Var(&walkStep, "walk-step", 0.5, "max delta per sample (walk)")
	f.Float64Var(&walkBias, "walk-bias", 0.0, "directional drift per step, negative = downward (walk)")
	f.Float64Var(&walkMin, "walk-min", 15.0, "lower clamp; clamping disabled when walk-min == walk-max (walk)")
	f.Float64Var(&walkMax, "walk-max", 35.0, "upper clamp; clamping disabled when walk-min == walk-max (walk)")

	// InfluxDB
	f.StringVar(&influxdbURL, "influxdb-url", "http://localhost:8086", "InfluxDB server URL (--output influxdb)")
	f.StringVar(&influxdbToken, "influxdb-token", "", "InfluxDB API token (--output influxdb)")
	f.StringVar(&influxdbOrg, "influxdb-org", "", "InfluxDB organisation (--output influxdb)")
	f.StringVar(&influxdbBucket, "influxdb-bucket", "genx", "InfluxDB bucket (--output influxdb)")

	// Geo
	f.Float64Var(&geoLat, "geo-lat", 48.8566, "starting latitude (geo)")
	f.Float64Var(&geoLon, "geo-lon", 2.3522, "starting longitude (geo)")
	f.Float64Var(&geoSpeed, "geo-speed", 10.0, "speed in m/s (geo)")
	f.Float64Var(&geoBearing, "geo-bearing", 0.0, "initial bearing in degrees: 0=N, 90=E, 180=S, 270=W (geo)")
	f.Float64Var(&geoDrift, "geo-drift", 15.0, "max random bearing change per step in degrees (geo)")

	// Output
	f.StringVar(&output, "output", "stdout", "output backend: stdout, webhook, nats, mqtt")
	f.StringVar(&format, "format", "json", "output format for stdout/file sinks: json, csv, or influx")
	f.BoolVar(&isoTime, "iso-time", false, "emit timestamp as ISO 8601 UTC string instead of Unix epoch")
	f.StringVar(&influxMeasurement, "influx-measurement", "genx", "InfluxDB measurement name (--format influx)")
	f.StringVar(&cloudEventSource, "cloudevent-source", "/genx", "CloudEvents source URI (--format cloudevent); device name is appended automatically")
	f.StringVar(&cloudEventType, "cloudevent-type", "io.genx.measurement", "CloudEvents type field (--format cloudevent)")

	// Prometheus pull
	f.IntVar(&prometheusPort, "prometheus-port", 9091, "port to expose /metrics on (--output prometheus)")
	f.StringVar(&prometheusMetric, "prometheus-metric", "genx", "base metric name (--output prometheus); multi-field appends _<fieldname>")
	f.StringVar(&payloadTemplate, "payload-template", "", "Go text/template string for JSON payload")
	f.StringVar(&payloadTemplateFile, "payload-template-file", "", "path to a Go text/template file for JSON payload")

	// Webhook
	f.StringVar(&webhookURL, "webhook-url", "", "webhook URL")
	f.StringVar(&webhookToken, "webhook-token", "", "bearer token for webhook Authorization header")

	// NATS
	f.StringVar(&natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
	f.StringVar(&natsSubject, "nats-subject", "genx", "NATS subject to publish to")
	f.StringVar(&natsUser, "nats-user", "", "NATS username")
	f.StringVar(&natsPassword, "nats-password", "", "NATS password")
	f.StringVar(&natsToken, "nats-token", "", "NATS authentication token")

	// MQTT
	f.StringVar(&mqttBroker, "mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	f.StringVar(&mqttTopic, "mqtt-topic", "genx", "MQTT topic to publish to")
	f.IntVar(&mqttQoS, "mqtt-qos", 0, "MQTT QoS level (0, 1, or 2)")
	f.StringVar(&mqttClientID, "mqtt-client-id", fmt.Sprintf("genx-%d", os.Getpid()), "MQTT client ID")
	f.StringVar(&mqttUser, "mqtt-user", "", "MQTT username")
	f.StringVar(&mqttPassword, "mqtt-password", "", "MQTT password")
	f.StringVar(&mqttCACert, "mqtt-ca-cert", "", "CA certificate for verifying the broker's TLS certificate")
	f.StringVar(&mqttCert, "mqtt-cert", "", "client certificate for mTLS authentication")
	f.StringVar(&mqttKey, "mqtt-key", "", "client private key for mTLS authentication")
	f.BoolVar(&mqttTLSInsecure, "mqtt-tls-insecure", false, "skip broker TLS certificate verification (testing only)")

	// File
	f.StringVar(&filePath, "file-path", "", "base path for file sink output (e.g. data.jsonl)")
	f.StringVar(&fileMaxSize, "file-max-size", "", "rotate file when it reaches this size (e.g. 10MB, 1GB)")
	f.StringVar(&fileMaxAge, "file-max-age", "", "rotate file after this duration (e.g. 1h, 30m)")

	// Kafka
	f.StringVar(&kafkaBrokers, "kafka-brokers", "localhost:9092", "comma-separated Kafka broker addresses")
	f.StringVar(&kafkaTopic, "kafka-topic", "genx", "Kafka topic to publish to")
	f.StringVar(&kafkaUsername, "kafka-username", "", "SASL/PLAIN username")
	f.StringVar(&kafkaPassword, "kafka-password", "", "SASL/PLAIN password")
	f.BoolVar(&kafkaTLS, "kafka-tls", false, "enable TLS (uses system cert pool)")
	f.BoolVar(&kafkaTLSInsecure, "kafka-tls-insecure", false, "skip broker TLS certificate verification (testing only)")

	// OTLP
	f.StringVar(&otlpEndpoint, "otlp-endpoint", "localhost:4317", "OTLP collector endpoint (host:port)")
	f.BoolVar(&otlpHTTP, "otlp-http", false, "use OTLP/HTTP instead of OTLP/gRPC")
	f.StringArrayVar(&otlpHeaders, "otlp-header", nil, "header to add to OTLP requests, repeatable (e.g. x-api-key=abc)")
	f.BoolVar(&otlpInsecure, "otlp-insecure", false, "disable TLS for the OTLP connection (plain-text)")
	f.StringVar(&otlpMetricName, "otlp-metric", "genx", "base metric name; multi-field appends .<fieldname>")

	// Annotate flags with groups for the help output.
	groups := []struct {
		name  string
		flags []string
	}{
		{"General", []string{"config", "generate-config", "type", "duration", "step", "device", "devices", "device-names", "realtime", "seed", "replay-file", "rate", "noise", "spread", "anomaly-rate", "anomaly-factor", "dropout-rate"}},
		{"Periodic curves (--type cos / sawtooth / square)", []string{"min", "max", "period", "duty-cycle"}},
		{"Linear curve (--type linear)", []string{"first", "last"}},
		{"Random walk (--type walk)", []string{"walk-start", "walk-step", "walk-bias", "walk-min", "walk-max"}},
		{"Geo (--type geo)", []string{"geo-lat", "geo-lon", "geo-speed", "geo-bearing", "geo-drift"}},
		{"InfluxDB (--output influxdb)", []string{"influxdb-url", "influxdb-token", "influxdb-org", "influxdb-bucket", "influx-measurement"}},
		{"Output", []string{"output", "format", "iso-time", "influx-measurement", "cloudevent-source", "cloudevent-type"}},
		{"Template", []string{"payload-template", "payload-template-file"}},
		{"Webhook (--output webhook)", []string{"webhook-url", "webhook-token"}},
		{"NATS (--output nats)", []string{"nats-url", "nats-subject", "nats-user", "nats-password", "nats-token"}},
		{"MQTT (--output mqtt)", []string{"mqtt-broker", "mqtt-topic", "mqtt-qos", "mqtt-client-id", "mqtt-user", "mqtt-password", "mqtt-ca-cert", "mqtt-cert", "mqtt-key", "mqtt-tls-insecure"}},
		{"File (--output file)", []string{"file-path", "file-max-size", "file-max-age"}},
		{"Kafka (--output kafka)", []string{"kafka-brokers", "kafka-topic", "kafka-username", "kafka-password", "kafka-tls", "kafka-tls-insecure"}},
		{"OTLP (--output otlp)", []string{"otlp-endpoint", "otlp-http", "otlp-header", "otlp-insecure", "otlp-metric"}},
		{"Prometheus pull (--output prometheus)", []string{"prometheus-port", "prometheus-metric"}},
	}

	for _, g := range groups {
		for _, name := range g.flags {
			if fl := f.Lookup(name); fl != nil {
				fl.Annotations = map[string][]string{"group": {g.name}}
			}
		}
	}

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), cmd.Long)
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n  genx [flags]\n\n")

		skippedDefaults := map[string]bool{"false": true, "0": true, "<nil>": true}
		for _, g := range groups {
			fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", g.name)
			for _, name := range g.flags {
				fl := f.Lookup(name)
				if fl == nil {
					continue
				}
				if d := fl.DefValue; d != "" && !skippedDefaults[d] {
					fmt.Fprintf(cmd.OutOrStdout(), "  --%-30s %s (default: %s)\n", fl.Name, fl.Usage, d)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  --%-30s %s\n", fl.Name, fl.Usage)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
