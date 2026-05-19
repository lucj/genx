package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestVerboseSinkLogsOK(t *testing.T) {
	var buf bytes.Buffer
	inner := &captureSink{}
	render := func(dp DataPoint) ([]byte, error) { return []byte(`{"device":"dev"}`), nil }
	s := &verboseSink{inner: inner, render: render, w: &buf}

	v := 1.0
	if err := s.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "[OK]") {
		t.Errorf("expected [OK] prefix, got %q", out)
	}
	if !strings.Contains(out, `{"device":"dev"}`) {
		t.Errorf("expected payload in output, got %q", out)
	}
}

func TestVerboseSinkLogsKO(t *testing.T) {
	var buf bytes.Buffer
	inner := &errorSink{}
	render := func(dp DataPoint) ([]byte, error) { return []byte(`{"device":"dev"}`), nil }
	s := &verboseSink{inner: inner, render: render, w: &buf}

	v := 1.0
	err := s.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v})
	if err == nil {
		t.Fatal("expected error from errorSink")
	}

	out := buf.String()
	if !strings.HasPrefix(out, "[KO]") {
		t.Errorf("expected [KO] prefix, got %q", out)
	}
	if !strings.Contains(out, "send failed") {
		t.Errorf("expected error message in output, got %q", out)
	}
}

func TestVerboseSinkPassesThroughError(t *testing.T) {
	var buf bytes.Buffer
	inner := &errorSink{}
	render := func(dp DataPoint) ([]byte, error) { return []byte("{}"), nil }
	s := &verboseSink{inner: inner, render: render, w: &buf}

	v := 1.0
	err := s.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v})
	if err == nil {
		t.Error("expected error to be passed through to caller")
	}
}

func TestVerboseSinkRenderError(t *testing.T) {
	var buf bytes.Buffer
	inner := &captureSink{}
	render := func(dp DataPoint) ([]byte, error) { return nil, fmt.Errorf("render failed") }
	s := &verboseSink{inner: inner, render: render, w: &buf}

	v := 1.0
	s.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v})

	if !strings.Contains(buf.String(), "<render error>") {
		t.Errorf("expected <render error> in output, got %q", buf.String())
	}
}

func TestVerboseSinkClose(t *testing.T) {
	inner := &captureSink{}
	s := &verboseSink{inner: inner, render: JSONRenderer, w: &bytes.Buffer{}}
	if err := s.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}
