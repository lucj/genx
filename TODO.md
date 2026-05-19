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

