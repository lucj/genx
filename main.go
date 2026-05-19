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
	output             string
	webhookURL         string
	webhookToken       string
	webhookContentType string
	natsURL            string
	natsSubject        string
	natsUser           string
	natsPassword       string
	natsToken          string
	mqttBroker         string
	mqttTopic          string
	mqttClientID       string
	mqttQoS            int
	mqttUser           string
	mqttPassword       string
	mqttCACert         string
	mqttCert           string
	mqttKey            string
	mqttTLSInsecure    bool
	mqttDeviceCerts    map[string]MqttDeviceCert
	filePath           string
	fileMaxBytes       int64
	fileMaxAge         time.Duration
	kafkaBrokers       string
	kafkaTopic         string
	kafkaUsername      string
	kafkaPassword      string
	kafkaTLS           bool
	kafkaTLSInsecure   bool
	otlpEndpoint       string
	otlpHTTP           bool
	otlpHeaders        map[string]string
	otlpInsecure       bool
	otlpMetricName     string
	prometheusPort     int
	prometheusMetric   string
	influxdbURL        string
	influxdbToken      string
	influxdbOrg        string
	influxdbBucket     string
	influxMeasurement  string
	renderer           Renderer
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

// buildRenderer resolves the Renderer and webhook Content-Type from flag values.
func buildRenderer(v *cliFlags) (Renderer, string, error) {
	renderer := Renderer(JSONRenderer)
	if v.isoTime {
		renderer = ISOJSONRenderer
	}
	webhookCT := "application/json"
	switch v.format {
	case "csv":
		renderer = NewCSVRenderer(v.isoTime)
	case "influx":
		renderer = NewInfluxRenderer(v.influxMeasurement)
	case "cloudevent":
		renderer = NewCloudEventRenderer(v.cloudEventSource, v.cloudEventType, v.isoTime)
		webhookCT = "application/cloudevents+json"
	case "json", "":
		// already set above
	default:
		return nil, "", fmt.Errorf("unknown --format %q (use json, csv, influx, or cloudevent)", v.format)
	}
	if v.payloadTemplateFile != "" {
		raw, err := os.ReadFile(v.payloadTemplateFile)
		if err != nil {
			return nil, "", fmt.Errorf("cannot read payload-template-file: %w", err)
		}
		tmpl, err := template.New("payload").Parse(string(raw))
		if err != nil {
			return nil, "", fmt.Errorf("invalid payload template: %w", err)
		}
		renderer = NewTemplateRenderer(tmpl, v.isoTime)
	} else if v.payloadTemplate != "" {
		tmpl, err := template.New("payload").Parse(v.payloadTemplate)
		if err != nil {
			return nil, "", fmt.Errorf("invalid payload template: %w", err)
		}
		renderer = NewTemplateRenderer(tmpl, v.isoTime)
	}
	return renderer, webhookCT, nil
}

func main() {
	var v cliFlags

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

			if v.generateConfig {
				printSampleConfig()
				return nil
			}

			// Load config file and apply values for flags not explicitly set on the CLI.
			var cfg *Config
			if v.configFile != "" {
				c, err := LoadConfig(v.configFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cfg = c
				applyConfig(cfg, cmd.Flags().Changed, &v)
			}

			if err := validateParams(&v); err != nil {
				return err
			}

			// Infer output sink from sink-specific flags when --output was not set.
			if !cmd.Flags().Changed("output") {
				switch {
				case cmd.Flags().Changed("webhook-url"):
					v.output = "webhook"
				case cmd.Flags().Changed("nats-url"):
					v.output = "nats"
				case cmd.Flags().Changed("mqtt-broker"):
					v.output = "mqtt"
				case cmd.Flags().Changed("file-path"):
					v.output = "file"
				case cmd.Flags().Changed("kafka-brokers"):
					v.output = "kafka"
				case cmd.Flags().Changed("otlp-endpoint"):
					v.output = "otlp"
				case cmd.Flags().Changed("prometheus-port"):
					v.output = "prometheus"
				case cmd.Flags().Changed("influxdb-url"):
					v.output = "influxdb"
				}
			}

			renderer, webhookCT, err := buildRenderer(&v)
			if err != nil {
				return err
			}

			// Initialise RNG.
			rng := newRand()
			if v.seed != 0 {
				rng = seededRand(uint64(v.seed))
			}

			// Parse file sink rotation parameters.
			var fileMaxBytes int64
			if v.fileMaxSize != "" {
				fileMaxBytes, err = ParseSize(v.fileMaxSize)
				if err != nil {
					return fmt.Errorf("invalid --file-max-size: %w", err)
				}
			}
			var fileMaxAgeDur time.Duration
			if v.fileMaxAge != "" {
				ageSecs, err := GetSeconds(v.fileMaxAge)
				if err != nil {
					return fmt.Errorf("invalid --file-max-age: %w", err)
				}
				fileMaxAgeDur = time.Duration(ageSecs) * time.Second
			}

			// Parse --otlp-header key=value pairs into a map.
			otlpHeaderMap := make(map[string]string, len(v.otlpHeaders))
			for _, h := range v.otlpHeaders {
				k, val, ok := strings.Cut(h, "=")
				if !ok {
					return fmt.Errorf("invalid --otlp-header %q: expected key=value", h)
				}
				otlpHeaderMap[k] = val
			}

			// Build the output sink.
			sink, err := buildSink(sinkConfig{
				output:             v.output,
				webhookURL:         v.webhookURL,
				webhookToken:       v.webhookToken,
				webhookContentType: webhookCT,
				influxdbURL:        v.influxdbURL,
				influxdbToken:      v.influxdbToken,
				influxdbOrg:        v.influxdbOrg,
				influxdbBucket:     v.influxdbBucket,
				influxMeasurement:  v.influxMeasurement,
				natsURL:            v.natsURL,
				natsSubject:        v.natsSubject,
				natsUser:           v.natsUser,
				natsPassword:       v.natsPassword,
				natsToken:          v.natsToken,
				mqttBroker:         v.mqttBroker,
				mqttTopic:          v.mqttTopic,
				mqttClientID:       v.mqttClientID,
				mqttQoS:            v.mqttQoS,
				mqttUser:           v.mqttUser,
				mqttPassword:       v.mqttPassword,
				mqttCACert:         v.mqttCACert,
				mqttCert:           v.mqttCert,
				mqttKey:            v.mqttKey,
				mqttTLSInsecure:    v.mqttTLSInsecure,
				mqttDeviceCerts:    v.mqttDeviceCerts,
				filePath:           v.filePath,
				fileMaxBytes:       fileMaxBytes,
				fileMaxAge:         fileMaxAgeDur,
				kafkaBrokers:       v.kafkaBrokers,
				kafkaTopic:         v.kafkaTopic,
				kafkaUsername:      v.kafkaUsername,
				kafkaPassword:      v.kafkaPassword,
				kafkaTLS:           v.kafkaTLS,
				kafkaTLSInsecure:   v.kafkaTLSInsecure,
				otlpEndpoint:       v.otlpEndpoint,
				otlpHTTP:           v.otlpHTTP,
				otlpHeaders:        otlpHeaderMap,
				otlpInsecure:       v.otlpInsecure,
				otlpMetricName:     v.otlpMetricName,
				prometheusPort:     v.prometheusPort,
				prometheusMetric:   v.prometheusMetric,
				renderer:           renderer,
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
			if v.replayFile != "" {
				stepSeconds, err := GetSeconds(v.step)
				if err != nil {
					return fmt.Errorf("invalid --step: %w", err)
				}
				return runReplay(ctx, v.replayFile, sink, v.realtime, stepSeconds)
			}

			durationSeconds, err := GetSeconds(v.duration)
			if err != nil {
				return fmt.Errorf("invalid --duration: %w", err)
			}
			stepSeconds, err := GetSeconds(v.step)
			if err != nil {
				return fmt.Errorf("invalid --step: %w", err)
			}

			start := time.Now().Unix()

			// Resolve device names: explicit list takes precedence over --device / --devices.
			var deviceNames []string
			if len(v.deviceNameList) > 0 {
				if cmd.Flags().Changed("devices") && v.devices != len(v.deviceNameList) {
					return fmt.Errorf("--devices=%d conflicts with --device-names (%d names provided)", v.devices, len(v.deviceNameList))
				}
				deviceNames = v.deviceNameList
				v.devices = len(deviceNames)
			} else {
				if v.devices < 1 {
					return fmt.Errorf("--devices must be at least 1")
				}
				deviceNames = make([]string, v.devices)
				for i := range deviceNames {
					if v.devices == 1 {
						deviceNames[i] = v.device
					} else {
						deviceNames[i] = fmt.Sprintf("%s-%d", v.device, i)
					}
				}
			}

			itemCount := durationSeconds / stepSeconds

			// Scenario mode: phases executed in sequence.
			if cfg != nil && len(cfg.Scenario) > 0 {
				if v.replayFile != "" {
					return fmt.Errorf("scenario mode is incompatible with --replay-file")
				}
				if len(cfg.Fields) > 0 {
					return fmt.Errorf("scenario mode is incompatible with top-level fields")
				}
				return runScenario(ctx, rng, &v, cfg.Scenario, sink, deviceNames, start)
			}

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
				scales := make([]float64, v.devices)
				for i := range scales {
					scales[i] = 1.0
					if v.spread > 0 {
						scales[i] = 1.0 + v.spread*(2*rng.Float64()-1)
					}
				}
				if v.realtime {
					runRealtimeMulti(ctx, rng, fieldFns, scales, v.noise, v.anomalyRate, v.anomalyFactor, v.dropoutRate, sink, deviceNames, itemCount, stepSeconds, v.rate)
				} else {
					runBatchMulti(rng, fieldFns, scales, v.noise, v.anomalyRate, v.anomalyFactor, v.dropoutRate, sink, deviceNames, start, itemCount, stepSeconds, v.rate)
				}
				return nil
			}

			// Geo mode.
			if v.curveType == "geo" {
				walkers := make([]*GeoWalker, v.devices)
				for i := range walkers {
					walkers[i] = NewGeoWalker(v.geoLat, v.geoLon, v.geoBearing, v.geoSpeed, v.geoDrift)
				}
				if v.realtime {
					runRealtimeGeo(ctx, rng, walkers, sink, deviceNames, itemCount, stepSeconds, v.dropoutRate, v.rate)
				} else {
					runBatchGeo(rng, walkers, sink, deviceNames, start, itemCount, stepSeconds, v.dropoutRate, v.rate)
				}
				return nil
			}

			// Single-field mode.
			var baseFn func(float64) float64
			switch v.curveType {
			case "linear":
				baseFn = GetLinear(v.linearFirst, v.linearLast, start, durationSeconds)
			case "cos":
				periodSeconds, err := GetSeconds(v.cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				baseFn = GetCosinus(v.cosMin, v.cosMax, periodSeconds)
			case "sawtooth":
				periodSeconds, err := GetSeconds(v.cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				baseFn = GetSawtooth(v.cosMin, v.cosMax, start, periodSeconds)
			case "square":
				periodSeconds, err := GetSeconds(v.cosPeriod)
				if err != nil {
					return fmt.Errorf("invalid --period: %w", err)
				}
				if v.dutyCycle <= 0 || v.dutyCycle >= 1 {
					return fmt.Errorf("--duty-cycle must be between 0 and 1 (exclusive), got %g", v.dutyCycle)
				}
				baseFn = GetSquare(v.cosMin, v.cosMax, start, periodSeconds, v.dutyCycle)
			case "log":
				baseFn = GetLog(start)
			case "exp":
				baseFn = GetExp(start, durationSeconds)
			case "walk":
				// baseFn intentionally left nil; each device gets its own closure below.
			default:
				return fmt.Errorf("unknown curve type %q (use cos, linear, log, exp, walk, sawtooth, square, geo)", v.curveType)
			}

			fns := make([]func(float64) float64, v.devices)
			for i := range fns {
				scale := 1.0
				if v.spread > 0 {
					scale = 1.0 + v.spread*(2*rng.Float64()-1)
				}
				if v.curveType == "walk" {
					fns[i] = WithAnomaly(rng, WithNoise(rng, GetRandomWalk(rng, v.walkStart*scale, v.walkStep, v.walkBias, v.walkMin, v.walkMax), v.noise), v.anomalyRate, v.anomalyFactor)
				} else {
					fn := baseFn
					s := scale
					fns[i] = WithAnomaly(rng, WithNoise(rng, func(x float64) float64 { return fn(x) * s }, v.noise), v.anomalyRate, v.anomalyFactor)
				}
			}
			if v.realtime {
				runRealtime(ctx, rng, fns, sink, deviceNames, itemCount, stepSeconds, v.dropoutRate, v.rate)
			} else {
				runBatch(rng, fns, sink, deviceNames, start, itemCount, stepSeconds, v.dropoutRate, v.rate)
			}
			return nil
		},
	}

	registerFlags(rootCmd, &v)
	setupHelp(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
