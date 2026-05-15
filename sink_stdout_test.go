package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestStdoutSinkSend(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	v := 42.5
	dp := DataPoint{Device: "dev1", Timestamp: 1000, Value: &v}
	sink := NewStdoutSink()
	if err := sink.Send(dp); err != nil {
		w.Close()
		os.Stdout = old
		t.Fatalf("Send returned error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	line := strings.TrimSpace(buf.String())

	var got DataPoint
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, line)
	}
	if got.Device != "dev1" || got.Timestamp != 1000 || got.Value == nil || *got.Value != 42.5 {
		t.Errorf("unexpected output: %s", line)
	}
}

func TestStdoutSinkClose(t *testing.T) {
	if err := NewStdoutSink().Close(); err != nil {
		t.Errorf("Close should return nil, got %v", err)
	}
}
