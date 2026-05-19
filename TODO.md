# TODO

Feature backlog in suggested implementation order.

---

## P1 — Scenario multi-field support

**Why:** Scenario phases currently only work in single-field mode. Real device payloads emit multiple fields (temperature, humidity, pressure), so a scenario that can't vary `fields` per phase is limited for testing realistic pipelines.

**Idea:** Allow each phase in a scenario to define a `fields` map that overrides the global one. If a phase has no `fields` key, it inherits the global fields (or falls back to single-field mode).

```yaml
device: env-sensor
step: 30s

scenario:
  - duration: 10m
    fields:
      temperature: { type: cos, min: 20, max: 25, period: 12h }
      humidity:    { type: cos, min: 40, max: 80, period: 8h }
  - duration: 5m      # sensor fault — only temperature reported
    fields:
      temperature: { type: cos, min: 60, max: 80, period: 1h }
  - duration: 10m     # recovery
    fields:
      temperature: { type: cos, min: 20, max: 25, period: 12h }
      humidity:    { type: cos, min: 40, max: 80, period: 8h }
```

**Notes:**
- Requires `PhaseConfig` to gain a `Fields map[string]FieldConfig` field.
- `buildFieldFn` already handles per-field curve building; reuse it per phase.
- The global top-level `fields` incompatibility with scenario can be relaxed once this is implemented.

---

## P1 — Count mode (`--count`)

**Why:** "Emit exactly 1000 points" is easier to reason about than computing `--duration` from `--step`. Common in load testing and CI fixtures where you want a fixed dataset size regardless of step interval.

**Idea:** Add `--count N` as an alternative to `--duration`. When set, the run stops after N points per device regardless of elapsed time. `--duration` and `--count` are mutually exclusive.

```
# Emit exactly 500 points per device
genx --type cos --devices 3 --count 500 --step 1s

# In config file
count: 500
```

**Notes:**
- In batch mode, timestamps are still computed as `start + i * stepSeconds`.
- In realtime mode, the run stops after N ticks per device.
- `--count` and `--duration` should error if both are set.
- Add `Count *int` to `Config` and a `count int` flag.

---

## P2 — Multiple output sinks simultaneously

**Why:** Writing to both stdout and InfluxDB at the same time is useful for debugging a live run — you see the data and it gets recorded. Also useful for fan-out scenarios (MQTT + webhook).

**Idea:** Allow `--output` to accept a comma-separated list, or add a `outputs` list in the config file. Each sink receives every point independently.

```yaml
outputs:
  - stdout
  - influxdb
  - webhook

influxdb-url: http://localhost:8086
webhook-url: http://localhost:8080
```

**Notes:**
- Implement a `fanoutSink` that wraps multiple `Sink` instances and calls `Send` on each.
- Errors from individual sinks should be logged but not abort the others.
- The `statsSink` wraps the fanout so the summary counts all sends across all sinks.

---

## P2 — Per-device scenario phases

**Why:** In a heterogeneous fleet, different devices may fault at different times. Right now all devices in a fleet follow the identical phase sequence simultaneously.

**Idea:** Allow `device-scenarios` in the config — a map from device name to its own scenario.

```yaml
device-names: [paris, london, tokyo]
step: 1m

device-scenarios:
  paris:
    - duration: 10m
      type: cos
      min: 20
      max: 25
    - duration: 5m
      dropout-rate: 1.0
  london:
    - duration: 15m
      type: cos
      min: 18
      max: 22
  tokyo:
    - duration: 10m
      type: walk
```

**Notes:**
- Devices not listed in `device-scenarios` fall back to the global `scenario` or single-phase behaviour.
- Requires running each device's scenario independently (already the case in realtime mode via goroutines).
- Batch mode would need each device to emit its own timestamp sequence.

---

## P3 — HTTP pull sink

**Why:** Pull-based systems (Prometheus, polling consumers) can't be tested with a push sink. An HTTP server that serves the latest point on demand fills this gap without requiring a separate mock server.

**Idea:** Add `--output http-server` that starts a local HTTP server. Each GET request to `/` returns the most recent point (or a configurable number of recent points).

```
genx --type cos --output http-server --http-port 8888 --realtime --step 5s
curl http://localhost:8888/
```

**Notes:**
- The server runs in a goroutine alongside the generator.
- A simple in-memory ring buffer holds the last N points.
- Could support `/metrics` in Prometheus exposition format as an alternative to the existing Prometheus pull sink.

---

## P3 — `genx validate` subcommand

**Why:** Long realtime runs shouldn't fail after 10 minutes because of a typo in the config. A validate subcommand lets users check config correctness without running.

**Idea:** Add a `validate` subcommand that loads the config, runs all validation checks, resolves device names, parses durations, and prints a summary of what would run — without connecting to any sink or emitting any data.

```
genx validate --config examples/scenario/config.yaml
✓ Config loaded
✓ 1 device: sensor-1
✓ 4 scenario phases: 10m normal, 5m overheat, 2m dropout, 10m recovery
✓ Total duration: 27m, ~54 points
✓ Output: stdout
```

**Notes:**
- Reuses all existing validation logic.
- Does not instantiate sinks (no network connections, no file creation).
- Could warn on suspicious values (e.g. anomaly-rate=0.99, very short step with realtime).

---

## P3 — Shell completion

**Why:** Tab completion for flags reduces friction for new users and is expected from mature CLI tools. Cobra supports it with minimal effort.

**Idea:** Wire up cobra's built-in completion command.

```
genx completion bash  > /etc/bash_completion.d/genx
genx completion zsh   > ~/.zsh/completions/_genx
genx completion fish  > ~/.config/fish/completions/genx.fish
```

**Notes:**
- One line in `main()`: `rootCmd.AddCommand(rootCmd.GenBashCompletionFile(...))` or just expose `completion` via cobra's default.
- No custom logic needed — cobra generates completions from flag definitions automatically.
- Add `--output` value completion (stdout, webhook, nats, mqtt, file, kafka, otlp, prometheus, influxdb) and `--type` value completion.

---
