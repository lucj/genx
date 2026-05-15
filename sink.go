package main

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
