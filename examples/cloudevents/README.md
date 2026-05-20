# CloudEvents output

Simulates three room sensors emitting temperature and humidity as
[CloudEvents 1.0](https://cloudevents.io) structured JSON, suitable for event-driven
pipelines (Knative, EventBridge, Azure Event Grid, etc.).

## Prerequisites

An HTTP endpoint to receive events. Start the bundled echo server:

```
docker compose up webhook
```

## Run

```
genx --config examples/cloudevents/config.yaml
```

Each point is a fully-formed CloudEvent:

```json
{
  "specversion": "1.0",
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "source": "/buildings/floor1/room-a",
  "type": "io.mycompany.sensor.reading",
  "time": "2024-05-07T10:00:30Z",
  "datacontenttype": "application/json",
  "data": {
    "device": "room-a",
    "timestamp": 1715076030,
    "fields": { "humidity": 61.3, "temperature": 22.4 }
  }
}
```

## Try it on stdout first

```
genx --config examples/cloudevents/config.yaml --output stdout
```

## Use ISO timestamps inside data

```
genx --config examples/cloudevents/config.yaml --output stdout --iso-time
```

The `data.timestamp` field becomes an ISO 8601 string instead of a Unix integer.

## Key options

| Flag | Purpose |
|---|---|
| `--format cloudevent` | Wraps each payload in a CloudEvents 1.0 envelope |
| `--cloudevent-source` | Sets the `source` field (appended with `/device-name`) |
| `--cloudevent-type` | Sets the `type` field |
| `--iso-time` | Formats `data.timestamp` as RFC 3339 instead of Unix seconds |

## Routing by device

Because `source` is set to `<cloudevent-source>/<device-name>`, event routers can
filter by device without inspecting the `data` payload:

```
/buildings/floor1/room-a
/buildings/floor1/room-b
/buildings/floor1/room-c
```
