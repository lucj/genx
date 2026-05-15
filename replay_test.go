package main

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func writeTempReplayFile(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp("", "genx-replay-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lines {
		fmt.Fprintln(f, l)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestReplayBatchSingleField(t *testing.T) {
	lines := []string{
		`{"device":"sensor","timestamp":1000,"value":10.5}`,
		`{"device":"sensor","timestamp":2000,"value":11.0}`,
		`{"device":"sensor","timestamp":3000,"value":11.5}`,
	}
	path := writeTempReplayFile(t, lines)
	sink := &captureSink{}

	runReplay(context.Background(), path, sink, false, 0)

	if len(sink.points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(sink.points))
	}
	if sink.points[0].Device != "sensor" {
		t.Errorf("expected device %q, got %q", "sensor", sink.points[0].Device)
	}
	if sink.points[0].Value == nil || *sink.points[0].Value != 10.5 {
		t.Errorf("expected value 10.5, got %v", sink.points[0].Value)
	}
}

func TestReplayBatchMultiField(t *testing.T) {
	lines := []string{
		`{"device":"sensor","timestamp":1000,"fields":{"temperature":22.4,"humidity":60.0}}`,
		`{"device":"sensor","timestamp":2000,"fields":{"temperature":22.6,"humidity":59.5}}`,
	}
	path := writeTempReplayFile(t, lines)
	sink := &captureSink{}

	runReplay(context.Background(), path, sink, false, 0)

	if len(sink.points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(sink.points))
	}
	if sink.points[0].Fields["temperature"] != 22.4 {
		t.Errorf("expected temperature 22.4, got %f", sink.points[0].Fields["temperature"])
	}
}

func TestReplaySkipsInvalidLines(t *testing.T) {
	lines := []string{
		`{"device":"sensor","timestamp":1000,"value":10.5}`,
		`not valid json`,
		`{"device":"sensor","timestamp":3000,"value":11.5}`,
	}
	path := writeTempReplayFile(t, lines)
	sink := &captureSink{}

	runReplay(context.Background(), path, sink, false, 0)

	if len(sink.points) != 2 {
		t.Fatalf("expected 2 valid points (invalid line skipped), got %d", len(sink.points))
	}
}

func TestReplaySkipsEmptyLines(t *testing.T) {
	lines := []string{
		`{"device":"sensor","timestamp":1000,"value":10.5}`,
		``,
		`{"device":"sensor","timestamp":2000,"value":11.0}`,
	}
	path := writeTempReplayFile(t, lines)
	sink := &captureSink{}

	runReplay(context.Background(), path, sink, false, 0)

	if len(sink.points) != 2 {
		t.Fatalf("expected 2 points (empty line skipped), got %d", len(sink.points))
	}
}

func TestReplayPreservesTimestampInBatchMode(t *testing.T) {
	lines := []string{
		`{"device":"sensor","timestamp":9999,"value":1.0}`,
	}
	path := writeTempReplayFile(t, lines)
	sink := &captureSink{}

	runReplay(context.Background(), path, sink, false, 0)

	if sink.points[0].Timestamp != 9999 {
		t.Errorf("batch mode should preserve original timestamp, got %d", sink.points[0].Timestamp)
	}
}
