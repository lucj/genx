package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"text/template"
	"time"
)

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

// verboseSink wraps any Sink and prints each payload with [OK] or [KO] to w.
type verboseSink struct {
	inner  Sink
	render Renderer
	w      io.Writer
}

func (s *verboseSink) Send(dp DataPoint) error {
	err := s.inner.Send(dp)
	payload, rerr := s.render(dp)
	if rerr != nil {
		payload = []byte("<render error>")
	}
	if err != nil {
		fmt.Fprintf(s.w, "[KO] %s  %v\n", payload, err)
	} else {
		fmt.Fprintf(s.w, "[OK] %s\n", payload)
	}
	return err
}

func (s *verboseSink) Close() error { return s.inner.Close() }

// startHTTPServer binds to the given port and starts an HTTP server.
// The port is reserved before this function returns, so any bind error is
// reported immediately rather than via a timing probe.
// srv.Addr holds the actual bound address (useful when port == 0).
func startHTTPServer(port int, handler http.Handler) (*http.Server, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	srv := &http.Server{
		Addr:         ln.Addr().String(),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	return srv, nil
}

// compileTopic returns a function that resolves a topic/subject pattern for a
// given DataPoint. Patterns containing "{{" are treated as Go templates;
// plain strings are returned as-is without any allocation.
func compileTopic(pattern string) (func(DataPoint) string, error) {
	if !strings.Contains(pattern, "{{") {
		return func(DataPoint) string { return pattern }, nil
	}
	tmpl, err := template.New("topic").Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid topic template %q: %w", pattern, err)
	}
	return func(dp DataPoint) string {
		var buf bytes.Buffer
		_ = tmpl.Execute(&buf, dp)
		return buf.String()
	}, nil
}
