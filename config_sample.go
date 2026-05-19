package main

import "fmt"

const sampleConfig = `# genx sample configuration
# All options can also be set via CLI flags; CLI flags take precedence.
# Generate this file: genx --generate-config > config.yaml

# ---------------------------------------------------------------------------
# Curve
# ---------------------------------------------------------------------------
type: cos        # cos | linear | log | exp | walk | sawtooth | square | geo
duration: 1h     # total duration  — units: d, h, m, s  (e.g. 2d, 30m)
step: 1m         # sampling interval
device: sensor   # device name, or prefix when devices > 1
devices: 1       # number of devices to simulate simultaneously
# device-names: [paris, london, tokyo]  # explicit names (overrides device/devices)
spread: 0        # per-device value spread ratio  (0.1 = ±10%)
noise: 0         # per-sample noise ratio          (0.05 = ±5%)
anomaly-rate: 0  # probability of injecting a spike or drop per point (0.02 = 2%)
anomaly-factor: 3 # anomaly magnitude: spike = value × factor, drop = value / factor
dropout-rate: 0  # probability of skipping a point entirely (0.05 = 5%)
realtime: false  # emit one point per step using the real wall clock
seed: 0          # RNG seed for reproducible output (0 = random)
rate: 0          # max points per second across all devices (0 = unlimited)
# count: 100     # emit exactly N points instead of using duration

# Cosine / sawtooth / square parameters
min: 10
max: 25
period: 1d
# duty-cycle: 0.5  # fraction of period in high state (square wave only)

# Linear parameters (type: linear)
# first: 0
# last: 100

# Random walk parameters (type: walk)
# walk-start: 20
# walk-step: 0.5
# walk-bias: 0    # per-step drift; negative = downward trend
# walk-min: 15    # lower clamp (clamping disabled when walk-min == walk-max)
# walk-max: 35    # upper clamp

# Geo parameters (type: geo) — GPS track simulation
# geo-lat: 48.8566    # starting latitude
# geo-lon: 2.3522     # starting longitude
# geo-speed: 10       # speed in m/s
# geo-bearing: 0      # initial bearing in degrees (0=N, 90=E, 180=S, 270=W)
# geo-drift: 15       # max random bearing change per step in degrees

# ---------------------------------------------------------------------------
# Scenario mode (phases executed in sequence; incompatible with fields)
# ---------------------------------------------------------------------------
# scenario:
#   - duration: 10m
#     type: cos
#     min: 20
#     max: 25
#   - duration: 5m      # sensor fault — all points dropped
#     dropout-rate: 1.0
#   - duration: 10m     # recovery
#     type: cos
#     min: 20
#     max: 25

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

# ---------------------------------------------------------------------------
# Output sink
# ---------------------------------------------------------------------------
output: stdout   # stdout | webhook | nats | mqtt | file | kafka | influxdb | otlp | prometheus
format: json     # json | csv | influx | cloudevent  (stdout and file sinks)
verbose: false   # print [OK]/[KO] <payload> to stderr for every point
iso-time: false  # emit timestamp as ISO 8601 UTC string instead of Unix epoch
influx-measurement: genx   # InfluxDB measurement name (--format influx)

# CloudEvents envelope (--format cloudevent)
# cloudevent-source: /genx
# cloudevent-type: io.genx.measurement

# ---------------------------------------------------------------------------
# Payload template (optional — overrides default JSON shape)
# ---------------------------------------------------------------------------
# payload-template: '{"id":"{{.Device}}","ts":{{.Timestamp}},"val":{{.Value}}}'
# payload-template-file: template.json

# ---------------------------------------------------------------------------
# Webhook (output: webhook)
# ---------------------------------------------------------------------------
# webhook-url: http://localhost:8080
# webhook-token: mysecrettoken

# ---------------------------------------------------------------------------
# NATS (output: nats)
# ---------------------------------------------------------------------------
# nats-url: nats://localhost:4222
# nats-subject: genx
# nats-user: alice
# nats-password: secret
# nats-token: mysecrettoken

# ---------------------------------------------------------------------------
# MQTT (output: mqtt)
# ---------------------------------------------------------------------------
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
# Kafka (output: kafka)
# ---------------------------------------------------------------------------
# kafka-brokers: localhost:9092   # comma-separated broker addresses
# kafka-topic: genx
# kafka-username: alice
# kafka-password: secret
# kafka-tls: false
# kafka-tls-insecure: false

# ---------------------------------------------------------------------------
# File (output: file)
# ---------------------------------------------------------------------------
# file-path: output.jsonl
# file-max-size: 10MB   # rotate when file exceeds this size (K/KB/M/MB/G/GB)
# file-max-age: 1h      # rotate after this duration (d/h/m/s)

# ---------------------------------------------------------------------------
# InfluxDB (output: influxdb)
# ---------------------------------------------------------------------------
# influxdb-url: http://localhost:8086
# influxdb-token: my-token
# influxdb-org: my-org
# influxdb-bucket: genx

# ---------------------------------------------------------------------------
# OpenTelemetry / OTLP (output: otlp)
# ---------------------------------------------------------------------------
# otlp-endpoint: localhost:4317
# otlp-http: false          # use OTLP/HTTP instead of gRPC
# otlp-insecure: false      # disable TLS
# otlp-headers:             # repeatable key=value headers
#   - x-api-key=my-key
# otlp-metric: genx         # base metric name

# ---------------------------------------------------------------------------
# Prometheus pull (output: prometheus)
# ---------------------------------------------------------------------------
# prometheus-port: 9091
# prometheus-metric: genx   # base metric name; multi-field appends _<fieldname>
`

func printSampleConfig() {
	fmt.Print(sampleConfig)
}
