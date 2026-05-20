package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file without running",
		Long:  "Load and validate a YAML config file, printing a summary of what would run. Does not connect to any sink or emit data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(configFile)
			if err != nil {
				fmt.Printf("✗ Config: %v\n", err)
				return err
			}
			fmt.Printf("✓ Config: %s\n", configFile)

			v := defaultCLIFlags()
			applyConfig(cfg, func(string) bool { return false }, v)

			if err := validateParams(v); err != nil {
				fmt.Printf("✗ %v\n", err)
				return err
			}

			// Resolve device names.
			var deviceNames []string
			if len(v.deviceNameList) > 0 {
				deviceNames = v.deviceNameList
				v.devices = len(deviceNames)
			} else {
				if v.devices < 1 {
					v.devices = 1
				}
				deviceNames = make([]string, v.devices)
				for i := range deviceNames {
					if v.devices == 1 {
						deviceNames[i] = v.device
					} else {
						deviceNames[i] = fmt.Sprintf("%s-%d", v.device, i)
					}
				}
			}
			printDeviceSummary(deviceNames)

			printOutputSummary(v)

			// Replay mode.
			if v.replayFile != "" {
				fmt.Printf("✓ Mode: replay from %s\n", v.replayFile)
				fmt.Println("✓ All checks passed")
				return nil
			}

			// Resolve and show start time.
			startTs, err := ParseFromTime(v.from)
			if err != nil {
				fmt.Printf("✗ from: %v\n", err)
				return err
			}
			if v.from != "" {
				fmt.Printf("✓ From: %s\n", time.Unix(startTs, 0).UTC().Format(time.RFC3339))
			}

			// Resolve step (shared across modes).
			stepSeconds, err := GetSeconds(v.step)
			if err != nil {
				fmt.Printf("✗ step: %v\n", err)
				return err
			}

			var totalPoints int

			switch {
			case cfg != nil && len(cfg.Scenario) > 0:
				totalPoints, err = validateScenarioSummary(v, cfg, stepSeconds, len(deviceNames))
				if err != nil {
					fmt.Printf("✗ %v\n", err)
					return err
				}

			case cfg != nil && len(cfg.Fields) > 0:
				totalPoints = validateMultiFieldSummary(v, cfg, stepSeconds, len(deviceNames))

			default:
				totalPoints, err = validateSingleFieldSummary(v, stepSeconds, len(deviceNames))
				if err != nil {
					fmt.Printf("✗ %v\n", err)
					return err
				}
			}

			fmt.Printf("✓ Total: ~%d points\n", totalPoints)
			fmt.Println("✓ All checks passed")
			return nil
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "path to YAML config file to validate")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

func printDeviceSummary(names []string) {
	switch {
	case len(names) == 1:
		fmt.Printf("✓ Device: %s\n", names[0])
	case len(names) <= 5:
		fmt.Printf("✓ Devices (%d): %s\n", len(names), strings.Join(names, ", "))
	default:
		fmt.Printf("✓ Devices (%d): %s … and %d more\n", len(names), strings.Join(names[:3], ", "), len(names)-3)
	}
}

func printOutputSummary(v *cliFlags) {
	switch v.output {
	case "stdout":
		fmt.Printf("✓ Output: stdout (format: %s)\n", v.format)
	case "webhook":
		fmt.Printf("✓ Output: webhook (url: %s, format: %s)\n", v.webhookURL, v.format)
	case "mqtt":
		fmt.Printf("✓ Output: mqtt (broker: %s, topic: %s)\n", v.mqttBroker, v.mqttTopic)
	case "nats":
		fmt.Printf("✓ Output: nats (url: %s, subject: %s)\n", v.natsURL, v.natsSubject)
	case "kafka":
		fmt.Printf("✓ Output: kafka (brokers: %s, topic: %s)\n", v.kafkaBrokers, v.kafkaTopic)
	case "influxdb":
		fmt.Printf("✓ Output: influxdb (url: %s, bucket: %s, measurement: %s)\n", v.influxdbURL, v.influxdbBucket, v.influxMeasurement)
	case "file":
		fmt.Printf("✓ Output: file (path: %s)\n", v.filePath)
	case "otlp":
		fmt.Printf("✓ Output: otlp (endpoint: %s)\n", v.otlpEndpoint)
	case "prometheus":
		fmt.Printf("✓ Output: prometheus (port: %d, metric: %s)\n", v.prometheusPort, v.prometheusMetric)
	case "http-server":
		fmt.Printf("✓ Output: http-server (port: %d, buffer: %d points)\n", v.httpPort, v.httpBuffer)
	default:
		fmt.Printf("✓ Output: %s\n", v.output)
	}
}

func validateScenarioSummary(v *cliFlags, cfg *Config, globalStep, devices int) (int, error) {
	fmt.Printf("✓ Mode: scenario (%d phases)\n", len(cfg.Scenario))
	total := 0
	for i, phase := range cfg.Scenario {
		pp, err := resolvePhase(v, phase)
		if err != nil {
			return 0, fmt.Errorf("scenario phase %d: %w", i+1, err)
		}
		pts := (pp.durationSeconds / pp.stepSeconds) * devices
		total += pts

		var desc string
		switch {
		case pp.dropoutRate == 1.0:
			desc = "dropout (no points)"
		case pp.curveType == "geo":
			desc = fmt.Sprintf("geo, %s, step %s", phase.Duration, stepStr(pp.stepSeconds))
		default:
			desc = fmt.Sprintf("%s, %s, step %s → %d pts/device", pp.curveType, phase.Duration, stepStr(pp.stepSeconds), pp.durationSeconds/pp.stepSeconds)
		}
		fmt.Printf("  Phase %d: %s\n", i+1, desc)
	}
	return total, nil
}

func validateMultiFieldSummary(v *cliFlags, cfg *Config, stepSeconds, devices int) int {
	fieldNames := make([]string, 0, len(cfg.Fields))
	for name := range cfg.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)
	fmt.Printf("✓ Mode: multi-field (%s)\n", strings.Join(fieldNames, ", "))

	itemCount := itemsFromFlags(v, stepSeconds)
	fmt.Printf("✓ Duration: %s, step %s → %d points/device\n", v.duration, stepStr(stepSeconds), itemCount)
	return itemCount * devices
}

func validateSingleFieldSummary(v *cliFlags, stepSeconds, devices int) (int, error) {
	// Validate period for periodic curves.
	switch v.curveType {
	case "cos", "sawtooth", "square":
		if _, err := GetSeconds(v.cosPeriod); err != nil {
			return 0, fmt.Errorf("invalid period: %w", err)
		}
	}
	itemCount := itemsFromFlags(v, stepSeconds)
	label := v.duration
	if v.count > 0 {
		label = fmt.Sprintf("%d points", v.count)
	}
	fmt.Printf("✓ Mode: %s, %s, step %s → %d points/device\n", v.curveType, label, stepStr(stepSeconds), itemCount)
	return itemCount * devices, nil
}

// itemsFromFlags resolves the point count per device from count or duration.
func itemsFromFlags(v *cliFlags, stepSeconds int) int {
	if v.count > 0 {
		return v.count
	}
	dur, err := GetSeconds(v.duration)
	if err != nil {
		return 0
	}
	return dur / stepSeconds
}

// stepStr formats a step in seconds back to a human-readable string.
func stepStr(s int) string {
	switch {
	case s%(24*3600) == 0:
		return fmt.Sprintf("%dd", s/(24*3600))
	case s%3600 == 0:
		return fmt.Sprintf("%dh", s/3600)
	case s%60 == 0:
		return fmt.Sprintf("%dm", s/60)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
