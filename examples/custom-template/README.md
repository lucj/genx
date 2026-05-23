# Custom payload template

Shows how to reshape genx output to match an existing API contract using
`--payload-template` or `--payload-template-file`, without touching the
curve definitions.

## The config

Three fields, each using a different curve type:

| Field | Type | What it models |
|---|---|---|
| `temperature` | `cos` | Slow sinusoidal oscillation (18–26 °C over 12 h) |
| `pressure` | `linear` | Steady drift (1010 → 1015 hPa over 1 h) |
| `vibration` | `walk` | Random sensor noise around 0.5 |

## Default output

```
genx --config examples/custom-template/config.yaml
```

```json
{"device":"sensor","timestamp":1715000000,"fields":{"pressure":1010.00,"temperature":26.00,"vibration":0.52}}
```

## Inline template

Remap the fields to a custom schema in one flag:

```
genx --config examples/custom-template/config.yaml \
     --payload-template '{"sensor_id":"{{.Device}}","ts":{{.Timestamp}},"readings":{"temperature":{{.Fields.temperature}},"pressure":{{.Fields.pressure}},"vibration":{{.Fields.vibration}}}}'
```

```json
{"sensor_id":"sensor","ts":1715000000,"readings":{"temperature":26.00,"pressure":1010.00,"vibration":0.52}}
```

## Template file

For larger or multi-line schemas, reference `template.json` in this directory:

```
genx --config examples/custom-template/config.yaml \
     --payload-template-file examples/custom-template/template.json
```

The template uses `{{.TimestampISO}}` instead of `{{.Timestamp}}` to emit a
human-readable timestamp:

```json
{"sensor_id":"sensor","recorded_at":"2024-05-07T10:00:00Z","readings":{"temperature":26.00,"pressure":1010.00,"vibration":0.52}}
```

## Available placeholders

| Placeholder | Description |
|---|---|
| `{{.Device}}` | Device name |
| `{{.Timestamp}}` | Unix epoch (integer seconds) |
| `{{.TimestampISO}}` | ISO 8601 UTC string |
| `{{.Value}}` | Single field value (single-field mode) |
| `{{.Fields.name}}` | Named field value (multi-field mode) |
