package main

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds values loaded from a YAML config file.
// Pointer fields distinguish "not set" from a zero value.
// String fields use "" as the "not set" sentinel.
type Config struct {
	Type     string   `yaml:"type"`
	Duration string   `yaml:"duration"`
	From     string   `yaml:"from"`
	Step     string   `yaml:"step"`
	Device      string   `yaml:"device"`
	Devices     *int     `yaml:"devices"`
	DeviceNames []string `yaml:"device-names"`
	Spread     *float64 `yaml:"spread"`
	Noise         *float64 `yaml:"noise"`
	AnomalyRate   *float64 `yaml:"anomaly-rate"`
	AnomalyFactor *float64 `yaml:"anomaly-factor"`
	DropoutRate   *float64 `yaml:"dropout-rate"`
	Count         *int     `yaml:"count"`
	Realtime      *bool    `yaml:"realtime"`
	Seed       *int64   `yaml:"seed"`
	ReplayFile string   `yaml:"replay-file"`
	Rate       *float64 `yaml:"rate"`

	First     *float64 `yaml:"first"`
	Last      *float64 `yaml:"last"`
	Min       *float64 `yaml:"min"`
	Max       *float64 `yaml:"max"`
	Period    string   `yaml:"period"`
	DutyCycle *float64 `yaml:"duty-cycle"`
	// Walk-specific
	WalkStart *float64 `yaml:"walk-start"`
	WalkStep  *float64 `yaml:"walk-step"`
	WalkBias  *float64 `yaml:"walk-bias"`
	WalkMin   *float64 `yaml:"walk-min"`
	WalkMax   *float64 `yaml:"walk-max"`
	// Geo-specific
	GeoLat     *float64 `yaml:"geo-lat"`
	GeoLon     *float64 `yaml:"geo-lon"`
	GeoSpeed   *float64 `yaml:"geo-speed"`
	GeoBearing *float64 `yaml:"geo-bearing"`
	GeoDrift   *float64 `yaml:"geo-drift"`

	Output             string `yaml:"output"`
	Format             string `yaml:"format"`
	Verbose            *bool  `yaml:"verbose"`
	ISOTimestamp       *bool  `yaml:"iso-time"`
	InfluxMeasurement  string `yaml:"influx-measurement"`
	CloudEventSource   string `yaml:"cloudevent-source"`
	CloudEventType     string `yaml:"cloudevent-type"`
	InfluxDBURL        string `yaml:"influxdb-url"`
	InfluxDBToken      string `yaml:"influxdb-token"`
	InfluxDBOrg        string `yaml:"influxdb-org"`
	InfluxDBBucket     string `yaml:"influxdb-bucket"`
	WebhookURL   string `yaml:"webhook-url"`
	WebhookToken string `yaml:"webhook-token"`
	NatsURL      string `yaml:"nats-url"`
	NatsSubject  string `yaml:"nats-subject"`
	NatsUser     string `yaml:"nats-user"`
	NatsPassword string `yaml:"nats-password"`
	NatsToken    string `yaml:"nats-token"`
	MqttBroker   string `yaml:"mqtt-broker"`
	MqttTopic    string `yaml:"mqtt-topic"`
	MqttQoS      *int   `yaml:"mqtt-qos"`
	MqttClientID string `yaml:"mqtt-client-id"`
	MqttUser        string `yaml:"mqtt-user"`
	MqttPassword    string `yaml:"mqtt-password"`
	MqttCACert      string `yaml:"mqtt-ca-cert"`
	MqttCert        string `yaml:"mqtt-cert"`
	MqttKey         string `yaml:"mqtt-key"`
	MqttTLSInsecure *bool  `yaml:"mqtt-tls-insecure"`
	// MqttDeviceCerts maps device name to its own client cert/key pair.
	// The shared --mqtt-ca-cert still applies to all per-device connections.
	MqttDeviceCerts map[string]MqttDeviceCert `yaml:"mqtt-device-certs"`

	FilePath    string `yaml:"file-path"`
	FileMaxSize string `yaml:"file-max-size"`
	FileMaxAge  string `yaml:"file-max-age"`

	KafkaBrokers     string `yaml:"kafka-brokers"`
	KafkaTopic       string `yaml:"kafka-topic"`
	KafkaUsername    string `yaml:"kafka-username"`
	KafkaPassword    string `yaml:"kafka-password"`
	KafkaTLS         *bool  `yaml:"kafka-tls"`
	KafkaTLSInsecure *bool  `yaml:"kafka-tls-insecure"`

	PrometheusPort   *int   `yaml:"prometheus-port"`
	PrometheusMetric string `yaml:"prometheus-metric"`

	HTTPPort   *int `yaml:"http-port"`
	HTTPBuffer *int `yaml:"http-buffer"`

	OTLPEndpoint   string   `yaml:"otlp-endpoint"`
	OTLPHTTP       *bool    `yaml:"otlp-http"`
	OTLPHeaders    []string `yaml:"otlp-headers"`
	OTLPInsecure   *bool    `yaml:"otlp-insecure"`
	OTLPMetricName string   `yaml:"otlp-metric"`

	PayloadTemplate     string `yaml:"payload-template"`
	PayloadTemplateFile string `yaml:"payload-template-file"`

	// Multi-field mode: when set, each key becomes a field in the payload.
	// Curve flags (--type, --min, etc.) are ignored when Fields is present.
	Fields map[string]FieldConfig `yaml:"fields"`

	// Scenario mode: a list of phases executed in sequence.
	// Incompatible with replay-file and top-level fields.
	Scenario []PhaseConfig `yaml:"scenario"`
}

// PhaseConfig defines one phase of a scenario. Each phase inherits global
// defaults and overrides only the fields it sets.
type PhaseConfig struct {
	Duration      string   `yaml:"duration"`
	Type          string   `yaml:"type"`
	Step          string   `yaml:"step"`
	Min           *float64 `yaml:"min"`
	Max           *float64 `yaml:"max"`
	Period        string   `yaml:"period"`
	DutyCycle     *float64 `yaml:"duty-cycle"`
	First         *float64 `yaml:"first"`
	Last          *float64 `yaml:"last"`
	WalkStart     *float64 `yaml:"walk-start"`
	WalkStep      *float64 `yaml:"walk-step"`
	WalkBias      *float64 `yaml:"walk-bias"`
	WalkMin       *float64 `yaml:"walk-min"`
	WalkMax       *float64 `yaml:"walk-max"`
	Noise         *float64 `yaml:"noise"`
	AnomalyRate   *float64 `yaml:"anomaly-rate"`
	AnomalyFactor *float64 `yaml:"anomaly-factor"`
	DropoutRate   *float64 `yaml:"dropout-rate"`
}

// MqttDeviceCert holds the client cert/key pair for a single device.
type MqttDeviceCert struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

func LoadConfig(path string) (*Config, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	var cfg Config
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
