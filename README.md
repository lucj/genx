# genx

**genx** is a lightweight time-series data generator. It emits synthetic measurements following a mathematical curve — useful for testing dashboards, messaging pipelines, or IoT simulators.

Each data point is output as JSON:
```json
{"device":"sensor1","timestamp":1715000000,"value":24.53}
```

## Quick start

```
docker run ghcr.io/lucj/genx [flags]
```

Or build locally:
```
go build -o genx . && ./genx [flags]
```

Generate a cosine curve, one point every 15 minutes over 1 hour:
```
$ genx --type cos --min 18 --max 26 --duration 1h --step 15m
{"device":"device","timestamp":1715000000,"value":26.00}
{"device":"device","timestamp":1715054400,"value":22.00}
{"device":"device","timestamp":1715108800,"value":18.00}
{"device":"device","timestamp":1715163200,"value":22.00}
```

Use `--generate-config` to print a fully commented YAML template covering every option:
```
$ genx --generate-config > config.yaml
```

## Curve types

### Linear

Ramps from `--first` to `--last` over the total duration.

```
$ docker run ghcr.io/lucj/genx -type linear -duration 3d -first 10 -last 30 -step 6h
{"device":"device","timestamp":1715000000,"value":10.00}
{"device":"device","timestamp":1715021600,"value":11.67}
{"device":"device","timestamp":1715043200,"value":13.33}
...
```

### Random walk

Drifts by a random delta each sample — useful for simulating battery drain, temperature drift, or stock prices.

```
$ genx --type walk --walk-start 100 --walk-step 2 --walk-bias -0.1 \
       --walk-min 0 --walk-max 120 --duration 1h --step 1m
{"device":"device","timestamp":1715000000,"value":100.00}
{"device":"device","timestamp":1715000060,"value":98.73}
{"device":"device","timestamp":1715000120,"value":97.21}
...
```

`--walk-bias` adds a constant drift per step (negative = downward trend). `--walk-min` / `--walk-max` clamp the value; clamping is disabled when both are `0`.

### Cosine

Oscillates between `--min` and `--max` over the given `--period`.

```
$ docker run ghcr.io/lucj/genx -type cos -duration 2d -min 20 -max 30 -step 3h -period 1d
{"device":"device","timestamp":1715000000,"value":30.00}
{"device":"device","timestamp":1715010800,"value":27.50}
{"device":"device","timestamp":1715021600,"value":22.50}
{"device":"device","timestamp":1715032400,"value":20.00}
...
```

### Logarithmic

Produces a slow-growing logarithmic curve (natural log of elapsed seconds).

```
$ docker run ghcr.io/lucj/genx -type log -duration 1d -step 2h
{"device":"device","timestamp":1715000000,"value":0.00}
{"device":"device","timestamp":1715007200,"value":8.88}
{"device":"device","timestamp":1715014400,"value":9.57}
...
```

### Exponential

Grows exponentially over the duration.

```
$ docker run ghcr.io/lucj/genx -type exp -duration 6h -step 30m
{"device":"device","timestamp":1715000000,"value":1.00}
{"device":"device","timestamp":1715001800,"value":1.07}
{"device":"device","timestamp":1715003600,"value":1.15}
...
```

## Fleet mode

Simulate multiple devices at once with `--devices`. Each device gets an independent value curve; `--spread` adds a per-device random offset so they don't all emit identical values.

```
$ genx --type cos --devices 3 --spread 0.1 --duration 1h --step 5m --realtime
{"device":"device-0","timestamp":1715000000,"value":24.10}
{"device":"device-1","timestamp":1715000000,"value":23.57}
{"device":"device-2","timestamp":1715000000,"value":25.02}
...
```

`--spread 0.1` means each device's values are randomly scaled by ±10%. Use `--device` to set the prefix (`--device sensor` → `sensor-0`, `sensor-1`, …).

## Noise

Add realistic random jitter to every sample with `--noise`:

```
$ genx --type cos --noise 0.05 --duration 1h --step 1m
```

`--noise 0.05` multiplies each value by a random factor in `[0.95, 1.05]`. Works in both single-device and fleet mode.

## Realtime mode

By default genx generates the full dataset instantly (batch mode). Add `--realtime` to emit one point per `--step` interval using the actual wall clock — handy for live pipeline testing.

```
$ docker run ghcr.io/lucj/genx -type cos -min 18 -max 26 -duration 1h -step 10s --realtime
```

## Reproducible runs

Use `--seed` to fix the random number generator so every run with the same flags produces identical output. Useful for CI fixtures and sharing reproducible scenarios.

```
$ genx --type cos --noise 0.05 --devices 3 --spread 0.1 --seed 42 --duration 1h --step 5m
```

## Replay mode

Replay a previously recorded JSON-lines file through any configured sink. Batch mode sends all points immediately; realtime mode waits `--step` between sends and stamps the current time.

```
# Record
$ genx --type cos --duration 1h --step 1m > recording.jsonl

# Replay to NATS in realtime
$ genx --replay-file recording.jsonl --output nats --nats-url nats://localhost:4222 \
       --realtime --step 1m
```

## Output sinks

By default data is written to stdout. Use `--output` to route it elsewhere.

### Webhook

POSTs each data point as JSON to an HTTP endpoint. Optionally attach a bearer token:

```
$ genx --type cos --duration 1h --step 5m \
       --output webhook --webhook-url http://myserver/ingest \
       --webhook-token mysecrettoken
```

### NATS

Publishes to a NATS subject. Supports username/password and token authentication:

```
# No auth
$ genx --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
       --type cos --duration 1h --step 1m

# Username / password
$ genx --output nats --nats-url nats://localhost:4222 \
       --nats-user alice --nats-password secret \
       --type cos --duration 1h --step 1m --realtime

# Token
$ genx --output nats --nats-url nats://localhost:4222 \
       --nats-token mysecrettoken \
       --type cos --duration 1h --step 1m --realtime
```

A `docker-compose.yml` is included for local testing — it covers NATS (no auth, user/password, token), MQTT (anonymous, user/password, TLS, mTLS), and a webhook echo server. See the comments inside for usage instructions.

### MQTT

Publishes to an MQTT topic. Supports username/password and TLS/mTLS authentication:

```
# Username / password
$ genx --type cos --duration 1h --step 5m \
       --output mqtt --mqtt-broker tcp://localhost:1883 --mqtt-topic home/temperature \
       --mqtt-user myuser --mqtt-password mypassword

# TLS with custom CA (verify broker certificate)
$ genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-topic sensors \
       --mqtt-ca-cert ca.crt --type cos --duration 1h --step 1m

# Mutual TLS (single shared client certificate)
$ genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-topic sensors \
       --mqtt-ca-cert ca.crt --mqtt-cert client.crt --mqtt-key client.key \
       --type cos --duration 1h --step 1m

# Mutual TLS with per-device certificates (YAML config only)
# Each device gets its own connection using its own cert/key pair.
# --mqtt-ca-cert still applies to all connections.
$ genx --config fleet.yaml
```

Example `fleet.yaml`:
```yaml
output: mqtt
mqtt-broker: ssl://localhost:8883
mqtt-topic: sensors
mqtt-ca-cert: ca.crt

type: cos
devices: 3
device: sensor
duration: 1h
step: 1m
realtime: true

mqtt-device-certs:
  sensor-0:
    cert: certs/sensor-0.crt
    key:  certs/sensor-0.key
  sensor-1:
    cert: certs/sensor-1.crt
    key:  certs/sensor-1.key
  sensor-2:
    cert: certs/sensor-2.crt
    key:  certs/sensor-2.key
```

```
# Skip broker certificate verification (testing only)
$ genx --output mqtt --mqtt-broker ssl://localhost:8883 --mqtt-topic sensors \
       --mqtt-tls-insecure --type cos --duration 1h --step 1m
```

## Multi-field payloads

Emit multiple named fields in a single data point — for example temperature, humidity, and pressure from the same device. This mode is available via a YAML config file (see below).

Example config (`multi.yaml`):
```yaml
duration: 1h
step: 1m
realtime: true
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

```
$ genx --config multi.yaml
{"device":"env-sensor","timestamp":1715000000,"fields":{"humidity":60.12,"pressure":1010.00,"temperature":22.43}}
```

## Custom payload template

Define the exact JSON shape using Go [`text/template`](https://pkg.go.dev/text/template) syntax. Useful when the consuming system expects a specific schema.

Available placeholders:

| Placeholder | Description |
|-------------|-------------|
| `{{.Device}}` | Device name |
| `{{.Timestamp}}` | Unix timestamp (int64) |
| `{{.Value}}` | Numeric value (0 if multi-field mode) |
| `{{.Fields.name}}` | Named field value (multi-field mode) |

**Inline template:**

```
$ genx --type cos --duration 1h --step 5m \
       --payload-template '{"sensor":"{{.Device}}","time":{{.Timestamp}},"celsius":{{.Value}}}'
{"sensor":"device","time":1715000000,"celsius":24.53}
```

**Template file:**

```
# template.json
{
  "sensor_id": "{{.Device}}",
  "recorded_at": {{.Timestamp}},
  "measurements": {
    "temperature": {{.Fields.temperature}},
    "humidity":    {{.Fields.humidity}}
  }
}
```

```
$ genx --config multi.yaml --payload-template-file template.json
```

## YAML config file

Any flag can be set in a YAML config file passed with `--config`. CLI flags always take precedence over config values.

```yaml
type: cos
duration: 24h
step: 5m
device: room-sensor
realtime: true
noise: 0.03
seed: 42

min: 18
max: 26
period: 12h

output: nats
nats-url: nats://localhost:4222
nats-subject: home.temperature
nats-user: alice
nats-password: secret
```

```
$ genx --config config.yaml
```

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | | Path to YAML config file (CLI flags take precedence) |
| `--generate-config` | | Print a sample YAML config file to stdout and exit |
| `--type` | `cos` | Curve type: `cos`, `linear`, `log`, `exp`, `walk` |
| `--duration` | `1d` | Total duration (e.g. `2d`, `6h`, `30m`) |
| `--step` | `1h` | Sampling interval (e.g. `5m`, `10s`) |
| `--device` | `device` | Device name (or prefix when `--devices > 1`) |
| `--devices` | `1` | Number of devices to simulate simultaneously |
| `--spread` | `0` | Per-device value spread as a ratio (e.g. `0.1` = ±10%) |
| `--noise` | `0` | Random noise per sample as a ratio (e.g. `0.05` = ±5%) |
| `--realtime` | false | Emit one point per step using real wall-clock time |
| `--seed` | `0` | Fix the RNG seed for reproducible output (0 = random) |
| `--min` | `10` | Min value (cos) |
| `--max` | `25` | Max value (cos) |
| `--period` | `1d` | Period (cos) |
| `--first` | `0` | First value (linear) |
| `--last` | `1` | Last value (linear) |
| `--walk-start` | `100` | Starting value (walk) |
| `--walk-step` | `1` | Max delta magnitude per sample (walk) |
| `--walk-bias` | `0` | Per-step directional drift (walk); negative = downward |
| `--walk-min` | `0` | Lower clamp bound (walk); disabled when equal to `--walk-max` |
| `--walk-max` | `0` | Upper clamp bound (walk); disabled when equal to `--walk-min` |
| `--output` | `stdout` | Output sink: `stdout`, `webhook`, `nats`, `mqtt` |
| `--replay-file` | | Path to a JSON-lines file to replay through the sink |
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
| `--mqtt-ca-cert` | | Path to CA certificate file for verifying the broker's TLS certificate |
| `--mqtt-cert` | | Path to client certificate file for mTLS authentication |
| `--mqtt-key` | | Path to client private key file for mTLS authentication |
| `--mqtt-tls-insecure` | false | Skip broker TLS certificate verification (testing only) |
| `mqtt-device-certs` (YAML only) | | Map of device name → `{cert, key}` for per-device mTLS |
| `--payload-template` | | Go `text/template` string for the JSON payload |
| `--payload-template-file` | | Path to a Go `text/template` file for the JSON payload |
