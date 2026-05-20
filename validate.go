package main

import "fmt"

// validateAnomalyDropoutNoise checks the shared constraints that apply to
// both top-level flags and per-phase scenario parameters.
func validateAnomalyDropoutNoise(anomalyRate, anomalyFactor, dropoutRate, noise float64) error {
	if anomalyRate < 0 || anomalyRate > 1 {
		return fmt.Errorf("anomaly-rate must be between 0 and 1, got %g", anomalyRate)
	}
	if anomalyRate > 0 && anomalyFactor <= 1 {
		return fmt.Errorf("anomaly-factor must be > 1 when anomaly-rate > 0, got %g", anomalyFactor)
	}
	if dropoutRate < 0 || dropoutRate > 1 {
		return fmt.Errorf("dropout-rate must be between 0 and 1, got %g", dropoutRate)
	}
	if noise < 0 {
		return fmt.Errorf("noise must be >= 0, got %g", noise)
	}
	return nil
}

// validateParams checks that all resolved flag values are within acceptable
// ranges. Called after config file values are merged so it covers both CLI
// flags and config file fields in one pass.
func validateParams(v *cliFlags) error {
	if err := validateAnomalyDropoutNoise(v.anomalyRate, v.anomalyFactor, v.dropoutRate, v.noise); err != nil {
		return err
	}
	if v.spread < 0 {
		return fmt.Errorf("spread must be >= 0, got %g", v.spread)
	}
	if v.rate < 0 {
		return fmt.Errorf("rate must be >= 0, got %g", v.rate)
	}
	if v.geoSpeed < 0 {
		return fmt.Errorf("geo-speed must be >= 0, got %g", v.geoSpeed)
	}
	if v.mqttQoS != 0 && v.mqttQoS != 1 && v.mqttQoS != 2 {
		return fmt.Errorf("mqtt-qos must be 0, 1, or 2, got %d", v.mqttQoS)
	}
	if v.prometheusPort < 1 || v.prometheusPort > 65535 {
		return fmt.Errorf("prometheus-port must be between 1 and 65535, got %d", v.prometheusPort)
	}
	if v.count < 0 {
		return fmt.Errorf("count must be >= 0, got %d", v.count)
	}
	return nil
}

// validatePhaseParams checks range constraints on a resolved phase.
func validatePhaseParams(pp phaseParams) error {
	return validateAnomalyDropoutNoise(pp.anomalyRate, pp.anomalyFactor, pp.dropoutRate, pp.noise)
}
