package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSinkWritesJSONLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	sink, err := NewFileSink(path, 0, 0, JSONRenderer)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	v := 42.0
	for i := int64(0); i < 3; i++ {
		if err := sink.Send(DataPoint{Device: "dev", Timestamp: 1000 + i, Value: &v}); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var points []DataPoint
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var dp DataPoint
		if err := json.Unmarshal(sc.Bytes(), &dp); err != nil {
			t.Fatalf("JSON decode: %v", err)
		}
		points = append(points, dp)
	}
	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}
}

func TestFileSinkNoRotation_UsesExactPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exact.jsonl")
	sink, _ := NewFileSink(path, 0, 0, JSONRenderer)
	v := 1.0
	sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: &v})
	sink.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at exact path %s: %v", path, err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) != 1 {
		t.Errorf("expected exactly 1 file, got %d: %v", len(files), files)
	}
}

func TestFileSinkSizeRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	// Max 1 byte forces a rotation before every send.
	sink, err := NewFileSink(path, 1, 0, JSONRenderer)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	v := 1.0
	dp := DataPoint{Device: "d", Timestamp: 1, Value: &v}
	for i := 0; i < 5; i++ {
		if err := sink.Send(dp); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}
	sink.Close()

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 {
		t.Errorf("expected multiple rotated files, got %d", len(files))
	}
}

func TestFileSinkAgeRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	// Rotate after 1ms — guaranteed to trigger between sends with a small sleep.
	sink, err := NewFileSink(path, 0, time.Millisecond, JSONRenderer)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	v := 1.0
	dp := DataPoint{Device: "d", Timestamp: 1, Value: &v}
	for i := 0; i < 3; i++ {
		if err := sink.Send(dp); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	sink.Close()

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 {
		t.Errorf("expected multiple rotated files, got %d", len(files))
	}
}

func TestFileSinkRotatedFilesContainValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	sink, _ := NewFileSink(path, 1, 0, JSONRenderer)
	v := 7.0
	for i := int64(0); i < 4; i++ {
		sink.Send(DataPoint{Device: "dev", Timestamp: i, Value: &v})
	}
	sink.Close()

	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	total := 0
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var dp DataPoint
			if err := json.Unmarshal(sc.Bytes(), &dp); err != nil {
				t.Errorf("invalid JSON in %s: %v", name, err)
			}
			total++
		}
		f.Close()
	}
	if total != 4 {
		t.Errorf("expected 4 total points across rotated files, got %d", total)
	}
}

func TestParseSizeValid(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"", 0},
		{"1024", 1024},
		{"1KB", 1024},
		{"1kb", 1024},
		{"10MB", 10 * 1024 * 1024},
		{"2GB", 2 * 1024 * 1024 * 1024},
		{"5M", 5 * 1024 * 1024},
		{"3K", 3 * 1024},
		{"1G", 1024 * 1024 * 1024},
	}
	for _, c := range cases {
		got, err := ParseSize(c.in)
		if err != nil {
			t.Errorf("ParseSize(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseSizeInvalid(t *testing.T) {
	for _, s := range []string{"abc", "10XB", "-1", "1.5MB"} {
		if _, err := ParseSize(s); err == nil {
			t.Errorf("ParseSize(%q) expected error, got nil", s)
		}
	}
}
