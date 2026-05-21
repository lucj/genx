# Testing a data pipeline

Record a dataset once, then replay it repeatedly through different sinks. Because the
dataset is seeded, the same anomalies appear at the same positions every time — making
it reliable for alerting rule validation or CI checks.

## 1. Record a dataset

```
genx --config examples/pipeline-testing/config.yaml > recording.jsonl
```

`recording.jsonl` now contains 60 JSON lines with a fixed anomaly pattern.

## 2. Replay to stdout (verify the file)

```
genx --replay-file recording.jsonl --realtime=false
```

Sends all 60 points immediately, preserving original timestamps.

## 3. Replay to a webhook

Start a local echo server that prints every incoming request to its logs:

```
# terminal 1 — start the echo server (stays in foreground)
docker run --rm -p 8080:8080 -e PORT=8080 ealen/echo-server
```

Or use the service defined in the root `compose.yaml`:

```
# terminal 1
docker compose up webhook

# watch what arrives
docker compose logs -f webhook
```

Then replay in a second terminal:

```
# terminal 2
genx --config examples/pipeline-testing/replay-webhook.yaml
```

Or pass flags directly:

```
genx --replay-file recording.jsonl --output webhook --webhook-url http://localhost:8080 --realtime=false
```

## 4. Replay to NATS in real time

Stream data into NATS at the original 1-minute pace:

```
genx --replay-file recording.jsonl \
     --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
     --realtime --step 1m
```

Use a shorter `--step` to speed up playback — `--step 5s` replays 1 hour of data in 5 minutes.

## Why this pattern

- **Decouples generation from testing** — generate once, replay many times against different backends.
- **Deterministic anomalies** — `seed: 42` ensures the same spike at the same position every replay.
- **Adjustable playback speed** — `--step` controls replay pace independently of the original recording interval.
