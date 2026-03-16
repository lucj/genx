package main

// DataPoint represents a single generated measurement.
type DataPoint struct {
	Device    string  `json:"device"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// Sink is the interface implemented by all output backends.
type Sink interface {
	Send(dp DataPoint) error
	Close() error
}
