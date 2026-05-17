package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yaml := `
type: linear
duration: 2h
step: 5m
device: sensor
devices: 10
spread: 0.1
realtime: true
first: 5.0
last: 50.0
output: nats
nats-url: nats://broker:4222
nats-subject: iot
`
	f, err := os.CreateTemp("", "genx-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(yaml)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Type != "linear" {
		t.Errorf("Type: got %q, want %q", cfg.Type, "linear")
	}
	if cfg.Duration != "2h" {
		t.Errorf("Duration: got %q, want %q", cfg.Duration, "2h")
	}
	if cfg.Devices == nil || *cfg.Devices != 10 {
		t.Errorf("Devices: got %v, want 10", cfg.Devices)
	}
	if cfg.Spread == nil || *cfg.Spread != 0.1 {
		t.Errorf("Spread: got %v, want 0.1", cfg.Spread)
	}
	if cfg.Realtime == nil || !*cfg.Realtime {
		t.Errorf("Realtime: got %v, want true", cfg.Realtime)
	}
	if cfg.First == nil || *cfg.First != 5.0 {
		t.Errorf("First: got %v, want 5.0", cfg.First)
	}
	if cfg.NatsURL != "nats://broker:4222" {
		t.Errorf("NatsURL: got %q, want %q", cfg.NatsURL, "nats://broker:4222")
	}
}

func TestLoadConfigStdin(t *testing.T) {
	yaml := "type: linear\nduration: 2h\n"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.WriteString(yaml)
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	cfg, err := LoadConfig("-")
	if err != nil {
		t.Fatalf("LoadConfig(\"-\") error: %v", err)
	}
	if cfg.Type != "linear" {
		t.Errorf("expected type linear, got %q", cfg.Type)
	}
	if cfg.Duration != "2h" {
		t.Errorf("expected duration 2h, got %q", cfg.Duration)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadConfigPartial(t *testing.T) {
	// Unset pointer fields must stay nil (no false zero-value overrides).
	yaml := `
type: cos
output: mqtt
`
	f, err := os.CreateTemp("", "genx-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(yaml)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Devices != nil {
		t.Errorf("Devices should be nil when absent from config, got %v", *cfg.Devices)
	}
	if cfg.Spread != nil {
		t.Errorf("Spread should be nil when absent from config, got %v", *cfg.Spread)
	}
	if cfg.Realtime != nil {
		t.Errorf("Realtime should be nil when absent from config, got %v", *cfg.Realtime)
	}
}
