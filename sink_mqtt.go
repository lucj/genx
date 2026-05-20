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
// When per-device certs are configured, each device uses its own connection;
// all other devices share the default connection.
type MqttSink struct {
	defaultClient mqtt.Client
	deviceClients map[string]mqtt.Client // keyed by device name; nil when unused
	topicFn       func(DataPoint) string
	qos           byte
	render        Renderer
}

func NewMqttSink(broker, topic, clientID string, qos int, user, password, caFile, certFile, keyFile string, tlsInsecure bool, deviceCerts map[string]MqttDeviceCert, render Renderer) (*MqttSink, error) {
	topicFn, err := compileTopic(topic)
	if err != nil {
		return nil, err
	}

	defaultClient, err := newMQTTClient(broker, clientID, user, password, caFile, certFile, keyFile, tlsInsecure)
	if err != nil {
		return nil, err
	}

	var deviceClients map[string]mqtt.Client
	if len(deviceCerts) > 0 {
		deviceClients = make(map[string]mqtt.Client, len(deviceCerts))
		for device, dc := range deviceCerts {
			// Per-device connections inherit the shared CA and insecure flag,
			// but use the device-specific client cert/key.
			c, err := newMQTTClient(broker, clientID+"-"+device, user, password, caFile, dc.Cert, dc.Key, tlsInsecure)
			if err != nil {
				// Disconnect already-opened clients before returning.
				for _, opened := range deviceClients {
					opened.Disconnect(250)
				}
				defaultClient.Disconnect(250)
				return nil, fmt.Errorf("device %q: %w", device, err)
			}
			deviceClients[device] = c
		}
	}

	return &MqttSink{
		defaultClient: defaultClient,
		deviceClients: deviceClients,
		topicFn:       topicFn,
		qos:           byte(qos),
		render:        render,
	}, nil
}

// newMQTTClient creates and connects a single MQTT client.
func newMQTTClient(broker, clientID, user, password, caFile, certFile, keyFile string, tlsInsecure bool) (mqtt.Client, error) {
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
	return client, nil
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
	client := s.defaultClient
	if c, ok := s.deviceClients[dp.Device]; ok {
		client = c
	}
	token := client.Publish(s.topicFn(dp), s.qos, false, b)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT publish timed out")
	}
	return token.Error()
}

func (s *MqttSink) Close() error {
	disconnected := make(map[mqtt.Client]bool)
	for _, c := range s.deviceClients {
		if !disconnected[c] {
			c.Disconnect(250)
			disconnected[c] = true
		}
	}
	if !disconnected[s.defaultClient] {
		s.defaultClient.Disconnect(250)
	}
	return nil
}
