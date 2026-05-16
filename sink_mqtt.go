package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttSink publishes each data point to an MQTT topic.
type MqttSink struct {
	client mqtt.Client
	topic  string
	qos    byte
	render Renderer
}

func NewMqttSink(broker, topic, clientID string, qos int, user, password, caFile, certFile, keyFile string, tlsInsecure bool, render Renderer) (*MqttSink, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true)
	if user != "" {
		opts.SetUsername(user)
		opts.SetPassword(password)
	}
	if caFile != "" || certFile != "" || keyFile != "" || tlsInsecure {
		tlsCfg, err := buildMQTTTLSConfig(caFile, certFile, keyFile, tlsInsecure)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsCfg)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("MQTT connection timed out")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return &MqttSink{client: client, topic: topic, qos: byte(qos), render: render}, nil
}

func buildMQTTTLSConfig(caFile, certFile, keyFile string, insecure bool) (*tls.Config, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec

	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse CA cert %s", caFile)
		}
		tlsCfg.RootCAs = pool
	}

	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("--mqtt-cert and --mqtt-key must both be provided")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

func (s *MqttSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	token := s.client.Publish(s.topic, s.qos, false, b)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT publish timed out")
	}
	return token.Error()
}

func (s *MqttSink) Close() error {
	s.client.Disconnect(250)
	return nil
}
