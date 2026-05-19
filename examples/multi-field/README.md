# Multi-field payloads with a custom template

Simulates an environment sensor emitting temperature, humidity, and pressure in a
single data point, shaped to match a specific JSON schema.

## Run

```
genx --config examples/multi-field/config.yaml
```

Output:

```json
{"device":"env-sensor","timestamp":1715000000,"fields":{"humidity":60.00,"pressure":1010.00,"temperature":22.00}}
```

## Reshape with a template

When the consuming system expects a different schema, use `--payload-template` to
reshape the output without touching the curve definitions:

```
genx --config examples/multi-field/config.yaml \
     --payload-template '{"sensor":"{{.Device}}","ts":{{.Timestamp}},"t":{{.Fields.temperature}},"h":{{.Fields.humidity}},"p":{{.Fields.pressure}}}'
```

Output:

```json
{"sensor":"env-sensor","ts":1715000000,"t":22.00,"h":60.00,"p":1010.00}
```

## Use a template file

For larger schemas, reference `template.json` included in this directory:

```
genx --config examples/multi-field/config.yaml \
     --payload-template-file examples/multi-field/template.json
```

Output:

```json
{
  "sensor_id": "env-sensor",
  "recorded_at": 1715000000,
  "measurements": {
    "temperature": 22.00,
    "humidity": 60.00,
    "pressure": 1010.00
  }
}
```

## Docker

```
docker run -i ghcr.io/lucj/genx --config - < examples/multi-field/config.yaml
```
