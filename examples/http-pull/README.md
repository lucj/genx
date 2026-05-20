# HTTP pull server

Starts a local HTTP server so that any client can poll for the latest data
points on demand. Useful for pull-based systems, quick `curl` inspection, or
dashboards that expect a REST endpoint rather than a push stream.

## This example

A cosine temperature curve emitted in realtime, exposed at
`http://localhost:8888/`. The last 10 points are kept in memory.

## Run

```bash
genx --config examples/http-pull/config.yaml
```

Then in another terminal:

```bash
# Fetch all buffered points (up to http-buffer)
curl http://localhost:8888/

# Fetch only the 3 most recent points
curl "http://localhost:8888/?n=3"
```

Example response:

```json
[
  {"device":"sensor","timestamp":1715000000,"value":22.14},
  {"device":"sensor","timestamp":1715000005,"value":22.31},
  {"device":"sensor","timestamp":1715000010,"value":22.47}
]
```

The response is always a JSON array, even when a single point is available.

## Key options

| Option | Default | Description |
|--------|---------|-------------|
| `http-port` | `8888` | Port the server listens on |
| `http-buffer` | `1` | Number of recent points kept in memory |

Use `?n=K` in the query string to return at most K points (must be ≤ `http-buffer`).

## Multi-device example

Works with fleet mode — each device appends to the shared buffer in the order
its point is generated:

```yaml
output: http-server
http-port: 8888
http-buffer: 30

device-names: [paris, london, tokyo]
type: cos
min: 18
max: 26
step: 5s
realtime: true
```
