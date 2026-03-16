package main

import (
	"encoding/json"
	"fmt"
)

// StdoutSink prints each data point as JSON to stdout.
type StdoutSink struct{}

func NewStdoutSink() *StdoutSink { return &StdoutSink{} }

func (s *StdoutSink) Send(dp DataPoint) error {
	b, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (s *StdoutSink) Close() error { return nil }
