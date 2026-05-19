package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp config: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	f.Close()
	return f.Name()
}

func runValidate(t *testing.T, configContent string) error {
	t.Helper()
	path := writeConfig(t, configContent)
	cmd := newValidateCmd()
	cmd.SetArgs([]string{"--config", path})
	return cmd.Execute()
}

func TestValidateCmdSimple(t *testing.T) {
	err := runValidate(t, `
type: cos
min: 18
max: 26
duration: 1h
step: 1m
`)
	if err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestValidateCmdInvalidAnomalyRate(t *testing.T) {
	err := runValidate(t, `anomaly-rate: 5.0`)
	if err == nil {
		t.Error("expected error for anomaly-rate > 1")
	}
}

func TestValidateCmdScenario(t *testing.T) {
	err := runValidate(t, `
device: sensor-1
step: 1m

scenario:
  - duration: 10m
    type: cos
    min: 20
    max: 25
  - duration: 5m
    dropout-rate: 1.0
`)
	if err != nil {
		t.Errorf("expected no error for valid scenario config, got: %v", err)
	}
}

func TestValidateCmdScenarioMissingDuration(t *testing.T) {
	err := runValidate(t, `
step: 1m
scenario:
  - type: cos
`)
	if err == nil {
		t.Error("expected error for scenario phase missing duration")
	}
}

func TestValidateCmdMultiField(t *testing.T) {
	err := runValidate(t, `
duration: 1h
step: 1m
fields:
  temperature:
    type: cos
    min: 18
    max: 26
  humidity:
    type: cos
    min: 40
    max: 80
`)
	if err != nil {
		t.Errorf("expected no error for valid multi-field config, got: %v", err)
	}
}

func TestValidateCmdBadConfigFile(t *testing.T) {
	cmd := newValidateCmd()
	cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "nonexistent.yaml")})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestValidateCmdDeviceNames(t *testing.T) {
	err := runValidate(t, `
device-names: [paris, london, tokyo]
type: cos
duration: 30m
step: 1m
`)
	if err != nil {
		t.Errorf("expected no error for device-names config, got: %v", err)
	}
}

func TestStepStr(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{60, "1m"},
		{3600, "1h"},
		{86400, "1d"},
		{30, "30s"},
		{120, "2m"},
	}
	for _, tc := range cases {
		if got := stepStr(tc.secs); got != tc.want {
			t.Errorf("stepStr(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}
