# genx — Future Features

## High Priority

### HTTP Server Mode (`--output http-server`)
Instead of pushing data, genx exposes a `/metrics` (Prometheus text) or `/data` (JSON)
endpoint and serves the most-recent data point on each request.

Turns genx into a **mock sensor API** that downstream services can poll — useful for
testing dashboards, alerting rules, and data-ingestion pipelines without any
message-broker setup.

---

## Medium Priority

### Scenario / Phase Scripting (`--scenario <file.yaml>`)
Define a YAML file with a sequence of named phases, each overriding curve type, noise,
anomaly rate, and duration:

```yaml
phases:
  - name: normal
    duration: 5m
    type: cos
    noise: 0.02
  - name: fault
    duration: 2m
    type: cos
    noise: 0.3
    anomaly_rate: 0.15
  - name: recovery
    duration: 3m
    type: cos
    noise: 0.05
```

genx runs each phase in sequence, stitching timestamps together. This is the most
differentiating feature — it lets teams replay realistic incident patterns (normal →
fault → recovery) for testing anomaly-detection models and alerting pipelines.
