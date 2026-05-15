package main

import "fmt"

// StdoutSink prints each data point as JSON to stdout.
type StdoutSink struct {
	render Renderer
}

func NewStdoutSink(render Renderer) *StdoutSink { return &StdoutSink{render: render} }

func (s *StdoutSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (s *StdoutSink) Close() error { return nil }
