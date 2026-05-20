# genx Streamlit visualiser

An interactive browser UI that configures and runs `genx`, then plots the
result as a live line chart.

## Prerequisites

- `genx` binary in your `PATH` (build with `go build -o genx .` from the
  repo root, then move it somewhere on your `PATH`)
- Python 3.9+

## Run

```bash
cd examples/streamlit
pip install -r requirements.txt
streamlit run app.py
```

Open the URL printed by Streamlit (usually <http://localhost:8501>).

## What it does

- **Sidebar** — pick curve type, duration, step, number of devices, and
  realism options (noise, anomaly rate, dropout rate)
- **Generate button** — runs `genx --format csv` as a subprocess and reads
  the output into a Pandas DataFrame
- **Chart** — `st.line_chart` with one series per device (fleet mode) or
  one series per field (multi-field mode)
- **Raw data / command expanders** — inspect the full dataset and the exact
  `genx` command that produced it

## Examples

```bash
# Single cosine device
# → set Type=cos, Duration=1h, Step=1m, click Generate

# Three-device fleet with noise
# → set Devices=3, Spread=0.1, Noise=0.05, click Generate

# Sawtooth wave
# → set Type=sawtooth, Period=15m, Duration=1h, Step=1m, click Generate

# Square wave, 30% duty cycle
# → set Type=square, Duty cycle=0.3, Period=30m, click Generate

# Reproducible run
# → set Seed=42, click Generate (same output every time)
```
