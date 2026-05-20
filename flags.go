package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// cliFlags holds all variables bound to CLI flags.
type cliFlags struct {
	// General
	configFile     string
	curveType      string
	duration       string
	step           string
	count          int
	device         string
	devices        int
	spread         float64
	noise          float64
	anomalyRate    float64
	anomalyFactor  float64
	dropoutRate    float64
	realtime       bool
	seed           int64
	replayFile     string
	rate           float64
	generateConfig bool
	deviceNameList []string

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

	// Output / format
	output            string
	format            string
	verbose           bool
	isoTime           bool
	influxMeasurement string
	cloudEventSource  string
	cloudEventType    string

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
	mqttDeviceCerts map[string]MqttDeviceCert

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

	// Prometheus
	prometheusPort   int
	prometheusMetric string

	// InfluxDB
	influxdbURL    string
	influxdbToken  string
	influxdbOrg    string
	influxdbBucket string
}

func registerFlags(cmd *cobra.Command, v *cliFlags) {
	f := cmd.Flags()

	// General
	f.StringVar(&v.configFile, "config", "", "path to YAML config file (CLI flags take precedence)")
	f.BoolVar(&v.generateConfig, "generate-config", false, "print a sample YAML config file and exit")
	f.StringVar(&v.curveType, "type", "walk", "curve type: cos, linear, log, exp, walk, sawtooth, square, geo")
	f.StringVar(&v.duration, "duration", "1d", "total duration (e.g. 2d, 6h, 30m); mutually exclusive with --count")
	f.StringVar(&v.step, "step", "1h", "sampling interval (e.g. 1h, 5m, 10s)")
	f.IntVar(&v.count, "count", 0, "number of points to emit per device (alternative to --duration; 0 = use --duration)")
	f.StringVar(&v.device, "device", "device", "device/sensor name (or prefix when --devices > 1)")
	f.IntVar(&v.devices, "devices", 1, "number of devices to simulate simultaneously")
	f.Float64Var(&v.spread, "spread", 0.0, "per-device value spread as a ratio, e.g. 0.1 = ±10%")
	f.Float64Var(&v.noise, "noise", 0.0, "random noise per sample as a ratio, e.g. 0.05 = ±5%")
	f.Float64Var(&v.anomalyRate, "anomaly-rate", 0.0, "probability of injecting an anomaly per point, e.g. 0.02 = 2%")
	f.Float64Var(&v.anomalyFactor, "anomaly-factor", 3.0, "anomaly magnitude: spike = value × factor, drop = value / factor")
	f.Float64Var(&v.dropoutRate, "dropout-rate", 0.0, "probability of skipping a point, e.g. 0.05 = 5% dropout")
	f.BoolVar(&v.realtime, "realtime", false, "emit one point per step interval using wall-clock time")
	f.Int64Var(&v.seed, "seed", 0, "random seed for reproducible output (0 = random); batch mode only")
	f.StringVar(&v.replayFile, "replay-file", "", "replay a JSON-lines file through the configured sink")
	f.Float64Var(&v.rate, "rate", 0, "maximum points per second across all devices (0 = unlimited)")
	f.StringSliceVar(&v.deviceNameList, "device-names", nil, "explicit device names, comma-separated (overrides --device and --devices)")

	// Periodic curves
	f.Float64Var(&v.cosMin, "min", 10, "minimum value (cos, sawtooth, square)")
	f.Float64Var(&v.cosMax, "max", 25, "maximum value (cos, sawtooth, square)")
	f.StringVar(&v.cosPeriod, "period", "1d", "period (cos, sawtooth, square), e.g. 1d, 12h")
	f.Float64Var(&v.dutyCycle, "duty-cycle", 0.5, "fraction of period in high state, e.g. 0.3 = 30% on (square)")

	// Linear
	f.Float64Var(&v.linearFirst, "first", 0, "starting value (linear)")
	f.Float64Var(&v.linearLast, "last", 1, "ending value (linear)")

	// Walk
	f.Float64Var(&v.walkStart, "walk-start", 20.0, "starting value (walk)")
	f.Float64Var(&v.walkStep, "walk-step", 0.5, "max delta per sample (walk)")
	f.Float64Var(&v.walkBias, "walk-bias", 0.0, "directional drift per step, negative = downward (walk)")
	f.Float64Var(&v.walkMin, "walk-min", 15.0, "lower clamp; clamping disabled when walk-min == walk-max (walk)")
	f.Float64Var(&v.walkMax, "walk-max", 35.0, "upper clamp; clamping disabled when walk-min == walk-max (walk)")

	// Geo
	f.Float64Var(&v.geoLat, "geo-lat", 48.8566, "starting latitude (geo)")
	f.Float64Var(&v.geoLon, "geo-lon", 2.3522, "starting longitude (geo)")
	f.Float64Var(&v.geoSpeed, "geo-speed", 10.0, "speed in m/s (geo)")
	f.Float64Var(&v.geoBearing, "geo-bearing", 0.0, "initial bearing in degrees: 0=N, 90=E, 180=S, 270=W (geo)")
	f.Float64Var(&v.geoDrift, "geo-drift", 15.0, "max random bearing change per step in degrees (geo)")

	// Output / format
	f.StringVar(&v.output, "output", "stdout", "output backend: stdout, webhook, nats, mqtt")
	f.StringVar(&v.format, "format", "json", "output format for stdout/file sinks: json, csv, or influx")
	f.BoolVar(&v.verbose, "verbose", false, "print each payload with [OK]/[KO] to stderr (useful when output is not stdout)")
	f.BoolVar(&v.isoTime, "iso-time", false, "emit timestamp as ISO 8601 UTC string instead of Unix epoch")
	f.StringVar(&v.influxMeasurement, "influx-measurement", "genx", "InfluxDB measurement name (--format influx)")
	f.StringVar(&v.cloudEventSource, "cloudevent-source", "/genx", "CloudEvents source URI (--format cloudevent); device name is appended automatically")
	f.StringVar(&v.cloudEventType, "cloudevent-type", "io.genx.measurement", "CloudEvents type field (--format cloudevent)")

	// Prometheus
	f.IntVar(&v.prometheusPort, "prometheus-port", 9091, "port to expose /metrics on (--output prometheus)")
	f.StringVar(&v.prometheusMetric, "prometheus-metric", "genx", "base metric name (--output prometheus); multi-field appends _<fieldname>")

	// Template
	f.StringVar(&v.payloadTemplate, "payload-template", "", "Go text/template string for JSON payload")
	f.StringVar(&v.payloadTemplateFile, "payload-template-file", "", "path to a Go text/template file for JSON payload")

	// Webhook
	f.StringVar(&v.webhookURL, "webhook-url", "", "webhook URL")
	f.StringVar(&v.webhookToken, "webhook-token", "", "bearer token for webhook Authorization header")

	// NATS
	f.StringVar(&v.natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
	f.StringVar(&v.natsSubject, "nats-subject", "genx", "NATS subject to publish to")
	f.StringVar(&v.natsUser, "nats-user", "", "NATS username")
	f.StringVar(&v.natsPassword, "nats-password", "", "NATS password")
	f.StringVar(&v.natsToken, "nats-token", "", "NATS authentication token")

	// MQTT
	f.StringVar(&v.mqttBroker, "mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	f.StringVar(&v.mqttTopic, "mqtt-topic", "genx", "MQTT topic to publish to")
	f.IntVar(&v.mqttQoS, "mqtt-qos", 0, "MQTT QoS level (0, 1, or 2)")
	f.StringVar(&v.mqttClientID, "mqtt-client-id", fmt.Sprintf("genx-%d", os.Getpid()), "MQTT client ID")
	f.StringVar(&v.mqttUser, "mqtt-user", "", "MQTT username")
	f.StringVar(&v.mqttPassword, "mqtt-password", "", "MQTT password")
	f.StringVar(&v.mqttCACert, "mqtt-ca-cert", "", "CA certificate for verifying the broker's TLS certificate")
	f.StringVar(&v.mqttCert, "mqtt-cert", "", "client certificate for mTLS authentication")
	f.StringVar(&v.mqttKey, "mqtt-key", "", "client private key for mTLS authentication")
	f.BoolVar(&v.mqttTLSInsecure, "mqtt-tls-insecure", false, "skip broker TLS certificate verification (testing only)")

	// File
	f.StringVar(&v.filePath, "file-path", "", "base path for file sink output (e.g. data.jsonl)")
	f.StringVar(&v.fileMaxSize, "file-max-size", "", "rotate file when it reaches this size (e.g. 10MB, 1GB)")
	f.StringVar(&v.fileMaxAge, "file-max-age", "", "rotate file after this duration (e.g. 1h, 30m)")

	// Kafka
	f.StringVar(&v.kafkaBrokers, "kafka-brokers", "localhost:9092", "comma-separated Kafka broker addresses")
	f.StringVar(&v.kafkaTopic, "kafka-topic", "genx", "Kafka topic to publish to")
	f.StringVar(&v.kafkaUsername, "kafka-username", "", "SASL/PLAIN username")
	f.StringVar(&v.kafkaPassword, "kafka-password", "", "SASL/PLAIN password")
	f.BoolVar(&v.kafkaTLS, "kafka-tls", false, "enable TLS (uses system cert pool)")
	f.BoolVar(&v.kafkaTLSInsecure, "kafka-tls-insecure", false, "skip broker TLS certificate verification (testing only)")

	// OTLP
	f.StringVar(&v.otlpEndpoint, "otlp-endpoint", "localhost:4317", "OTLP collector endpoint (host:port)")
	f.BoolVar(&v.otlpHTTP, "otlp-http", false, "use OTLP/HTTP instead of OTLP/gRPC")
	f.StringArrayVar(&v.otlpHeaders, "otlp-header", nil, "header to add to OTLP requests, repeatable (e.g. x-api-key=abc)")
	f.BoolVar(&v.otlpInsecure, "otlp-insecure", false, "disable TLS for the OTLP connection (plain-text)")
	f.StringVar(&v.otlpMetricName, "otlp-metric", "genx", "base metric name; multi-field appends .<fieldname>")

	// InfluxDB
	f.StringVar(&v.influxdbURL, "influxdb-url", "http://localhost:8086", "InfluxDB server URL (--output influxdb)")
	f.StringVar(&v.influxdbToken, "influxdb-token", "", "InfluxDB API token (--output influxdb)")
	f.StringVar(&v.influxdbOrg, "influxdb-org", "", "InfluxDB organisation (--output influxdb)")
	f.StringVar(&v.influxdbBucket, "influxdb-bucket", "genx", "InfluxDB bucket (--output influxdb)")

	// Annotate flags with groups for the custom help output.
	groups := flagGroups()
	for _, g := range groups {
		for _, name := range g.flags {
			if fl := f.Lookup(name); fl != nil {
				fl.Annotations = map[string][]string{"group": {g.name}}
			}
		}
	}
}

// applyConfig copies values from cfg into v for any flag the user did not
// explicitly set on the CLI (changed reports whether a flag was set).
func applyConfig(cfg *Config, changed func(string) bool, v *cliFlags) {
	if cfg.Type != "" && !changed("type")                      { v.curveType = cfg.Type }
	if cfg.Duration != "" && !changed("duration")              { v.duration = cfg.Duration }
	if cfg.Step != "" && !changed("step")                      { v.step = cfg.Step }
	if cfg.Device != "" && !changed("device")                  { v.device = cfg.Device }
	if cfg.Devices != nil && !changed("devices")               { v.devices = *cfg.Devices }
	if cfg.Spread != nil && !changed("spread")                 { v.spread = *cfg.Spread }
	if cfg.Noise != nil && !changed("noise")                   { v.noise = *cfg.Noise }
	if cfg.AnomalyRate != nil && !changed("anomaly-rate")      { v.anomalyRate = *cfg.AnomalyRate }
	if cfg.AnomalyFactor != nil && !changed("anomaly-factor")  { v.anomalyFactor = *cfg.AnomalyFactor }
	if cfg.DropoutRate != nil && !changed("dropout-rate")      { v.dropoutRate = *cfg.DropoutRate }
	if cfg.Realtime != nil && !changed("realtime")             { v.realtime = *cfg.Realtime }
	if cfg.Seed != nil && !changed("seed")                     { v.seed = *cfg.Seed }
	if cfg.ReplayFile != "" && !changed("replay-file")         { v.replayFile = cfg.ReplayFile }
	if cfg.Count != nil && !changed("count")                   { v.count = *cfg.Count }
	if cfg.Rate != nil && !changed("rate")                     { v.rate = *cfg.Rate }

	if cfg.First != nil && !changed("first")                   { v.linearFirst = *cfg.First }
	if cfg.Last != nil && !changed("last")                     { v.linearLast = *cfg.Last }
	if cfg.Min != nil && !changed("min")                       { v.cosMin = *cfg.Min }
	if cfg.Max != nil && !changed("max")                       { v.cosMax = *cfg.Max }
	if cfg.Period != "" && !changed("period")                  { v.cosPeriod = cfg.Period }
	if cfg.DutyCycle != nil && !changed("duty-cycle")          { v.dutyCycle = *cfg.DutyCycle }
	if cfg.WalkStart != nil && !changed("walk-start")          { v.walkStart = *cfg.WalkStart }
	if cfg.WalkStep != nil && !changed("walk-step")            { v.walkStep = *cfg.WalkStep }
	if cfg.WalkBias != nil && !changed("walk-bias")            { v.walkBias = *cfg.WalkBias }
	if cfg.WalkMin != nil && !changed("walk-min")              { v.walkMin = *cfg.WalkMin }
	if cfg.WalkMax != nil && !changed("walk-max")              { v.walkMax = *cfg.WalkMax }
	if cfg.GeoLat != nil && !changed("geo-lat")                { v.geoLat = *cfg.GeoLat }
	if cfg.GeoLon != nil && !changed("geo-lon")                { v.geoLon = *cfg.GeoLon }
	if cfg.GeoSpeed != nil && !changed("geo-speed")            { v.geoSpeed = *cfg.GeoSpeed }
	if cfg.GeoBearing != nil && !changed("geo-bearing")        { v.geoBearing = *cfg.GeoBearing }
	if cfg.GeoDrift != nil && !changed("geo-drift")            { v.geoDrift = *cfg.GeoDrift }

	if cfg.Output != "" && !changed("output")                  { v.output = cfg.Output }
	if cfg.Format != "" && !changed("format")                  { v.format = cfg.Format }
	if cfg.Verbose != nil && !changed("verbose")               { v.verbose = *cfg.Verbose }
	if cfg.ISOTimestamp != nil && !changed("iso-time")         { v.isoTime = *cfg.ISOTimestamp }
	if cfg.InfluxMeasurement != "" && !changed("influx-measurement")     { v.influxMeasurement = cfg.InfluxMeasurement }
	if cfg.CloudEventSource != "" && !changed("cloudevent-source")       { v.cloudEventSource = cfg.CloudEventSource }
	if cfg.CloudEventType != "" && !changed("cloudevent-type")           { v.cloudEventType = cfg.CloudEventType }

	if cfg.WebhookURL != "" && !changed("webhook-url")         { v.webhookURL = cfg.WebhookURL }
	if cfg.WebhookToken != "" && !changed("webhook-token")     { v.webhookToken = cfg.WebhookToken }
	if cfg.NatsURL != "" && !changed("nats-url")               { v.natsURL = cfg.NatsURL }
	if cfg.NatsSubject != "" && !changed("nats-subject")       { v.natsSubject = cfg.NatsSubject }
	if cfg.NatsUser != "" && !changed("nats-user")             { v.natsUser = cfg.NatsUser }
	if cfg.NatsPassword != "" && !changed("nats-password")     { v.natsPassword = cfg.NatsPassword }
	if cfg.NatsToken != "" && !changed("nats-token")           { v.natsToken = cfg.NatsToken }
	if cfg.MqttBroker != "" && !changed("mqtt-broker")         { v.mqttBroker = cfg.MqttBroker }
	if cfg.MqttTopic != "" && !changed("mqtt-topic")           { v.mqttTopic = cfg.MqttTopic }
	if cfg.MqttQoS != nil && !changed("mqtt-qos")              { v.mqttQoS = *cfg.MqttQoS }
	if cfg.MqttClientID != "" && !changed("mqtt-client-id")    { v.mqttClientID = cfg.MqttClientID }
	if cfg.MqttUser != "" && !changed("mqtt-user")             { v.mqttUser = cfg.MqttUser }
	if cfg.MqttPassword != "" && !changed("mqtt-password")     { v.mqttPassword = cfg.MqttPassword }
	if cfg.MqttCACert != "" && !changed("mqtt-ca-cert")        { v.mqttCACert = cfg.MqttCACert }
	if cfg.MqttCert != "" && !changed("mqtt-cert")             { v.mqttCert = cfg.MqttCert }
	if cfg.MqttKey != "" && !changed("mqtt-key")               { v.mqttKey = cfg.MqttKey }
	if cfg.MqttTLSInsecure != nil && !changed("mqtt-tls-insecure") { v.mqttTLSInsecure = *cfg.MqttTLSInsecure }
	v.mqttDeviceCerts = cfg.MqttDeviceCerts

	if cfg.FilePath != "" && !changed("file-path")             { v.filePath = cfg.FilePath }
	if cfg.FileMaxSize != "" && !changed("file-max-size")      { v.fileMaxSize = cfg.FileMaxSize }
	if cfg.FileMaxAge != "" && !changed("file-max-age")        { v.fileMaxAge = cfg.FileMaxAge }
	if cfg.KafkaBrokers != "" && !changed("kafka-brokers")     { v.kafkaBrokers = cfg.KafkaBrokers }
	if cfg.KafkaTopic != "" && !changed("kafka-topic")         { v.kafkaTopic = cfg.KafkaTopic }
	if cfg.KafkaUsername != "" && !changed("kafka-username")   { v.kafkaUsername = cfg.KafkaUsername }
	if cfg.KafkaPassword != "" && !changed("kafka-password")   { v.kafkaPassword = cfg.KafkaPassword }
	if cfg.KafkaTLS != nil && !changed("kafka-tls")            { v.kafkaTLS = *cfg.KafkaTLS }
	if cfg.KafkaTLSInsecure != nil && !changed("kafka-tls-insecure") { v.kafkaTLSInsecure = *cfg.KafkaTLSInsecure }
	if cfg.PrometheusPort != nil && !changed("prometheus-port")      { v.prometheusPort = *cfg.PrometheusPort }
	if cfg.PrometheusMetric != "" && !changed("prometheus-metric")   { v.prometheusMetric = cfg.PrometheusMetric }
	if cfg.OTLPEndpoint != "" && !changed("otlp-endpoint")     { v.otlpEndpoint = cfg.OTLPEndpoint }
	if cfg.OTLPInsecure != nil && !changed("otlp-insecure")    { v.otlpInsecure = *cfg.OTLPInsecure }
	if cfg.OTLPHTTP != nil && !changed("otlp-http")            { v.otlpHTTP = *cfg.OTLPHTTP }
	if cfg.OTLPMetricName != "" && !changed("otlp-metric")     { v.otlpMetricName = cfg.OTLPMetricName }
	if len(cfg.OTLPHeaders) > 0 && !changed("otlp-header")     { v.otlpHeaders = cfg.OTLPHeaders }
	if cfg.InfluxDBURL != "" && !changed("influxdb-url")        { v.influxdbURL = cfg.InfluxDBURL }
	if cfg.InfluxDBToken != "" && !changed("influxdb-token")    { v.influxdbToken = cfg.InfluxDBToken }
	if cfg.InfluxDBOrg != "" && !changed("influxdb-org")        { v.influxdbOrg = cfg.InfluxDBOrg }
	if cfg.InfluxDBBucket != "" && !changed("influxdb-bucket")  { v.influxdbBucket = cfg.InfluxDBBucket }
	if len(cfg.DeviceNames) > 0 && !changed("device-names")     { v.deviceNameList = cfg.DeviceNames }
	if cfg.PayloadTemplate != "" && !changed("payload-template")          { v.payloadTemplate = cfg.PayloadTemplate }
	if cfg.PayloadTemplateFile != "" && !changed("payload-template-file") { v.payloadTemplateFile = cfg.PayloadTemplateFile }
}

type flagGroup struct {
	name  string
	flags []string
}

func flagGroups() []flagGroup {
	return []flagGroup{
		{"General", []string{"config", "generate-config", "type", "duration", "count", "step", "device", "devices", "device-names", "realtime", "seed", "replay-file", "rate", "noise", "spread", "anomaly-rate", "anomaly-factor", "dropout-rate"}},
		{"Periodic curves (--type cos / sawtooth / square)", []string{"min", "max", "period", "duty-cycle"}},
		{"Linear curve (--type linear)", []string{"first", "last"}},
		{"Random walk (--type walk)", []string{"walk-start", "walk-step", "walk-bias", "walk-min", "walk-max"}},
		{"Geo (--type geo)", []string{"geo-lat", "geo-lon", "geo-speed", "geo-bearing", "geo-drift"}},
		{"InfluxDB (--output influxdb)", []string{"influxdb-url", "influxdb-token", "influxdb-org", "influxdb-bucket", "influx-measurement"}},
		{"Output", []string{"output", "format", "verbose", "iso-time", "influx-measurement", "cloudevent-source", "cloudevent-type"}},
		{"Template", []string{"payload-template", "payload-template-file"}},
		{"Webhook (--output webhook)", []string{"webhook-url", "webhook-token"}},
		{"NATS (--output nats)", []string{"nats-url", "nats-subject", "nats-user", "nats-password", "nats-token"}},
		{"MQTT (--output mqtt)", []string{"mqtt-broker", "mqtt-topic", "mqtt-qos", "mqtt-client-id", "mqtt-user", "mqtt-password", "mqtt-ca-cert", "mqtt-cert", "mqtt-key", "mqtt-tls-insecure"}},
		{"File (--output file)", []string{"file-path", "file-max-size", "file-max-age"}},
		{"Kafka (--output kafka)", []string{"kafka-brokers", "kafka-topic", "kafka-username", "kafka-password", "kafka-tls", "kafka-tls-insecure"}},
		{"OTLP (--output otlp)", []string{"otlp-endpoint", "otlp-http", "otlp-header", "otlp-insecure", "otlp-metric"}},
		{"Prometheus pull (--output prometheus)", []string{"prometheus-port", "prometheus-metric"}},
	}
}

// defaultCLIFlags returns a cliFlags populated with the same defaults as
// registerFlags, without requiring a cobra command to be initialised.
func defaultCLIFlags() *cliFlags {
	return &cliFlags{
		curveType:         "walk",
		duration:          "1d",
		step:              "1h",
		device:            "device",
		devices:           1,
		anomalyFactor:     3.0,
		cosPeriod:         "1d",
		dutyCycle:         0.5,
		cosMin:            10,
		cosMax:            25,
		linearLast:        1,
		walkStart:         20.0,
		walkStep:          0.5,
		walkMin:           15.0,
		walkMax:           35.0,
		geoLat:            48.8566,
		geoLon:            2.3522,
		geoSpeed:          10.0,
		geoDrift:          15.0,
		output:            "stdout",
		format:            "json",
		influxMeasurement: "genx",
		cloudEventSource:  "/genx",
		cloudEventType:    "io.genx.measurement",
		prometheusPort:    9091,
		prometheusMetric:  "genx",
		mqttBroker:        "tcp://localhost:1883",
		mqttTopic:         "genx",
		natsURL:           "nats://localhost:4222",
		natsSubject:       "genx",
		kafkaBrokers:      "localhost:9092",
		kafkaTopic:        "genx",
		otlpEndpoint:      "localhost:4317",
		otlpMetricName:    "genx",
		influxdbURL:       "http://localhost:8086",
		influxdbBucket:    "genx",
	}
}

func setupHelp(rootCmd *cobra.Command) {
	f := rootCmd.Flags()
	groups := flagGroups()
	skippedDefaults := map[string]bool{"false": true, "0": true, "<nil>": true}

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// For subcommands (validate, completion, …) use simple default rendering.
		if cmd != rootCmd {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Short)
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n  %s\n\n", cmd.UseLine())
			if cmd.HasAvailableSubCommands() {
				fmt.Fprintf(cmd.OutOrStdout(), "Available Commands:\n")
				for _, sub := range cmd.Commands() {
					if !sub.Hidden {
						fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", sub.Name(), sub.Short)
					}
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			if cmd.HasAvailableFlags() {
				fmt.Fprintf(cmd.OutOrStdout(), "Flags:\n%s\n", cmd.Flags().FlagUsages())
			}
			return
		}

		// Root command: custom grouped help.
		fmt.Fprintln(cmd.OutOrStdout(), cmd.Long)
		fmt.Fprintln(cmd.OutOrStdout())

		if subs := rootCmd.Commands(); len(subs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Commands:\n")
			for _, sub := range subs {
				if !sub.Hidden {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", sub.Name(), sub.Short)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n  genx [flags]\n\n")

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
}

func registerCompletions(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"walk", "cos", "linear", "log", "exp", "sawtooth", "square", "geo"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("output", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"stdout", "webhook", "nats", "mqtt", "file", "kafka", "otlp", "prometheus", "influxdb"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("format", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "csv", "influx", "cloudevent"}, cobra.ShellCompDirectiveNoFileComp
	})
}
