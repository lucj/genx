package main

import "github.com/nats-io/nats.go"

// NatsSink publishes each data point to a NATS subject.
type NatsSink struct {
	nc      *nats.Conn
	subject string
}

func NewNatsSink(url, subject, user, password, token string) (*NatsSink, error) {
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
	return &NatsSink{nc: nc, subject: subject}, nil
}

func (s *NatsSink) Send(dp DataPoint) error {
	b, err := renderPayload(dp)
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
