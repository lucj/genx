package main

import "github.com/nats-io/nats.go"

// NatsSink publishes each data point to a NATS subject.
type NatsSink struct {
	nc      *nats.Conn
	subject string
	render  Renderer
}

func NewNatsSink(url, subject, user, password, token string, render Renderer) (*NatsSink, error) {
	opts := []nats.Option{}
	switch {
	case user != "":
		opts = append(opts, nats.UserInfo(user, password))
	case token != "":
		opts = append(opts, nats.Token(token))
	}
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	return &NatsSink{nc: nc, subject: subject, render: render}, nil
}

func (s *NatsSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	return s.nc.Publish(s.subject, b)
}

// Close drains the connection, which flushes pending messages before disconnecting.
func (s *NatsSink) Close() error {
	return s.nc.Drain()
}
