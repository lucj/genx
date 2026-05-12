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

## Curve types

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

![Cosinus graph](/images/cos.png)

### Linear

Ramps from `--first` to `--last` over the total duration.

```
$ docker run ghcr.io/lucj/genx -type linear -duration 3d -first 10 -last 30 -step 6h
{"device":"device","timestamp":1715000000,"value":10.00}
{"device":"device","timestamp":1715021600,"value":11.67}
{"device":"device","timestamp":1715043200,"value":13.33}
...
```

![Linear graph](/images/linear.png)

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

## Output sinks

By default data is written to stdout. Use `--output` to route it elsewhere.

### Webhook

POSTs each data point as JSON to an HTTP endpoint.

```
$ docker run ghcr.io/lucj/genx -type cos -duration 1h -step 5m \
    -output webhook -webhook-url http://myserver/ingest
```

### NATS

Publishes to a NATS subject.

```
$ docker run ghcr.io/lucj/genx -type linear -duration 1h -step 1m \
    -output nats -nats-url nats://<NATS_HOST>:4222 -nats-subject sensors.temp
```

**End-to-end example with a containerised NATS server:**

```
# 1. shared network
docker network create genx-net

# 2. NATS server
docker run -d --name nats --network genx-net nats

# 3. subscriber (another terminal)
docker run --rm --network genx-net natsio/nats-box \
    nats sub -s nats://nats:4222 sensors.temp

# 4. generate and publish
docker run --network genx-net ghcr.io/lucj/genx \
    -type linear -duration 1h -step 1m \
    -output nats -nats-url nats://nats:4222 -nats-subject sensors.temp
```

### MQTT

Publishes to an MQTT topic.

```
$ docker run ghcr.io/lucj/genx -type cos -duration 1h -step 5m \
    -output mqtt -mqtt-broker tcp://<MQTT_HOST>:1883 -mqtt-topic home/temperature
```

## Realtime mode

By default genx generates the full dataset instantly (batch mode). Add `--realtime` to emit one point per `--step` interval using the actual wall clock — handy for live pipeline testing.

```
$ docker run ghcr.io/lucj/genx -type cos -min 18 -max 26 -duration 1h -step 10s --realtime
```

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `-type` | `cos` | Curve type: `cos`, `linear`, `log`, `exp` |
| `-duration` | `1d` | Total duration (e.g. `2d`, `6h`, `30m`) |
| `-step` | `1h` | Sampling interval (e.g. `5m`, `10s`) |
| `-device` | `device` | Device/sensor name in each data point |
| `-realtime` | false | Emit one point per step using real time |
| `-min` | `10` | Min value (cos) |
| `-max` | `25` | Max value (cos) |
| `-period` | `1d` | Period (cos) |
| `-first` | `0` | First value (linear) |
| `-last` | `1` | Last value (linear) |
| `-output` | `stdout` | Output sink: `stdout`, `webhook`, `nats`, `mqtt` |
| `-webhook-url` | | Webhook endpoint URL |
| `-nats-url` | `nats://<NATS_HOST>:4222` | NATS server URL |
| `-nats-subject` | `genx` | NATS subject |
| `-mqtt-broker` | `tcp://<MQTT_HOST>:1883` | MQTT broker URL |
| `-mqtt-topic` | `genx` | MQTT topic |
| `-mqtt-qos` | `0` | MQTT QoS level (0, 1, 2) |
| `-mqtt-client-id` | `genx-<pid>` | MQTT client ID |
