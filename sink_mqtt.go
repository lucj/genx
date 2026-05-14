package main

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttSink publishes each data point to an MQTT topic.
type MqttSink struct {
	client mqtt.Client
	topic  string
	qos    byte
}

func NewMqttSink(broker, topic, clientID string, qos int, user, password string) (*MqttSink, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true)
	if user != "" {
		opts.SetUsername(user)
		opts.SetPassword(password)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("MQTT connection timed out")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return &MqttSink{client: client, topic: topic, qos: byte(qos)}, nil
}

func (s *MqttSink) Send(dp DataPoint) error {
	b, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	token := s.client.Publish(s.topic, s.qos, false, b)
	token.Wait()
	return token.Error()
}

func (s *MqttSink) Close() error {
	s.client.Disconnect(250)
	return nil
}
