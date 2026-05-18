# Testing a data pipeline

This example shows how to use genx to test a data pipeline end to end: record a
dataset once, then replay it repeatedly through different sinks.

## 1. Record a dataset

Generate data and save it to a file. Use `--seed` so the dataset is reproducible:

```
$ genx --type cos --min 18 --max 26 --duration 1h --step 1m \
       --noise 0.03 --anomaly-rate 0.02 --seed 42 \
       > recording.jsonl
```

`recording.jsonl` now contains 60 JSON lines with a fixed anomaly pattern.

## 2. Replay to stdout (verify the file)

```
$ genx --replay-file recording.jsonl
```

Batch mode — sends all points immediately, preserving original timestamps.

## 3. Replay to a webhook

POST each point to an HTTP endpoint as if it arrived from a real device:

```
$ genx --replay-file recording.jsonl \
       --output webhook --webhook-url http://localhost:8080/ingest \
       --webhook-token mysecret
```

Use the bundled echo server to inspect payloads:

```
# terminal 1
docker compose up webhook

# terminal 2
genx --replay-file recording.jsonl --output webhook --webhook-url http://localhost:8080
```

## 4. Replay to NATS in real time

Stream the recorded data into NATS at the original 1-minute pace, stamping each point
with the current time:

```
$ genx --replay-file recording.jsonl \
       --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
       --realtime --step 1m
```

Reduce `--step` to speed up playback (e.g. `--step 5s` replays 1 hour of data in 5 minutes).

## 5. Using Docker Compose for a full end-to-end test

```bash
# Start NATS and a subscriber in the background
docker compose up -d nats
docker compose run --rm nats-sub &

# Replay the dataset
genx --replay-file recording.jsonl \
     --output nats --nats-url nats://localhost:4222 --nats-subject sensors.temp \
     --realtime --step 2s
```

## Why this pattern

- **Decouples generation from testing** — generate once, replay many times against
  different backends or configurations.
- **Deterministic anomalies** — `--seed 42` ensures the same spike appears at the same
  position every replay, so alerts fire at predictable points.
- **Adjustable playback speed** — `--step` controls replay pace independently of the
  original recording interval.
