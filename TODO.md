# TODO

Feature backlog in suggested implementation order.

---

## P1 — Scenario scripting (state machine)

**Why:** Turns genx from a data generator into a proper test harness. The only tool that lets you script predictable device lifecycles for alerting, dashboard, and pipeline validation tests.

**Idea:** Add a `scenario` key to the YAML config — a list of phases executed in sequence. Each phase inherits global defaults and overrides whatever it needs.

```yaml
# config.yaml
device: sensor-1
step: 10s
realtime: true
output: mqtt
mqtt-broker: tcp://localhost:1883

scenario:
  - duration: 10m
    type: cos
    min: 20
    max: 25
  - duration: 5m     # overheat ramp
    type: cos
    min: 30
    max: 50
  - duration: 2m     # connectivity loss
    dropout-rate: 1.0
  - duration: 10m    # recovery
    type: cos
    min: 20
    max: 25
```

**Notes:**
- Only available via config file (no CLI-only representation of a list of phases).
- `--type`, `--min`, `--max`, `--noise`, `--anomaly-rate`, `--dropout-rate`, `--step` should all be overridable per phase.
- Incompatible with `--replay-file` and the top-level `fields` map (at least initially).
- Fleet mode (`--devices`) should still work — all devices follow the same scenario.

---

## P2 — Redis Streams sink (`--output redis`)

**Why:** Redis Streams (`XADD`) is the de-facto lightweight message bus in IoT edge stacks and is trivially added with `go-redis/v9`. Removes a common reason to reach for a heavier broker just to test a pipeline.

**Flags:**
```
--redis-url        Redis URL (default: redis://localhost:6379)
--redis-stream     Stream key to publish to (default: genx)
--redis-password   Password
--redis-tls        Enable TLS
```

**Notes:**
- Each data point becomes an XADD entry; field map keys match the JSON keys (`device`, `timestamp`, `value` / named fields).
- Output inference: `--redis-url` set → `--output redis` implied.

---

## P3 — Rate cap (`--rate`)

**Why:** Decouples throughput from the shape of the curve. Currently `--step 1ms --realtime` is the only way to push fast; `--rate` lets you load-test a sink at a precise ingestion rate without distorting the time axis.

**Flag:**
```
--rate   maximum points per second across all devices (0 = unlimited)
```

**Example:**
```bash
# Push 500 points/s to Kafka regardless of --step
genx --type cos --step 1s --devices 10 --rate 500 --output kafka ...
```

**Notes:**
- Implemented as a `time.Ticker`-based token bucket or a simple `time.Sleep` throttle.
- In realtime mode `--rate` acts as a cap (won't speed up beyond wall-clock pace).
- Useful companion to a future load-test report (points sent, errors, elapsed time).

---

## P4 — Geospatial simulation (`--type geo`)

**Why:** Vehicle, drone, and asset-tracking use cases are large and underserved. No lightweight CLI tool generates realistic GPS tracks out of the box.

**Flags:**
```
--geo-lat        Starting latitude  (default: 48.8566)
--geo-lon        Starting longitude (default: 2.3522)
--geo-speed      Speed in m/s       (default: 10)
--geo-bearing    Initial bearing in degrees (default: 0, i.e. north)
--geo-drift      Max random bearing change per step in degrees (default: 15)
--geo-bbox       Optional bounding box "minLat,minLon,maxLat,maxLon" — wraps/bounces when breached
```

**Output:**
```json
{"device":"device","timestamp":1715000000,"lat":48.8571,"lon":2.3534}
```

In multi-field mode, `lat` and `lon` become regular named fields and all existing sinks (MQTT, Kafka, OTLP, …) work unchanged.

**Notes:**
- `--value` / `--min` / `--max` flags are ignored for this type.
- `--devices` produces independent tracks from the same starting point.
- Haversine math only — no external mapping dependency.

---

## P5 — Cloud IoT sinks

**Why:** Cloud-native teams test against managed services, not self-hosted brokers. Absence of AWS/Azure sinks is the main reason to wrap genx in custom glue code.

### AWS Kinesis Data Streams (`--output kinesis`)

```
--kinesis-stream      Stream name
--kinesis-region      AWS region (default: us-east-1)
--kinesis-partition   Partition key template (default: device name)
```

Uses the standard AWS SDK v2; credentials from env / `~/.aws/credentials` as usual.

### Azure Event Hubs (`--output eventhub`)

```
--eventhub-conn-str   Event Hub connection string
--eventhub-name       Event Hub name
```

Uses `azure-sdk-for-go`.

**Notes:**
- Implement one at a time; Kinesis first (larger user base in the OSS/DevOps space).
- Both are AMQP/HTTP under the hood — no broker to spin up in compose tests, so integration tests should be optional / behind a build tag.
