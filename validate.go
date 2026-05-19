package main

import "fmt"

// validateParams checks that all resolved flag values are within acceptable
// ranges. Called after config file values are merged so it covers both CLI
// flags and config file fields in one pass.
func validateParams(v *cliFlags) error {
	if v.anomalyRate < 0 || v.anomalyRate > 1 {
		return fmt.Errorf("anomaly-rate must be between 0 and 1, got %g", v.anomalyRate)
	}
	if v.anomalyRate > 0 && v.anomalyFactor <= 1 {
		return fmt.Errorf("anomaly-factor must be > 1 when anomaly-rate > 0, got %g", v.anomalyFactor)
	}
	if v.dropoutRate < 0 || v.dropoutRate > 1 {
		return fmt.Errorf("dropout-rate must be between 0 and 1, got %g", v.dropoutRate)
	}
	if v.noise < 0 {
		return fmt.Errorf("noise must be >= 0, got %g", v.noise)
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
	if pp.anomalyRate < 0 || pp.anomalyRate > 1 {
		return fmt.Errorf("anomaly-rate must be between 0 and 1, got %g", pp.anomalyRate)
	}
	if pp.anomalyRate > 0 && pp.anomalyFactor <= 1 {
		return fmt.Errorf("anomaly-factor must be > 1 when anomaly-rate > 0, got %g", pp.anomalyFactor)
	}
	if pp.dropoutRate < 0 || pp.dropoutRate > 1 {
		return fmt.Errorf("dropout-rate must be between 0 and 1, got %g", pp.dropoutRate)
	}
	if pp.noise < 0 {
		return fmt.Errorf("noise must be >= 0, got %g", pp.noise)
	}
	return nil
}
