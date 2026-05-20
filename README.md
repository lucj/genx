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
git clone https://github.com/lucj/genx.git && cd genx && go build -o genx .
```

## Quick start

```bash
genx --type cos --min 18 --max 26 --duration 1h --step 15m
{"device":"device","timestamp":1715000000,"value":26.00}
{"device":"device","timestamp":1715054400,"value":22.00}
...
```

Use `--generate-config` to print a fully commented YAML template covering every option:
```bash
genx --generate-config > config.yaml
```

## Curve types

| Type | Description | Key flags |
|------|-------------|-----------|
| `walk` (default) | Random drift each sample | `--walk-start`, `--walk-step`, `--walk-bias`, `--walk-min`, `--walk-max` |
| `cos` | Sinusoidal oscillation | `--min`, `--max`, `--period` |
| `linear` | Steady ramp | `--first`, `--last` |
| `sawtooth` | Ramp then reset (/|/|) | `--min`, `--max`, `--period` |
| `square` | Binary high/low | `--min`, `--max`, `--period`, `--duty-cycle` |
| `log` | Logarithmic growth | — |
| `exp` | Exponential growth | — |
| `geo` | GPS track (lat/lon walk) | `--geo-lat`, `--geo-lon`, `--geo-speed`, `--geo-bearing`, `--geo-drift` |

```bash
# cosine between 20–30 °C, 24 h period
genx --type cos --min 20 --max 30 --period 1d --duration 2d --step 3h

# random walk, slow downward drift, clamped 0–120
genx --type walk --walk-start 100 --walk-step 2 --walk-bias -0.1 \
     --walk-min 0 --walk-max 120 --duration 1h --step 1m
```

`--walk-bias` adds a constant drift per step. `--walk-min` / `--walk-max` clamp the value; clamping is disabled when both are `0`.

## Geospatial simulation

`--type geo` simulates a moving device. Each step advances the position by `speed × step` metres in the current bearing, then randomly adjusts the bearing by up to ±`drift` degrees. Outputs `lat` and `lon` as named fields.

```bash
# Single vehicle starting in Paris, 10 m/s, heading north
genx --type geo --duration 1h --step 30s --realtime
{"device":"device","timestamp":1715000000,"fields":{"lat":48.8620,"lon":2.3522}}
{"device":"device","timestamp":1715000030,"fields":{"lat":48.8674,"lon":2.3519}}
...

# Fleet of 3 trucks, moderate bearing drift
genx --type geo --devices 3 --geo-speed 25 --geo-drift 20 --duration 2h --step 10s --realtime
```

Works with all output sinks and formats unchanged.

## Fleet mode

Simulate multiple devices with `--devices`. `--spread` adds a per-device random offset so they emit distinct values.

```bash
genx --type cos --devices 3 --spread 0.1 --duration 1h --step 5m --realtime
{"device":"device-0","timestamp":1715000000,"value":24.10}
{"device":"device-1","timestamp":1715000000,"value":23.57}
{"device":"device-2","timestamp":1715000000,"value":25.02}
```

Use `--device` to set the name prefix (`--device sensor` → `sensor-0`, `sensor-1`, …). Use `--device-names` for explicit names:

```bash
genx --type cos --device-names "paris,london,berlin" --duration 1h --step 5m
{"device":"paris","timestamp":1715000000,"value":24.10}
{"device":"london","timestamp":1715000000,"value":23.57}
{"device":"berlin","timestamp":1715000000,"value":25.02}
```

## Noise & anomalies

```bash
genx --type cos --duration 1h --step 1m \
     --noise 0.05 --anomaly-rate 0.02 --anomaly-factor 5 --dropout-rate 0.03
```

- `--noise 0.05` — multiply each value by a random factor in `[0.95, 1.05]`
- `--anomaly-rate 0.02` — ~2% of points become spikes or drops (magnitude set by `--anomaly-factor`)
- `--dropout-rate 0.03` — ~3% of points are silently skipped

## Run summary

After every run, genx prints a one-line summary to stderr:

```
sent 360 points in 30.1s (11.9 pts/s, 0 errors)
```

This lets you verify throughput and catch send errors without digging through logs — particularly useful when load-testing a sink.

## Historical data generation

Use `--from` to anchor the dataset to a specific point in time — useful for generating historical data that aligns with real calendar dates:

```bash
# One week of hourly data starting 2024-01-01
genx --type cos --from 2024-01-01T00:00:00Z --duration 7d --step 1h

# Last 24 hours of data (Unix epoch)
genx --type walk --from $(date -d '24 hours ago' +%s) --duration 24h --step 5m
```

Accepted formats: ISO 8601 with timezone (`2024-01-01T00:00:00Z`), date only (`2024-01-01`, treated as midnight UTC), or a Unix epoch integer. Defaults to now.

`--from` applies to batch mode. In realtime mode timestamps follow the wall clock.

## Realtime mode & reproducibility

By default genx generates the full dataset instantly. Add `--realtime` to pace output to one point per `--step` interval. Use `--seed` to fix the RNG for reproducible runs.

```bash
genx --type cos --step 10s --realtime
genx --type cos --noise 0.05 --devices 3 --seed 42 --duration 1h --step 5m
```

## Replay mode

Replay a previously recorded JSON-lines file through any configured sink:

```bash
genx --type cos --duration 1h --step 1m > recording.jsonl
genx --replay-file recording.jsonl --output nats --nats-url nats://localhost:4222 --realtime --step 1m
```

## Output formats

Use `--format` to control serialisation for the `stdout` and `file` sinks (and the `webhook` sink for CloudEvents).

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
```

Field columns are sorted alphabetically in multi-field mode. Combine with `--iso-time` for human-readable timestamps.

### CloudEvents

Wraps each data point in a [CloudEvents 1.0](https://cloudevents.io/) structured JSON envelope. Works with any CloudEvents-compatible consumer (Azure Event Grid, AWS EventBridge via HTTP, GCP Eventarc, Knative, Dapr, …) — no vendor SDK required.

```bash
genx --type cos --devices 3 --step 5s --realtime --format cloudevent
```
```json
{
  "specversion": "1.0",
  "id": "304a2996-2fa4-4bf9-885b-0acb7c87dd32",
  "source": "/genx/device-0",
  "type": "io.genx.measurement",
  "time": "2024-05-15T12:00:00Z",
  "datacontenttype": "application/json",
  "data": {"device":"device-0","timestamp":1715000000,"value":24.53}
}
```

Pipe into a CloudEvents HTTP endpoint via the webhook sink — the `Content-Type` is set automatically to `application/cloudevents+json`:

```bash
genx --type cos --step 5s --realtime --format cloudevent \
     --output webhook --webhook-url http://my-eventing-broker/default
```

Use `--cloudevent-source` and `--cloudevent-type` to customise the envelope:

```bash
genx --format cloudevent \
     --cloudevent-source /plant/line-a \
     --cloudevent-type com.acme.sensor.temperature \
     --type cos --step 1m
```

### InfluxDB line protocol

```bash
genx --type cos --duration 1h --step 10m --format influx
genx,device=device value=26 1715000000000000000
```

Timestamps are in nanoseconds as required by the protocol. Pipe directly into `influx write`:

```bash
genx --type cos --devices 3 --step 10s --realtime --format influx \
  | influx write --bucket my-bucket --org my-org
```

Use `--influx-measurement` to override the measurement name (default: `genx`).

## Output sinks

### Webhook

```bash
genx --output webhook --webhook-url http://myserver/ingest --webhook-token mysecrettoken \
     --type cos --duration 1h --step 5m
```

### NATS

```bash
genx --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
     --type cos --duration 1h --step 1m
```

### MQTT

```bash
# Plain
genx --output mqtt --mqtt-broker tcp://localhost:1883 --mqtt-topic home/temp \
     --type cos --duration 1h --step 5m

# TLS
genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-ca-cert ca.crt \
     --type cos --duration 1h --step 1m

# mTLS
genx --output mqtt --mqtt-broker ssl://localhost:8883 \
     --mqtt-ca-cert ca.crt --mqtt-cert client.crt --mqtt-key client.key \
     --type cos --duration 1h --step 1m
```

Per-device certificates are supported via a YAML config file — see `genx --generate-config` for `mqtt-device-certs`.

### Kafka

```bash
genx --output kafka --kafka-brokers localhost:9092 --kafka-topic sensors \
     --type cos --duration 1h --step 5m

# SASL/PLAIN + TLS
genx --output kafka --kafka-brokers localhost:9092 --kafka-topic sensors \
     --kafka-username alice --kafka-password secret --kafka-tls \
     --type cos --step 5s --realtime
```

### File (with rotation)

```bash
# Rotate every 10 MB or every hour, whichever comes first
genx --type cos --duration 24h --step 1m --realtime \
     --file-path data.jsonl --file-max-size 10MB --file-max-age 1h
```

Rotated files are named with a UTC timestamp suffix: `data.20240506T120000.jsonl`.

### InfluxDB

Writes directly to an InfluxDB v2 instance using the line-protocol write API. No need to pipe through `influx write`.

```bash
genx --output influxdb \
     --influxdb-url http://localhost:8086 \
     --influxdb-token my-token \
     --influxdb-org my-org \
     --influxdb-bucket sensors \
     --type cos --devices 3 --step 5s --realtime
```

Use `--influx-measurement` to set the measurement name (default: `genx`). Specifying `--influxdb-url` also implies `--output influxdb`.

### OpenTelemetry (OTLP)

Pushes metrics as OTLP gauges. Each device name becomes a `device` attribute; multi-field payloads produce one gauge per field.

```bash
# gRPC (default)
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type cos --devices 3 --step 5s --realtime

# HTTP/protobuf
genx --output otlp --otlp-endpoint localhost:4318 --otlp-http --otlp-insecure \
     --type cos --devices 3 --step 5s --realtime

# Authenticated (Grafana Cloud, Honeycomb, …)
genx --output otlp --otlp-endpoint otlp.example.com:4317 \
     --otlp-header "x-api-key=your-key" --type cos --step 10s --realtime
```

A ready-made Docker Compose stack (OTel Collector + Prometheus + Grafana) is in `examples/otlp/`:

```bash
cd examples/otlp
docker compose --profile demo up   # genx → OTel Collector → Prometheus → Grafana
```

Open <http://localhost:3000> to see the live dashboard.

### Prometheus (pull / scrape)

Starts an HTTP server and exposes the latest values at `/metrics` in Prometheus text format.

```bash
genx --output prometheus --prometheus-port 9091 \
     --type cos --devices 3 --step 5s --realtime
# Prometheus scrapes http://localhost:9091/metrics
```

Multi-field payloads produce one metric per field: `genx_temperature`, `genx_humidity`, etc. The same `examples/otlp/` stack supports pull mode:

```bash
cd examples/otlp && docker compose --profile pull up
```

## Scenario scripting

A scenario is a sequence of phases executed in order. Each phase inherits global defaults and overrides only what it sets. This lets you simulate realistic device behaviour: normal operation, fault, recovery.

```yaml
# scenario.yaml
device: env-sensor
step: 30s

scenario:
  - duration: 10m
    type: cos
    min: 20
    max: 25
  - duration: 5m      # sensor fault — all points dropped
    dropout-rate: 1.0
  - duration: 10m     # recovery
    type: cos
    min: 20
    max: 25
```

```bash
genx --config scenario.yaml
```

Timestamps are continuous across phases. Geo walkers preserve position between phases. Scenario mode is incompatible with `--replay-file` and top-level `fields`.

## Count mode

Use `--count` to emit exactly N points instead of a time-based duration:

```bash
genx --type cos --count 100 --step 5m
genx --config scenario.yaml --count 500
```

`--count` and `--duration` are mutually exclusive.

## Verbose output

`--verbose` prints one `[OK]` or `[KO]` line per point to stderr alongside the normal sink output. Useful for diagnosing delivery failures when testing a remote sink:

```bash
genx --type cos --step 5s --realtime --output webhook --webhook-url http://localhost:8080 --verbose
[OK] {"device":"device","timestamp":1715000005,"value":24.81}
[KO] {"device":"device","timestamp":1715000010,"value":24.12}  Post "http://localhost:8080": connection refused
```

## Validate config

`genx validate` loads a config file and prints a dry-run summary without connecting to any sink or emitting data:

```bash
genx validate --config config.yaml
✓ Config: config.yaml
✓ Device: env-sensor
✓ Output: stdout (format: json)
✓ Mode: scenario (3 phases)
  Phase 1: cos, 10m, step 30s → 20 pts/device
  Phase 2: dropout (no points)
  Phase 3: cos, 10m, step 30s → 20 pts/device
✓ Total: ~40 points
✓ All checks passed
```

### HTTP pull server (`--output http-server`)

Starts a local HTTP server. Any client can poll `GET /` to retrieve the most recent point(s) as a JSON array — useful for pull-based systems, quick `curl` inspection, or dashboards that poll a REST endpoint.

```bash
genx --type cos --step 5s --realtime --output http-server --http-port 8888 --http-buffer 10
curl http://localhost:8888/
[{"device":"device","timestamp":1715000005,"value":24.81}]
```

Use `--http-buffer` to control how many recent points are kept in memory. Clients can request fewer with `?n=`:

```bash
curl "http://localhost:8888/?n=3"
[
  {"device":"device","timestamp":1715000000,"value":24.10},
  {"device":"device","timestamp":1715000005,"value":24.53},
  {"device":"device","timestamp":1715000010,"value":24.81}
]
```

The response is always a JSON array, even when only one point is available. The endpoint always returns JSON regardless of `--format`.

The server stays alive until Ctrl-C. `--realtime` is enabled automatically so the buffer updates on every step. Pass `--realtime=false` explicitly to pre-generate all points instantly and serve them from the buffer until the process is stopped.

## Multi-field payloads

```yaml
# multi.yaml
duration: 1h
step: 1m
device: env-sensor
fields:
  temperature: { type: cos, min: 18, max: 26, period: 12h }
  humidity:    { type: cos, min: 40, max: 80, period: 8h }
  pressure:    { type: linear, first: 1010, last: 1015 }
```

```bash
genx --config multi.yaml
{"device":"env-sensor","timestamp":1715000000,"fields":{"humidity":60.12,"pressure":1010.00,"temperature":22.43}}
```

## Custom payload template

```bash
genx --type cos --duration 1h --step 5m \
     --payload-template '{"sensor":"{{.Device}}","time":{{.Timestamp}},"celsius":{{.Value}}}'
```

Available placeholders: `{{.Device}}`, `{{.Timestamp}}`, `{{.TimestampISO}}`, `{{.Value}}`, `{{.Fields.name}}`.

## YAML config file

Any flag can be set in a YAML file passed with `--config`. CLI flags always take precedence.

```yaml
type: cos
duration: 24h
step: 5m
realtime: true
noise: 0.03
output: nats
nats-url: nats://localhost:4222
```

```bash
genx --config config.yaml
docker run -i ghcr.io/lucj/genx --config - < config.yaml
```

## All flags

### General

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | | Path to YAML config file (CLI flags take precedence) |
| `--generate-config` | | Print a sample YAML config file to stdout and exit |
| `--type` | `walk` | Curve type: `cos`, `linear`, `log`, `exp`, `walk`, `sawtooth`, `square` |
| `--duration` | `1m` | Total duration (e.g. `2d`, `6h`, `30m`) |
| `--from` | now | Start timestamp: ISO 8601 (`2024-01-01T00:00:00Z`), date (`2024-01-01`), or Unix epoch |
| `--step` | `5s` | Sampling interval (e.g. `5m`, `10s`) |
| `--device` | `device` | Device name (or prefix when `--devices > 1`) |
| `--devices` | `1` | Number of devices to simulate simultaneously |
| `--device-names` | | Explicit device names, comma-separated (overrides `--device` and `--devices`) |
| `--realtime` | true | Emit one point per step using real wall-clock time |
| `--seed` | `0` | Fix the RNG seed for reproducible output (0 = random) |
| `--replay-file` | | Path to a JSON-lines file to replay through the sink |
| `--rate` | `0` | Max points per second across all devices (0 = unlimited) |
| `--noise` | `0` | Random noise per sample as a ratio (e.g. `0.05` = ±5%) |
| `--spread` | `0` | Per-device value spread as a ratio (e.g. `0.1` = ±10%) |
| `--anomaly-rate` | `0` | Probability of injecting a spike or drop per point |
| `--anomaly-factor` | `3` | Anomaly magnitude: spike = value × factor, drop = value / factor |
| `--dropout-rate` | `0` | Probability of skipping a point entirely |
| `--count` | | Emit exactly N points instead of using `--duration` (mutually exclusive) |

### Periodic curves (`--type cos` / `sawtooth` / `square`)

| Flag | Default | Description |
|------|---------|-------------|
| `--min` | `10` | Minimum value |
| `--max` | `25` | Maximum value |
| `--period` | `1d` | Period (e.g. `1d`, `12h`) |
| `--duty-cycle` | `0.5` | Fraction of period in high state (square only) |

### Linear curve (`--type linear`)

| Flag | Default | Description |
|------|---------|-------------|
| `--first` | `0` | Starting value |
| `--last` | `1` | Ending value |

### Random walk (`--type walk`)

| Flag | Default | Description |
|------|---------|-------------|
| `--walk-start` | `20` | Starting value |
| `--walk-step` | `0.5` | Max delta magnitude per sample |
| `--walk-bias` | `0` | Per-step directional drift (negative = downward) |
| `--walk-min` | `15` | Lower clamp; disabled when equal to `--walk-max` |
| `--walk-max` | `35` | Upper clamp; disabled when equal to `--walk-min` |

### Geo (`--type geo`)

| Flag | Default | Description |
|------|---------|-------------|
| `--geo-lat` | `48.8566` | Starting latitude |
| `--geo-lon` | `2.3522` | Starting longitude |
| `--geo-speed` | `10` | Speed in m/s |
| `--geo-bearing` | `0` | Initial bearing in degrees (0=N, 90=E, 180=S, 270=W) |
| `--geo-drift` | `15` | Max random bearing change per step in degrees |

### Output

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `stdout` | Sink: `stdout`, `webhook`, `nats`, `mqtt`, `kafka`, `file`, `otlp`, `prometheus`, `influxdb` |
| `--format` | `json` | Format for stdout/file: `json`, `csv`, `influx`, `cloudevent` |
| `--verbose` | false | Print `[OK]`/`[KO] <payload>` to stderr for every point sent |
| `--iso-time` | false | Emit timestamp as ISO 8601 UTC string instead of Unix epoch |
| `--influx-measurement` | `genx` | InfluxDB measurement name (`--format influx`) |
| `--cloudevent-source` | `/genx` | CloudEvents source URI; device name is appended automatically |
| `--cloudevent-type` | `io.genx.measurement` | CloudEvents `type` field |
| `--payload-template` | | Go `text/template` string for the JSON payload |
| `--payload-template-file` | | Path to a Go `text/template` file for the JSON payload |

### Webhook (`--output webhook`)

| Flag | Default | Description |
|------|---------|-------------|
| `--webhook-url` | | Webhook endpoint URL |
| `--webhook-token` | | Bearer token for the `Authorization` header |

### NATS (`--output nats`)

| Flag | Default | Description |
|------|---------|-------------|
| `--nats-url` | `nats://localhost:4222` | NATS server URL |
| `--nats-subject` | `genx` | Subject to publish to |
| `--nats-user` | | Username |
| `--nats-password` | | Password |
| `--nats-token` | | Authentication token |

### MQTT (`--output mqtt`)

| Flag | Default | Description |
|------|---------|-------------|
| `--mqtt-broker` | `tcp://localhost:1883` | Broker URL |
| `--mqtt-topic` | `genx` | Topic to publish to |
| `--mqtt-qos` | `0` | QoS level (0, 1, or 2) |
| `--mqtt-client-id` | `genx-<pid>` | Client ID |
| `--mqtt-user` | | Username |
| `--mqtt-password` | | Password |
| `--mqtt-ca-cert` | | CA certificate for verifying the broker's TLS certificate |
| `--mqtt-cert` | | Client certificate for mTLS |
| `--mqtt-key` | | Client private key for mTLS |
| `--mqtt-tls-insecure` | false | Skip broker TLS certificate verification (testing only) |

### Kafka (`--output kafka`)

| Flag | Default | Description |
|------|---------|-------------|
| `--kafka-brokers` | `localhost:9092` | Comma-separated broker addresses |
| `--kafka-topic` | `genx` | Topic to publish to |
| `--kafka-username` | | SASL/PLAIN username |
| `--kafka-password` | | SASL/PLAIN password |
| `--kafka-tls` | false | Enable TLS (uses system cert pool) |
| `--kafka-tls-insecure` | false | Skip broker TLS certificate verification (testing only) |

### File (`--output file`)

| Flag | Default | Description |
|------|---------|-------------|
| `--file-path` | | Base path for file output (e.g. `data.jsonl`) |
| `--file-max-size` | | Rotate when file reaches this size (e.g. `10MB`) |
| `--file-max-age` | | Rotate after this duration (e.g. `1h`) |

### InfluxDB (`--output influxdb`)

| Flag | Default | Description |
|------|---------|-------------|
| `--influxdb-url` | `http://localhost:8086` | InfluxDB server URL |
| `--influxdb-token` | | API token |
| `--influxdb-org` | | Organisation name |
| `--influxdb-bucket` | `genx` | Bucket name |
| `--influx-measurement` | `genx` | Measurement name |

### OTLP (`--output otlp`)

| Flag | Default | Description |
|------|---------|-------------|
| `--otlp-endpoint` | `localhost:4317` | Collector endpoint (`host:port`) |
| `--otlp-http` | false | Use OTLP/HTTP instead of OTLP/gRPC |
| `--otlp-header` | | Header to add to requests, repeatable (`key=value`) |
| `--otlp-insecure` | false | Disable TLS for the OTLP connection |
| `--otlp-metric` | `genx` | Base metric name; multi-field appends `.<fieldname>` |

### Prometheus pull (`--output prometheus`)

| Flag | Default | Description |
|------|---------|-------------|
| `--prometheus-port` | `9091` | Port to expose `/metrics` on |
| `--prometheus-metric` | `genx` | Base metric name; multi-field appends `_<fieldname>` |

### HTTP pull server (`--output http-server`)

| Flag | Default | Description |
|------|---------|-------------|
| `--http-port` | `8888` | Port the server listens on |
| `--http-buffer` | `1` | Number of recent points kept in memory |
