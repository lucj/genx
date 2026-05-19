package main

import "sync/atomic"

// DataPoint represents a single generated measurement.
// In single-field mode, Value is set and Fields is nil.
// In multi-field mode, Fields is set and Value is nil.
type DataPoint struct {
	Device    string             `json:"device"`
	Timestamp int64              `json:"timestamp"`
	Value     *float64           `json:"value,omitempty"`
	Fields    map[string]float64 `json:"fields,omitempty"`
}

// Sink is the interface implemented by all output backends.
type Sink interface {
	Send(dp DataPoint) error
	Close() error
}

// statsSink wraps any Sink and counts successful sends and errors.
type statsSink struct {
	inner  Sink
	sent   atomic.Int64
	errors atomic.Int64
}

func (s *statsSink) Send(dp DataPoint) error {
	err := s.inner.Send(dp)
	if err != nil {
		s.errors.Add(1)
	} else {
		s.sent.Add(1)
	}
	return err
}

func (s *statsSink) Close() error { return s.inner.Close() }
