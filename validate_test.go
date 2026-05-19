package main

import "testing"

func defaultFlags() *cliFlags {
	return &cliFlags{
		anomalyFactor:  3.0,
		prometheusPort: 9091,
		mqttQoS:        0,
		geoSpeed:       10,
	}
}

func TestValidateParamsOK(t *testing.T) {
	if err := validateParams(defaultFlags()); err != nil {
		t.Errorf("expected no error for valid defaults, got: %v", err)
	}
}

func TestValidateAnomalyRate(t *testing.T) {
	v := defaultFlags()
	v.anomalyRate = 1.5
	if err := validateParams(v); err == nil {
		t.Error("expected error for anomaly-rate > 1")
	}
	v.anomalyRate = -0.1
	if err := validateParams(v); err == nil {
		t.Error("expected error for anomaly-rate < 0")
	}
}

func TestValidateAnomalyFactorRequiresRate(t *testing.T) {
	v := defaultFlags()
	v.anomalyRate = 0.05
	v.anomalyFactor = 1.0
	if err := validateParams(v); err == nil {
		t.Error("expected error for anomaly-factor <= 1 with anomaly-rate > 0")
	}
}

func TestValidateDropoutRate(t *testing.T) {
	v := defaultFlags()
	v.dropoutRate = 1.5
	if err := validateParams(v); err == nil {
		t.Error("expected error for dropout-rate > 1")
	}
	v.dropoutRate = -0.1
	if err := validateParams(v); err == nil {
		t.Error("expected error for dropout-rate < 0")
	}
}

func TestValidateNoise(t *testing.T) {
	v := defaultFlags()
	v.noise = -0.05
	if err := validateParams(v); err == nil {
		t.Error("expected error for noise < 0")
	}
}

func TestValidateSpread(t *testing.T) {
	v := defaultFlags()
	v.spread = -0.1
	if err := validateParams(v); err == nil {
		t.Error("expected error for spread < 0")
	}
}

func TestValidateRate(t *testing.T) {
	v := defaultFlags()
	v.rate = -1
	if err := validateParams(v); err == nil {
		t.Error("expected error for rate < 0")
	}
}

func TestValidateGeoSpeed(t *testing.T) {
	v := defaultFlags()
	v.geoSpeed = -5
	if err := validateParams(v); err == nil {
		t.Error("expected error for geo-speed < 0")
	}
}

func TestValidateMqttQoS(t *testing.T) {
	v := defaultFlags()
	v.mqttQoS = 3
	if err := validateParams(v); err == nil {
		t.Error("expected error for mqtt-qos = 3")
	}
}

func TestValidatePrometheusPort(t *testing.T) {
	v := defaultFlags()
	v.prometheusPort = 0
	if err := validateParams(v); err == nil {
		t.Error("expected error for prometheus-port = 0")
	}
	v.prometheusPort = 99999
	if err := validateParams(v); err == nil {
		t.Error("expected error for prometheus-port > 65535")
	}
}

func TestValidatePhaseParamsOK(t *testing.T) {
	pp := phaseParams{anomalyFactor: 3.0}
	if err := validatePhaseParams(pp); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidatePhaseAnomalyRate(t *testing.T) {
	pp := phaseParams{anomalyRate: 2.0, anomalyFactor: 3.0}
	if err := validatePhaseParams(pp); err == nil {
		t.Error("expected error for anomaly-rate > 1 in phase")
	}
}

func TestValidatePhaseDropoutRate(t *testing.T) {
	pp := phaseParams{dropoutRate: -0.1}
	if err := validatePhaseParams(pp); err == nil {
		t.Error("expected error for dropout-rate < 0 in phase")
	}
}

func TestValidatePhaseNoise(t *testing.T) {
	pp := phaseParams{noise: -1.0}
	if err := validatePhaseParams(pp); err == nil {
		t.Error("expected error for noise < 0 in phase")
	}
}
