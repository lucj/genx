# IoT fleet simulation

Simulates five named temperature sensors publishing to MQTT in real time, with
per-device value variation, realistic noise, and occasional anomalies.

## Prerequisites

A running MQTT broker on `localhost:1883`. Start one with Docker:

```
docker run -d -p 1883:1883 eclipse-mosquitto
```

## Run

```
genx --config examples/iot-fleet/config.yaml
```

Subscribe in another terminal to watch points arrive:

```
mosquitto_sub -t 'factory/temperature' -v
```

## Try it without MQTT first

```
genx --type cos --min 18 --max 26 --period 12h \
     --device-names paris,london,tokyo,berlin,sydney \
     --spread 0.1 --noise 0.02 \
     --anomaly-rate 0.01 --anomaly-factor 4 \
     --duration 5m --step 10s --realtime
```

## Key options

| Flag | Purpose |
|---|---|
| `--device-names` | Named devices instead of auto-numbered `sensor-0…N` |
| `--spread 0.1` | Each device gets a ±10% random offset so curves diverge |
| `--noise 0.02` | ±2% sample jitter, mimicking measurement error |
| `--anomaly-rate 0.01` | ~1% of readings are spikes (×4) or drops (÷4) |

## Reproducible runs

Pin `--seed` to replay the exact same anomaly pattern:

```
genx --config examples/iot-fleet/config.yaml --seed 42 --output stdout
```
