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

## Key rules

- `duration` is required for every phase.
- `step` can be overridden per phase; falls back to the global value.
- The following fields are overridable per phase: `type`, `min`, `max`, `period`,
  `duty-cycle`, `first`, `last`, `walk-*`, `noise`, `anomaly-rate`,
  `anomaly-factor`, `dropout-rate`.
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
