# Scenario scripting

A scenario is a list of phases executed in sequence. Each phase inherits global
defaults (device name, step, output sink, etc.) and overrides only the fields it
needs. This lets you script realistic device lifecycles — normal operation,
fault conditions, connectivity loss, recovery — in a single config file.

## This example

Four phases that simulate a sensor going through a full fault cycle:

| Phase | Duration | What it models |
|---|---|---|
| Normal | 10 min | Steady temperature (cos 20–25 °C) |
| Overheat | 5 min | Rising temperature (cos 30–50 °C) |
| Dropout | 2 min | Connectivity loss — no points emitted |
| Recovery | 10 min | Back to normal range |

## Run

```
genx --config examples/scenario/config.yaml
```

## Try it in realtime

```
genx --config examples/scenario/config.yaml --realtime
```

Each phase runs at wall-clock speed. The dropout phase is silent for 2 minutes,
then the recovery phase resumes.

## Multi-field phases

A phase can emit multiple named fields instead of a single value. Set `fields`
in that phase and each field gets its own curve:

```yaml
device: env-sensor
step: 30s

scenario:
  - duration: 10m
    type: cos
    min: 20
    max: 25

  - duration: 5m          # multi-field phase: temperature + humidity
    fields:
      temperature: { type: cos, min: 18, max: 26, period: 12h }
      humidity:    { type: cos, min: 40, max: 80, period: 8h }

  - duration: 2m          # connectivity loss
    dropout-rate: 1.0

  - duration: 10m         # recovery
    type: cos
    min: 20
    max: 25
```

During the multi-field phase each point carries a `fields` map instead of a
single `value`:

```json
{"device":"env-sensor","timestamp":1715000600,"fields":{"humidity":61.3,"temperature":22.4}}
```

## Key rules

- `duration` is required for every phase.
- `step` can be overridden per phase; falls back to the global value.
- The following fields are overridable per phase: `type`, `min`, `max`, `period`,
  `duty-cycle`, `first`, `last`, `walk-*`, `noise`, `anomaly-rate`,
  `anomaly-factor`, `dropout-rate`.
- A phase with `fields` emits multi-field points; `type`/`min`/`max` on that
  phase are ignored.
- Scenario mode is incompatible with `replay-file` and top-level `fields`.
- Fleet mode works: set `devices` or `device-names` globally and all devices
  follow the same phase sequence.

## Fleet example

```yaml
device-names: [paris, london, tokyo]
step: 1m

scenario:
  - duration: 30m
    type: cos
    min: 18
    max: 26

  - duration: 10m
    anomaly-rate: 0.3
    anomaly-factor: 5

  - duration: 30m
    type: cos
    min: 18
    max: 26
```

All three devices go through the same phases simultaneously — each with its own
independent RNG state.
