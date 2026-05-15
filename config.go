package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds values loaded from a YAML config file.
// Pointer fields distinguish "not set" from a zero value.
// String fields use "" as the "not set" sentinel.
type Config struct {
	Type     string   `yaml:"type"`
	Duration string   `yaml:"duration"`
	Step     string   `yaml:"step"`
	Device   string   `yaml:"device"`
	Devices    *int     `yaml:"devices"`
	Spread     *float64 `yaml:"spread"`
	Noise      *float64 `yaml:"noise"`
	Realtime   *bool    `yaml:"realtime"`
	Seed       *int64   `yaml:"seed"`
	ReplayFile string   `yaml:"replay-file"`

	First  *float64 `yaml:"first"`
	Last   *float64 `yaml:"last"`
	Min    *float64 `yaml:"min"`
	Max    *float64 `yaml:"max"`
	Period string   `yaml:"period"`
	// Walk-specific
	WalkStart *float64 `yaml:"walk-start"`
	WalkStep  *float64 `yaml:"walk-step"`
	WalkBias  *float64 `yaml:"walk-bias"`
	WalkMin   *float64 `yaml:"walk-min"`
	WalkMax   *float64 `yaml:"walk-max"`

	Output       string `yaml:"output"`
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
	MqttUser     string `yaml:"mqtt-user"`
	MqttPassword string `yaml:"mqtt-password"`

	PayloadTemplate     string `yaml:"payload-template"`
	PayloadTemplateFile string `yaml:"payload-template-file"`

	// Multi-field mode: when set, each key becomes a field in the payload.
	// Curve flags (--type, --min, etc.) are ignored when Fields is present.
	Fields map[string]FieldConfig `yaml:"fields"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
