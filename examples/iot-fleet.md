# IoT fleet simulation

This example simulates a fleet of temperature sensors publishing to MQTT in real time,
with per-device value variation, realistic noise, and occasional anomalies to exercise
alerting rules.

## Run locally (stdout)

```
$ genx --type cos --min 18 --max 26 --period 12h \
       --devices 5 --device sensor --spread 0.1 --noise 0.02 \
       --anomaly-rate 0.01 --anomaly-factor 4 \
       --duration 1h --step 1m --realtime
{"device":"sensor-0","timestamp":1715000060,"value":22.14}
{"device":"sensor-1","timestamp":1715000060,"value":21.87}
{"device":"sensor-2","timestamp":1715000060,"value":23.61}
{"device":"sensor-3","timestamp":1715000060,"value":88.44}   <- anomaly (spike)
{"device":"sensor-4","timestamp":1715000060,"value":22.05}
```

- `--spread 0.1` gives each device a ±10% random offset so they don't all track the same curve.
- `--noise 0.02` adds ±2% sample-level jitter, mimicking sensor measurement error.
- `--anomaly-rate 0.01` means roughly 1% of readings are outliers — either a spike (×4) or a drop (÷4).

## Publish to MQTT

```yaml
# examples/fleet-mqtt.yaml
type: cos
min: 18
max: 26
period: 12h
devices: 5
device: sensor
spread: 0.1
noise: 0.02
anomaly-rate: 0.01
anomaly-factor: 4
duration: 1h
step: 30s
realtime: true

output: mqtt
mqtt-broker: tcp://localhost:1883
mqtt-topic: factory/temperature
```

```
$ genx --config examples/fleet-mqtt.yaml
```

## Reproducible scenario

Pin the RNG seed to replay the exact same fault pattern every time — useful for CI or
sharing a specific anomaly scenario with a colleague:

```
$ genx --type cos --devices 5 --spread 0.1 --noise 0.02 \
       --anomaly-rate 0.05 --anomaly-factor 4 \
       --seed 42 --duration 10m --step 1m
```

Every run with `--seed 42` produces identical output.

## Using Docker Compose

Start a local MQTT broker and watch messages arrive:

```
# terminal 1 — subscriber
docker compose run --rm mqtt-sub

# terminal 2 — publisher
docker run -i ghcr.io/lucj/genx --config - < examples/fleet-mqtt.yaml
```
