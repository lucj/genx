# Anomaly detection testing

Populates InfluxDB with a reproducible temperature dataset that contains deliberate
anomalies, so you can validate alerting rules or ML models against known spikes.

## Prerequisites

InfluxDB running locally. Start one with Docker:

```
docker run -d -p 8086:8086 \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=password \
  -e DOCKER_INFLUXDB_INIT_ORG=my-org \
  -e DOCKER_INFLUXDB_INIT_BUCKET=iot \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=my-token \
  influxdb:2
```

## Run

```
genx --config examples/anomaly-detection/config.yaml
```

Because `seed: 42` is set, every run produces identical data — the same spikes at
the same timestamps — making it suitable for CI or for sharing a specific scenario.

## Try it on stdout first

```
genx --config examples/anomaly-detection/config.yaml --output stdout
```

## Key options

| Flag | Purpose |
|---|---|
| `--anomaly-rate 0.05` | 5% of readings are outliers |
| `--anomaly-factor 5` | Outliers are 5× the normal value (spike) or ÷5 (drop) |
| `--seed 42` | Fixes the RNG — same anomaly positions every run |
| `--spread 0.08` | Devices track slightly different baselines |

## Expected output in InfluxDB

Query the data after the run:

```flux
from(bucket: "iot")
  |> range(start: -7h)
  |> filter(fn: (r) => r._measurement == "temperature")
  |> filter(fn: (r) => r._value > 100)  // catch spikes
```

## Adjusting anomaly severity

```
# More frequent but smaller spikes
genx --config examples/anomaly-detection/config.yaml \
     --anomaly-rate 0.1 --anomaly-factor 2 --output stdout

# Rare but extreme spikes
genx --config examples/anomaly-detection/config.yaml \
     --anomaly-rate 0.005 --anomaly-factor 20 --output stdout
```
