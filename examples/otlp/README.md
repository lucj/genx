# genx OpenTelemetry example

Runs an **OTel Collector** and **Grafana** locally, then uses genx's
`--output otlp` sink to stream live IoT metrics into a pre-built dashboard.

```
genx ──OTLP gRPC──► OTel Collector ──Prometheus──► Grafana :3000
```

## Prerequisites

- Docker + Docker Compose
- `genx` binary in your `PATH`
  (build: `go build -o genx .` from the repo root, then add to `PATH`)

## Quick start

**1 — Start the stack**

```bash
cd examples/otlp
docker compose up -d
```

This starts:
| Container | Port | Purpose |
|---|---|---|
| `genx-otel-collector` | 4317 (gRPC) / 4318 (HTTP) | Receives OTLP from genx |
| `genx-grafana` | 3000 | Live dashboard |

**2 — Send data with genx**

```bash
# Single device, cosine curve, one point every 5 s
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type cos --duration 10m --step 5s --realtime

# Three-device fleet with noise
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type cos --devices 3 --noise 0.05 --duration 10m --step 5s --realtime

# Random walk
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type walk --walk-start 20 --walk-step 0.5 --duration 10m --step 5s --realtime

# Sawtooth, two devices
genx --output otlp --otlp-endpoint localhost:4317 --otlp-insecure \
     --type sawtooth --min 0 --max 100 --period 2m \
     --devices 2 --duration 10m --step 5s --realtime
```

**3 — Open Grafana**

Navigate to <http://localhost:3000> — no login required.

The **genx IoT Metrics** dashboard opens automatically. It auto-refreshes
every 5 s and shows:
- A time-series panel with one line per device
- A stat panel with the latest value per device

**4 — Stop the stack**

```bash
docker compose down
```

---

## Run everything in Docker (optional)

If you prefer not to install the genx binary, use the `demo` profile to
build and run genx inside Docker as well:

```bash
docker compose --profile demo up
```

This builds genx from the repo root and starts a 3-device cosine fleet
automatically.

---

## How it works

| Component | Role |
|---|---|
| `otelcol-config.yaml` | OTel Collector pipeline: OTLP receiver → batch processor → Prometheus exporter |
| `grafana/provisioning/datasources/` | Auto-wires the Prometheus datasource to the collector's `:9090` endpoint |
| `grafana/provisioning/dashboards/` | Provisions the `genx IoT Metrics` dashboard on startup |

The OTel Collector's Prometheus exporter converts resource attributes
(e.g. `service.name`) into metric labels, so the Grafana query
`{service_name="genx"}` picks up all metrics from genx regardless of
curve type or number of fields.

## Sending to a remote OTel backend

Swap `--otlp-endpoint` for any OTLP-compatible collector. For
authenticated backends (Grafana Cloud, Honeycomb, Datadog, …) add
`--otlp-header` for each required header:

```bash
genx --output otlp \
     --otlp-endpoint otlp.example.com:4317 \
     --otlp-header "x-api-key=your-key" \
     --type cos --duration 1h --step 10s --realtime
```
