# GPS fleet tracking

Simulates three trucks moving across a map starting from Paris, publishing their
position as `{lat, lon}` fields to a webhook endpoint every 10 seconds.

## Prerequisites

An HTTP endpoint to receive position updates. Start the bundled echo server:

```
docker compose up webhook
```

## Run

```
genx --config examples/gps-tracking/config.yaml
```

Each point looks like:

```json
{"device":"truck-01","timestamp":1715000010,"fields":{"lat":48.8579,"lon":2.3541}}
```

## Try it on stdout first

```
genx --type geo \
     --geo-lat 48.8566 --geo-lon 2.3522 \
     --geo-speed 15 --geo-bearing 45 --geo-drift 10 \
     --device-names truck-01,truck-02,truck-03 \
     --step 10s --duration 5m
```

## Key options

| Flag | Purpose |
|---|---|
| `--geo-lat / --geo-lon` | Starting position (decimal degrees) |
| `--geo-speed` | Speed in m/s (15 m/s ≈ 54 km/h) |
| `--geo-bearing` | Initial heading in degrees (0=N, 90=E, 180=S, 270=W) |
| `--geo-drift` | Max random bearing change per step (degrees) — higher = more winding path |

## Vary the scenario

Start multiple fleets from different cities by running genx twice with different configs:

```
# London fleet
genx --type geo --geo-lat 51.5074 --geo-lon -0.1278 \
     --geo-speed 20 --geo-bearing 90 \
     --device-names van-01,van-02 --step 10s --duration 10m

# Berlin fleet
genx --type geo --geo-lat 52.5200 --geo-lon 13.4050 \
     --geo-speed 12 --geo-bearing 270 \
     --device-names bike-01,bike-02 --step 10s --duration 10m
```
