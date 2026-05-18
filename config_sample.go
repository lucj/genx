package main

import "fmt"

const sampleConfig = `# genx sample configuration
# All options can also be set via CLI flags; CLI flags take precedence.
# Generate this file: genx --generate-config > config.yaml

# ---------------------------------------------------------------------------
# Curve
# ---------------------------------------------------------------------------
type: cos        # cos | linear | log | exp | walk
duration: 1h     # total duration  — units: d, h, m, s  (e.g. 2d, 30m)
step: 1m         # sampling interval
device: sensor   # device name, or prefix when devices > 1
devices: 1       # number of devices to simulate simultaneously
spread: 0        # per-device value spread ratio  (0.1 = ±10%)
noise: 0         # per-sample noise ratio          (0.05 = ±5%)
anomaly-rate: 0  # probability of injecting a spike or drop per point (0.02 = 2%)
anomaly-factor: 3 # anomaly magnitude: spike = value × factor, drop = value / factor
dropout-rate: 0  # probability of skipping a point entirely (0.05 = 5%)
realtime: false  # emit one point per step using the real wall clock
seed: 0          # RNG seed for reproducible output (0 = random)

# Cosine parameters (type: cos)
min: 10
max: 25
period: 1d

# Linear parameters (type: linear)
# first: 0
# last: 100

# Random walk parameters (type: walk)
# walk-start: 100
# walk-step: 1
# walk-bias: 0    # per-step drift; negative = downward trend
# walk-min: 0     # lower clamp (clamping disabled when walk-min == walk-max)
# walk-max: 0     # upper clamp

# ---------------------------------------------------------------------------
# Output sink
# ---------------------------------------------------------------------------
output: stdout   # stdout | webhook | nats | mqtt | file | kafka

# Kafka
# kafka-brokers: localhost:9092   # comma-separated broker addresses
# kafka-topic: genx
# kafka-username: alice
# kafka-password: secret
# kafka-tls: false
# kafka-tls-insecure: false

# File
# file-path: output.jsonl
# file-max-size: 10MB   # rotate when file exceeds this size (K/KB/M/MB/G/GB)
# file-max-age: 1h      # rotate after this duration (d/h/m/s)

# Webhook
# webhook-url: http://localhost:8080
# webhook-token: mysecrettoken

# NATS
# nats-url: nats://localhost:4222
# nats-subject: genx
# nats-user: alice
# nats-password: secret
# nats-token: mysecrettoken

# MQTT
# mqtt-broker: tcp://localhost:1883
# mqtt-topic: genx
# mqtt-qos: 0
# mqtt-client-id: genx-client
# mqtt-user: alice
# mqtt-password: secret
# TLS — use ssl:// broker URL when specifying cert options
# mqtt-ca-cert: ca.crt
# mqtt-cert: client.crt
# mqtt-key: client.key
# mqtt-tls-insecure: false
# Per-device mTLS certificates (fleet mode, YAML only):
# mqtt-device-certs:
#   sensor-0:
#     cert: certs/sensor-0.crt
#     key:  certs/sensor-0.key
#   sensor-1:
#     cert: certs/sensor-1.crt
#     key:  certs/sensor-1.key

# ---------------------------------------------------------------------------
# Payload template (optional — overrides default JSON shape)
# ---------------------------------------------------------------------------
# payload-template: '{"id":"{{.Device}}","ts":{{.Timestamp}},"val":{{.Value}}}'
# payload-template-file: template.json

# ---------------------------------------------------------------------------
# Multi-field mode (overrides type/min/max/etc. when fields is present)
# ---------------------------------------------------------------------------
# fields:
#   temperature:
#     type: cos
#     min: 18
#     max: 26
#     period: 12h
#   humidity:
#     type: cos
#     min: 40
#     max: 80
#     period: 8h
#   pressure:
#     type: linear
#     first: 1010
#     last: 1015
`

func printSampleConfig() {
	fmt.Print(sampleConfig)
}
