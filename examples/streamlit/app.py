import io
import subprocess

import pandas as pd
import streamlit as st

st.set_page_config(page_title="genx visualiser", layout="wide")
st.title("data visualiser")

# ---------------------------------------------------------------------------
# Sidebar controls
# ---------------------------------------------------------------------------
with st.sidebar:
    st.header("Curve")
    curve_type = st.selectbox(
        "Type",
        ["cos", "walk", "linear", "sawtooth", "square", "exp", "log"],
        key="curve_type",
    )
    duration = st.selectbox("Duration", ["15m", "1h", "6h", "1d", "7d"], index=1, key="duration")
    step = st.selectbox("Step", ["10s", "1m", "5m", "15m", "1h"], index=2, key="step")
    devices = st.number_input("Devices", min_value=1, max_value=20, value=1, key="devices")
    spread = st.slider("Spread", 0.0, 0.5, 0.0, 0.05, key="spread") if devices > 1 else 0.0

    if curve_type in ("cos", "sawtooth", "square"):
        st.subheader("Curve parameters")
        c1, c2 = st.columns(2)
        min_val = c1.number_input("Min", value=10.0, key="min_val")
        max_val = c2.number_input("Max", value=25.0, key="max_val")
        # cos: period = duration → one full cycle; sawtooth/square: short period → multiple cycles visible.
        period_default = {"cos": 1, "sawtooth": 0, "square": 0}
        period = st.selectbox("Period", ["15m", "1h", "6h", "12h", "1d"],
                              index=period_default.get(curve_type, 1), key="period")
        if curve_type == "square":
            duty_cycle = st.slider("Duty cycle", 0.1, 0.9, 0.5, 0.05, key="duty_cycle")
    elif curve_type == "linear":
        st.subheader("Curve parameters")
        c1, c2 = st.columns(2)
        first_val = c1.number_input("First", value=0.0, key="first_val")
        last_val = c2.number_input("Last", value=100.0, key="last_val")
    elif curve_type == "walk":
        st.subheader("Walk parameters")
        walk_start = st.number_input("Start", value=20.0, key="walk_start")
        walk_step_size = st.number_input("Step size", value=0.5, min_value=0.01, key="walk_step_size")
        walk_bias = st.number_input("Bias", value=0.0, key="walk_bias",
                                    help="Constant drift per step; negative = downward")
        c1, c2 = st.columns(2)
        walk_min = c1.number_input("Min clamp", value=0.0, key="walk_min",
                                   help="Lower bound; set both to 0 to disable clamping")
        walk_max = c2.number_input("Max clamp", value=0.0, key="walk_max",
                                   help="Upper bound; set both to 0 to disable clamping")

    st.subheader("Realism")
    noise = st.slider("Noise", 0.0, 0.5, 0.0, 0.01, key="noise")
    anomaly_rate = st.slider("Anomaly rate", 0.0, 0.2, 0.0, 0.01, key="anomaly_rate")
    if anomaly_rate > 0:
        anomaly_factor = st.slider("Anomaly factor", 1.5, 20.0, 3.0, 0.5, key="anomaly_factor",
                                   help="Spike = value × factor, drop = value ÷ factor")
    dropout_rate = st.slider("Dropout rate", 0.0, 0.5, 0.0, 0.01, key="dropout_rate")

    st.subheader("Reproducibility")
    seed = st.number_input("Seed (0 = random)", min_value=0, value=0, key="seed")

    generate = st.button("Generate", type="primary", use_container_width=True)

# ---------------------------------------------------------------------------
# Generate + render
# ---------------------------------------------------------------------------
if not generate:
    st.info("Configure parameters in the sidebar and click **Generate**.")
    st.stop()

cmd = [
    "genx",
    "--type", curve_type,
    "--duration", duration,
    "--step", step,
    "--devices", str(int(devices)),
    "--noise", str(noise),
    "--anomaly-rate", str(anomaly_rate),
    "--dropout-rate", str(dropout_rate),
    "--realtime=false",
    "--format", "csv",
]
if devices > 1 and spread > 0:
    cmd += ["--spread", str(spread)]
if seed > 0:
    cmd += ["--seed", str(int(seed))]
if anomaly_rate > 0:
    cmd += ["--anomaly-factor", str(anomaly_factor)]

if curve_type in ("cos", "sawtooth", "square"):
    cmd += ["--min", str(min_val), "--max", str(max_val), "--period", period]
    if curve_type == "square":
        cmd += ["--duty-cycle", str(duty_cycle)]
elif curve_type == "linear":
    cmd += ["--first", str(first_val), "--last", str(last_val)]
elif curve_type == "walk":
    cmd += [
        "--walk-start", str(walk_start),
        "--walk-step", str(walk_step_size),
        "--walk-bias", str(walk_bias),
    ]
    if walk_min != 0 or walk_max != 0:
        cmd += ["--walk-min", str(walk_min), "--walk-max", str(walk_max)]

with st.spinner("Generating…"):
    result = subprocess.run(cmd, capture_output=True, text=True)

if result.returncode != 0:
    st.error(f"genx error:\n```\n{result.stderr.strip()}\n```")
    st.stop()

df = pd.read_csv(io.StringIO(result.stdout))
df["timestamp"] = pd.to_datetime(df["timestamp"], unit="s", utc=True)

value_cols = [c for c in df.columns if c not in ("device", "timestamp")]

# One column per device (fleet mode); single device keeps its value column(s) as-is.
if int(devices) > 1:
    chart_df = df.pivot_table(
        index="timestamp", columns="device", values=value_cols[0], aggfunc="first"
    )
else:
    chart_df = df.set_index("timestamp")[value_cols]

st.subheader("Chart")
st.line_chart(chart_df)

col1, col2, col3, col4 = st.columns(4)
col1.metric("Points", f"{len(df):,}")
col2.metric("Devices", int(df["device"].nunique()))
col3.metric("Fields", len(value_cols))
if len(value_cols) == 1:
    col4.metric("Value range",
                f"{df[value_cols[0]].min():.2f} – {df[value_cols[0]].max():.2f}")

with st.expander("Raw data"):
    st.dataframe(df, use_container_width=True)

with st.expander("genx command"):
    st.code(" ".join(cmd), language="bash")
