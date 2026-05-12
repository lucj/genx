package main

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

// NatsSink publishes each data point to a NATS subject.
type NatsSink struct {
	nc      *nats.Conn
	subject string
}

func NewNatsSink(url, subject string) (*NatsSink, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsSink{nc: nc, subject: subject}, nil
}

func (s *NatsSink) Send(dp DataPoint) error {
	b, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	return s.nc.Publish(s.subject, b)
}

func (s *NatsSink) Close() error {
	if err := s.nc.Flush(); err != nil {
		return err
	}
	return s.nc.Drain()
}
