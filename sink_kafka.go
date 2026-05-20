package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// KafkaSink publishes each data point as a JSON message to a Kafka topic.
// The device name is used as the message key for consistent per-device partitioning.
type KafkaSink struct {
	writer  *kafka.Writer
	topicFn func(DataPoint) string
	render  Renderer
}

func NewKafkaSink(brokers, topic, username, password string, tlsEnabled, tlsInsecure bool, render Renderer) (*KafkaSink, error) {
	if brokers == "" {
		return nil, fmt.Errorf("--kafka-brokers must not be empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("--kafka-topic must not be empty")
	}

	topicFn, err := compileTopic(topic)
	if err != nil {
		return nil, err
	}

	brokerList := parseBrokers(brokers)

	// Topic is left empty so each message can carry its own topic, which
	// supports template patterns like "sensors/{{.Device}}".
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerList...),
		Balancer: &kafka.LeastBytes{},
	}

	if username != "" || tlsEnabled || tlsInsecure {
		t := &kafka.Transport{}
		if username != "" {
			t.SASL = plain.Mechanism{Username: username, Password: password}
		}
		if tlsEnabled || tlsInsecure {
			t.TLS = &tls.Config{InsecureSkipVerify: tlsInsecure} //nolint:gosec
		}
		w.Transport = t
	}

	return &KafkaSink{writer: w, topicFn: topicFn, render: render}, nil
}

func (s *KafkaSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	return s.writer.WriteMessages(context.Background(), kafka.Message{
		Topic: s.topicFn(dp),
		Key:   []byte(dp.Device),
		Value: b,
	})
}

func (s *KafkaSink) Close() error {
	return s.writer.Close()
}

// parseBrokers splits a comma-separated broker string and trims whitespace.
func parseBrokers(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if b := strings.TrimSpace(p); b != "" {
			out = append(out, b)
		}
	}
	return out
}
