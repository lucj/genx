# Multi-field payloads with a custom template

This example simulates an environment sensor that emits temperature, humidity, and
pressure in a single data point, shaped to match a specific JSON schema.

## Config

```yaml
# examples/env-sensor.yaml
duration: 1h
step: 1m
device: env-sensor

fields:
  temperature:
    type: cos
    min: 18
    max: 26
    period: 12h
  humidity:
    type: cos
    min: 40
    max: 80
    period: 8h
  pressure:
    type: linear
    first: 1010
    last: 1015
```

## Default output

```
$ genx --config examples/env-sensor.yaml
{"device":"env-sensor","timestamp":1715000000,"fields":{"humidity":60.00,"pressure":1010.00,"temperature":22.00}}
```

## Custom template

When the consuming system expects a different schema, use `payload-template` to
reshape the output without touching the curve definitions.

```yaml
# add to env-sensor.yaml
payload-template: >-
  {"sensor":"{{.Device}}","recorded_at":{{.Timestamp}},
  "temp_c":{{.Fields.temperature}},
  "humidity_pct":{{.Fields.humidity}},
  "pressure_hpa":{{.Fields.pressure}}}
```

Or pass the template inline:

```
$ genx --config examples/env-sensor.yaml \
       --payload-template '{"sensor":"{{.Device}}","ts":{{.Timestamp}},"t":{{.Fields.temperature}},"h":{{.Fields.humidity}},"p":{{.Fields.pressure}}}'
{"sensor":"env-sensor","ts":1715000000,"t":22.00,"h":60.00,"p":1010.00}
```

## Template file

For larger schemas, put the template in a separate file:

```json
{
  "sensor_id": "{{.Device}}",
  "recorded_at": {{.Timestamp}},
  "measurements": {
    "temperature": {{.Fields.temperature}},
    "humidity":    {{.Fields.humidity}},
    "pressure":    {{.Fields.pressure}}
  }
}
```

```
$ genx --config examples/env-sensor.yaml --payload-template-file template.json
```

## Docker

```
$ docker run -i ghcr.io/lucj/genx --config - < examples/env-sensor.yaml
```
