# genx — Feature Backlog

Priority levels: 🔴 High · 🟡 Medium · 🟢 Nice to have

Items marked ✅ are implemented.

---

## The 3 next features that would make genx stand out

These are picked for differentiation — no other lightweight CLI simulator does all three well:

1. ~~**Multi-field payloads**~~ ✅ — closes the biggest gap vs real IoT devices; makes the tool credible for actual teams, not just demos
2. **Anomaly injection** — unique angle: test your alerting pipeline without crafting edge cases by hand
3. ~~**Reproducible runs (`--seed`)**~~ ✅ — makes genx CI-ready and scenario-shareable; rare in data simulation tools

---

## Realism

### ✅ Noise injection
`--noise <ratio>` adds multiplicative random jitter to every sample (e.g. `0.05` = ±5%).
Raw sensor data is never a perfect mathematical function.

### 🔴 Anomaly injection
`--anomaly-rate <probability>` randomly spikes or drops a value (e.g. `0.01` = 1% of points).
Unique feature: lets you test alert pipelines and anomaly-detection systems without hand-crafting
edge-case datasets. No other lightweight CLI simulator does this out of the box.

### 🟡 Missing data / dropouts
`--dropout-rate <probability>` randomly skips sending a point.
Simulates connectivity loss or sensor failure; lets consumers prove they handle gaps correctly.
Pair with anomaly injection for a full fault-simulation mode.

### ✅ Reproducible runs (`--seed <int>`)
Fix the random number generator seed so every run with the same parameters produces identical output.
Guaranteed in batch mode; best-effort in realtime fleet mode (goroutine scheduling is non-deterministic).
Also configurable via YAML (`seed: 42`).

Note: different from replay mode — `--seed` makes the generator deterministic, not replaying captured data.

---

## Multi-device simulation

### ✅ Fleet mode (`--devices`, `--spread`)
Spin up N devices simultaneously, each with a random value offset.
Real-time mode uses one goroutine per device; all sinks are concurrency-safe.

---

## Curve types

### ✅ cos · linear · log · exp

### ✅ Random walk (`--type walk`)
Stateful closure that drifts by a random delta each sample. Path-dependent values
make it more authentic than smooth curves for battery drain, temperature drift, etc.
Flags: `--walk-start`, `--walk-step`, `--walk-bias`, `--walk-min`, `--walk-max`.
Fleet mode: each device gets its own independent walk; spread varies the starting value.

### 🟢 Sawtooth / square wave
Useful for simulating on/off equipment cycles (pumps, valves, HVAC units).
`--type sawtooth` and `--type square` with a `--duty-cycle` for square waves.

---

## Richer data model

### ✅ Multi-field payloads
Emit `{"fields": {"temperature": 22.4, "humidity": 58.1, "pressure": 1013.2}}` instead of a single `value`.
Available via config file only (CLI flags remain single-value). Each field has its own
independent curve type and parameters. Spread and noise apply per-field per-sample.

### ✅ Custom payload template (`--payload-template` / `--payload-template-file`)
Define the exact JSON shape using Go `text/template` syntax.
Placeholders: `{{.Device}}`, `{{.Timestamp}}`, `{{.Value}}`, `{{.Fields.fieldname}}`.
Template output is validated as JSON before sending. Inline string or file path accepted;
also configurable via YAML (`payload-template:` / `payload-template-file:`).

### 🟢 CSV output
`--format csv` on the stdout sink.
Makes it easy to pipe into spreadsheets, InfluxDB line protocol converters, or Pandas.

---

## More sinks

### 🟡 File sink with rotation
Write JSON lines to disk, rotating by size or time (`--file-path`, `--file-max-size`).
Useful for generating fixture datasets and offline testing without running a broker.

### 🟢 Kafka
`--kafka-brokers`, `--kafka-topic`. The obvious missing broker for larger pipelines.
Would use `segmentio/kafka-go` to avoid CGO dependency.

---

## Operational

### ✅ YAML config file (`--config`)
Flags always take precedence over config values.

### ✅ Authentication
NATS (`--nats-user`, `--nats-password`), MQTT (`--mqtt-user`, `--mqtt-password`),
webhook (`--webhook-token`).

### 🔴 Graceful shutdown
Currently Ctrl+C in realtime fleet mode kills goroutines mid-flight.
Catch `SIGINT`/`SIGTERM`, signal each device goroutine to stop after its current point,
then drain sinks before exiting. Correctness fix more than a feature.

### ✅ Replay mode (`--replay-file`)
Feed a recorded JSON-lines file back through any sink.
Batch mode: sends immediately, preserving original timestamps.
Realtime mode (`--realtime`): waits `--step` between sends, stamps current time.
Handles both single-field and multi-field payloads; skips invalid/empty lines gracefully.
