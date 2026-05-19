# genx

**genx** is a lightweight time-series data generator. It emits synthetic measurements following a mathematical curve, useful for testing dashboards, messaging pipelines, or IoT simulators.

Each data point is output as JSON by default:
```json
{"device":"sensor1","timestamp":1715000000,"value":24.53}
```

## Installation

**Docker (no install required):**
```bash
docker run ghcr.io/lucj/genx [flags]
```

**Download a binary** from the [Releases page](https://github.com/lucj/genx/releases), extract it, and place `genx` somewhere on your `PATH`.

**Build from source:**
```bash
git clone https://github.com/lucj/genx.git
cd genx
go build -o genx .
```

## Quick start

Generate a cosine curve, one point every 15 minutes over 1 hour:
```bash
genx --type cos --min 18 --max 26 --duration 1h --step 15m
{"device":"device","timestamp":1715000000,"value":26.00}
{"device":"device","timestamp":1715054400,"value":22.00}
{"device":"device","timestamp":1715108800,"value":18.00}
{"device":"device","timestamp":1715163200,"value":22.00}
```

Use `--generate-config` to print a fully commented YAML template covering every option:
```bash
genx --generate-config > config.yaml
```

## Curve types

### Linear

Ramps from `--first` to `--last` over the total duration.

```bash
genx --type linear --duration 3d --first 10 --last 30 --step 6h
{"device":"device","timestamp":1715000000,"value":10.00}
{"device":"device","timestamp":1715021600,"value":11.67}
{"device":"device","timestamp":1715043200,"value":13.33}
...
```

### Random walk

Drifts by a random delta each sample. Useful for simulating battery drain, temperature drift, or stock prices.

```bash
genx --type walk --walk-start 100 --walk-step 2 --walk-bias -0.1 \
     --walk-min 0 --walk-max 120 --duration 1h --step 1m
{"device":"device","timestamp":1715000000,"value":100.00}
{"device":"device","timestamp":1715000060,"value":98.73}
...
```

`--walk-bias` adds a constant drift per step (negative = downward trend). `--walk-min` / `--walk-max` clamp the value; clamping is disabled when both are `0`.

### Cosine

Oscillates between `--min` and `--max` over the given `--period`.

```bash
genx --type cos --duration 2d --min 20 --max 30 --step 3h --period 1d
{"device":"device","timestamp":1715000000,"value":30.00}
{"device":"device","timestamp":1715010800,"value":27.50}
...
```

### Sawtooth

Ramps linearly from `--min` to `--max` over each `--period`, then resets — producing a /|/|/| waveform. Useful for simulating cyclic fill levels or ramp-up/reset processes.

```bash
genx --type sawtooth --min 0 --max 100 --period 1h --duration 3h --step 10m
{"device":"device","timestamp":1715000000,"value":0.00}
{"device":"device","timestamp":1715000600,"value":16.67}
...
```

### Square wave

Alternates between `--max` (high) and `--min` (low). `--duty-cycle` sets the fraction of each period spent in the high state (default `0.5` = 50% on). Useful for simulating on/off equipment cycles.

```bash
genx --type square --min 0 --max 1 --period 1h --duty-cycle 0.3 --duration 3h --step 5m
```

`--duty-cycle 0.3` means the output is high for 30% of each period and low for the remaining 70%.

### Logarithmic

Produces a slow-growing logarithmic curve (natural log of elapsed seconds).

```bash
genx --type log --duration 1d --step 2h
```

### Exponential

Grows exponentially over the duration.

```bash
genx --type exp --duration 6h --step 30m
```

## Fleet mode

Simulate multiple devices at once with `--devices`. Each device gets an independent value curve; `--spread` adds a per-device random offset so they don't all emit identical values.

```bash
genx --type cos --devices 3 --spread 0.1 --duration 1h --step 5m --realtime
{"device":"device-0","timestamp":1715000000,"value":24.10}
{"device":"device-1","timestamp":1715000000,"value":23.57}
{"device":"device-2","timestamp":1715000000,"value":25.02}
...
```

`--spread 0.1` means each device's values are randomly scaled by ±10%. Use `--device` to set the prefix (`--device sensor` → `sensor-0`, `sensor-1`, …).

## Noise & anomalies

Add realistic jitter with `--noise`, random spikes/drops with `--anomaly-rate`, and connectivity gaps with `--dropout-rate`:

```bash
genx --type cos --duration 1h --step 1m \
     --noise 0.05 --anomaly-rate 0.02 --anomaly-factor 5 --dropout-rate 0.03
```

- `--noise 0.05` — multiply each value by a random factor in `[0.95, 1.05]`
- `--anomaly-rate 0.02` — roughly 2% of points become spikes or drops
- `--dropout-rate 0.03` — roughly 3% of points are silently skipped

## Realtime mode

By default genx generates the full dataset instantly (batch mode). Add `--realtime` to emit one point per `--step` interval using the actual wall clock:

```bash
genx --type cos --min 18 --max 26 --duration 1h --step 10s --realtime
```

## Reproducible runs

Use `--seed` to fix the RNG so every run with the same flags produces identical output:

```bash
genx --type cos --noise 0.05 --devices 3 --spread 0.1 --seed 42 --duration 1h --step 5m
```

## Replay mode

Replay a previously recorded JSON-lines file through any configured sink:

```bash
# Record
genx --type cos --duration 1h --step 1m > recording.jsonl

# Replay to NATS in realtime
genx --replay-file recording.jsonl --output nats --nats-url nats://localhost:4222 \
     --realtime --step 1m
```

## Output formats

Use `--format` to control the serialisation format for `stdout` and `file` sinks.

### JSON (default)

```bash
genx --type cos --duration 1h --step 10m
{"device":"device","timestamp":1715000000,"value":26.00}
```

### CSV

```bash
genx --type cos --duration 1h --step 10m --format csv
device,timestamp,value
device,1715000000,26.00
device,1715000600,25.73
```

The header row is emitted once on the first point. In multi-field mode field columns are sorted alphabetically. Combine with `--iso-time` for human-readable timestamps.

### InfluxDB line protocol

```bash
genx --type cos --duration 1h --step 10m --format influx
genx,device=device value=26 1715000000000000000
genx,device=device value=25.73 1715000600000000000
```

Timestamps are emitted in nanoseconds as required by the line protocol. Pipe directly into `influx write`:

```bash
genx --type cos --devices 3 --step 10s --realtime --format influx \
  | influx write --bucket my-bucket --org my-org
```

Use `--influx-measurement` to override the measurement name (default: `genx`).

## Output sinks

By default data is written to stdout. Use `--output` to route it to a sink.

### Webhook

POSTs each data point as JSON to an HTTP endpoint:

```bash
genx --type cos --duration 1h --step 5m \
     --output webhook --webhook-url http://myserver/ingest \
     --webhook-token mysecrettoken
```

### NATS

Publishes to a NATS subject. Supports username/password and token authentication:

```bash
genx --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
     --type cos --duration 1h --step 1m
```

### MQTT

Publishes to an MQTT topic. Supports username/password and TLS/mTLS:

```bash
# Username / password
genx --type cos --duration 1h --step 5m \
     --output mqtt --mqtt-broker tcp://localhost:1883 --mqtt-topic home/temperature \
     --mqtt-user myuser --mqtt-password mypassword

# TLS
genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-topic sensors \
     --mqtt-ca-cert ca.crt --type cos --duration 1h --step 1m

# Mutual TLS (mTLS)
genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-topic sensors \
     --mqtt-ca-cert ca.crt --mqtt-cert client.crt --mqtt-key client.key \
     --type cos --duration 1h --step 1m
```

Per-device certificates (one TLS identity per simulated device) are supported via a YAML config file — see `genx --generate-config` for the `mqtt-device-certs` option.

### Kafka

Publishes to a Kafka topic. The device name is used as the message key:

```bash
genx --type cos --duration 1h --step 5m \
     --output kafka --kafka-brokers localhost:9092 --kafka-topic sensors

# SASL/PLAIN + TLS
genx --output kafka --kafka-brokers localhost:9092 --kafka-topic sensors \
     --kafka-username alice --kafka-password secret --kafka-tls \
     --type cos --duration 1h --step 1m --realtime
```

### File sink with rotation

Writes data to disk and optionally rotates files by size or age:

```bash
# Single file
genx --type cos --duration 1h --step 1m --file-path output.jsonl

# Rotate every 10 MB or every hour, whichever comes first
genx --type cos --duration 24h --step 1m --realtime \
     --file-path data.jsonl --file-max-size 10MB --file-max-age 1h
```

Rotated files are named with a UTC timestamp suffix: `data.20240506T120000.jsonl`. Supported size suffixes: `K`/`KB`, `M`/`MB`, `G`/`GB`.

### OpenTelemetry (OTLP)

Pushes metrics as OTLP gauges to any OpenTelemetry collector (Grafana Agent, OTel Collector, Honeycomb, Datadog, …). Each device name becomes a `device` attribute; multi-field payloads produce one gauge per field.

```bash
# gRPC (default)
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type cos --devices 3 --step 5s --realtime

# HTTP/protobuf
genx --output otlp --otlp-endpoint localhost:4318 --otlp-http --otlp-insecure \
     --type cos --devices 3 --step 5s --realtime

# Authenticated (e.g. Grafana Cloud, Honeycomb)
genx --output otlp --otlp-endpoint otlp.example.com:4317 \
     --otlp-header "x-api-key=your-key" \
     --type cos --step 10s --realtime
```

A ready-made Docker Compose stack (OTel Collector + Prometheus + Grafana) is in `examples/otlp/`:

```bash
cd examples/otlp
docker compose --profile demo up   # push mode: genx → OTel Collector → Prometheus → Grafana
```

Open <http://localhost:3000> to see the live dashboard.

### Prometheus (pull / scrape)

Starts an HTTP server and exposes the latest gauge values at `/metrics` in [Prometheus text exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/). Prometheus (or any compatible scraper) pulls the data on its own schedule — no push infrastructure required.

```bash
genx --output prometheus --prometheus-port 9091 \
     --type cos --devices 3 --step 5s --realtime

# Prometheus scrapes http://localhost:9091/metrics
```

Multi-field payloads produce one metric per field: `genx_temperature`, `genx_humidity`, etc. Use `--prometheus-metric` to set the base name (default: `genx`).

The same `examples/otlp/` stack supports pull mode:

```bash
cd examples/otlp
docker compose --profile pull up   # pull mode: Prometheus scrapes genx directly
```

## Multi-field payloads

Emit multiple named fields in a single data point via a YAML config file:

```yaml
# multi.yaml
duration: 1h
step: 1m
device: env-sensor

fields:
  temperature:
    type: cos
    min: 18
    max: 26
    period: 12h
  humidity:
    type: cos
    min: 40
    max: 80
    period: 8h
  pressure:
    type: linear
    first: 1010
    last: 1015
```

```bash
genx --config multi.yaml
{"device":"env-sensor","timestamp":1715000000,"fields":{"humidity":60.12,"pressure":1010.00,"temperature":22.43}}
```

## Custom payload template

Define the exact JSON shape using Go [`text/template`](https://pkg.go.dev/text/template) syntax:

```bash
genx --type cos --duration 1h --step 5m \
     --payload-template '{"sensor":"{{.Device}}","time":{{.Timestamp}},"celsius":{{.Value}}}'
```

Available placeholders: `{{.Device}}`, `{{.Timestamp}}`, `{{.TimestampISO}}`, `{{.Value}}`, `{{.Fields.name}}`.

## YAML config file

Any flag can be set in a YAML config file passed with `--config`. CLI flags always take precedence:

```yaml
type: cos
duration: 24h
step: 5m
device: room-sensor
realtime: true
noise: 0.03
seed: 42
output: nats
nats-url: nats://localhost:4222
nats-subject: home.temperature
```

```bash
genx --config config.yaml

# Read config from stdin (useful with Docker)
docker run -i ghcr.io/lucj/genx --config - < config.yaml
```

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | | Path to YAML config file (CLI flags take precedence) |
| `--generate-config` | | Print a sample YAML config file to stdout and exit |
| `--type` | `walk` | Curve type: `cos`, `linear`, `log`, `exp`, `walk`, `sawtooth`, `square` |
| `--duration` | `1d` | Total duration (e.g. `2d`, `6h`, `30m`) |
| `--step` | `1h` | Sampling interval (e.g. `5m`, `10s`) |
| `--device` | `device` | Device name (or prefix when `--devices > 1`) |
| `--devices` | `1` | Number of devices to simulate simultaneously |
| `--spread` | `0` | Per-device value spread as a ratio (e.g. `0.1` = ±10%) |
| `--noise` | `0` | Random noise per sample as a ratio (e.g. `0.05` = ±5%) |
| `--anomaly-rate` | `0` | Probability of injecting a spike or drop per point |
| `--anomaly-factor` | `3` | Anomaly magnitude: spike = value × factor, drop = value / factor |
| `--dropout-rate` | `0` | Probability of skipping a point entirely |
| `--realtime` | false | Emit one point per step using real wall-clock time |
| `--seed` | `0` | Fix the RNG seed for reproducible output (0 = random) |
| `--replay-file` | | Path to a JSON-lines file to replay through the sink |
| `--min` | `10` | Min value (cos, sawtooth, square) |
| `--max` | `25` | Max value (cos, sawtooth, square) |
| `--period` | `1d` | Period (cos, sawtooth, square) |
| `--duty-cycle` | `0.5` | Fraction of period in high state (square only) |
| `--first` | `0` | First value (linear) |
| `--last` | `1` | Last value (linear) |
| `--walk-start` | `20` | Starting value (walk) |
| `--walk-step` | `0.5` | Max delta magnitude per sample (walk) |
| `--walk-bias` | `0` | Per-step directional drift (walk); negative = downward |
| `--walk-min` | `15` | Lower clamp bound (walk); disabled when equal to `--walk-max` |
| `--walk-max` | `35` | Upper clamp bound (walk); disabled when equal to `--walk-min` |
| `--output` | `stdout` | Output sink: `stdout`, `webhook`, `nats`, `mqtt`, `kafka`, `file`, `otlp`, `prometheus` |
| `--format` | `json` | Serialisation format for stdout/file: `json`, `csv`, `influx` |
| `--iso-time` | false | Emit timestamp as ISO 8601 UTC string instead of Unix epoch |
| `--influx-measurement` | `genx` | InfluxDB measurement name (`--format influx`) |
| `--payload-template` | | Go `text/template` string for the JSON payload |
| `--payload-template-file` | | Path to a Go `text/template` file for the JSON payload |
| `--webhook-url` | | Webhook endpoint URL |
| `--webhook-token` | | Bearer token for the `Authorization` header |
| `--nats-url` | `nats://localhost:4222` | NATS server URL |
| `--nats-subject` | `genx` | NATS subject to publish to |
| `--nats-user` | | NATS username |
| `--nats-password` | | NATS password |
| `--nats-token` | | NATS authentication token |
| `--mqtt-broker` | `tcp://localhost:1883` | MQTT broker URL |
| `--mqtt-topic` | `genx` | MQTT topic to publish to |
| `--mqtt-qos` | `0` | MQTT QoS level (0, 1, or 2) |
| `--mqtt-client-id` | `genx-<pid>` | MQTT client ID |
| `--mqtt-user` | | MQTT username |
| `--mqtt-password` | | MQTT password |
| `--mqtt-ca-cert` | | CA certificate for verifying the broker's TLS certificate |
| `--mqtt-cert` | | Client certificate for mTLS authentication |
| `--mqtt-key` | | Client private key for mTLS authentication |
| `--mqtt-tls-insecure` | false | Skip broker TLS certificate verification (testing only) |
| `--kafka-brokers` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `--kafka-topic` | `genx` | Kafka topic to publish to |
| `--kafka-username` | | SASL/PLAIN username |
| `--kafka-password` | | SASL/PLAIN password |
| `--kafka-tls` | false | Enable TLS (uses system cert pool) |
| `--kafka-tls-insecure` | false | Skip broker TLS certificate verification (testing only) |
| `--file-path` | | Base path for file sink output (e.g. `data.jsonl`) |
| `--file-max-size` | | Rotate file when it reaches this size (e.g. `10MB`, `1GB`) |
| `--file-max-age` | | Rotate file after this duration (e.g. `1h`, `30m`) |
| `--otlp-endpoint` | `localhost:4317` | OTLP collector endpoint (`host:port`) |
| `--otlp-http` | false | Use OTLP/HTTP instead of OTLP/gRPC |
| `--otlp-header` | | Header to add to OTLP requests, repeatable (`key=value`) |
| `--otlp-insecure` | false | Disable TLS for the OTLP connection |
| `--otlp-metric` | `genx` | Base metric name; multi-field appends `.<fieldname>` |
| `--prometheus-port` | `9091` | Port to expose `/metrics` on (`--output prometheus`) |
| `--prometheus-metric` | `genx` | Base metric name; multi-field appends `_<fieldname>` |
