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

## P2 — Cloud IoT sinks

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
