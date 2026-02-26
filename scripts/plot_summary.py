import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
from pathlib import Path

# Config
SUMMARY_PATH = Path("../results/summary.csv")
PLOTS_DIR = Path("../results/plots")
TIMESERIES_DIR = Path("results/timeseries")
CONFIDENCE_Z = 1.96  # 95% CI
SHOW_PLOTS = True

PLOTS_DIR.mkdir(parents=True, exist_ok=True)

# Load summary
if not SUMMARY_PATH.exists():
    raise FileNotFoundError(f"{SUMMARY_PATH} not found")

df = pd.read_csv(SUMMARY_PATH)

numeric_cols = [
    "final_loss_deadline",
    "overhead_ratio_bytes",
    "mean_queue_delay_ms",
    "mean_policy_r",
]
for c in numeric_cols:
    if c in df.columns:
        df[c] = pd.to_numeric(df[c], errors="coerce")

# Aggregation (mean + CI)
grouped = (
    df.groupby(["scenario", "mode"])
    .agg(
        loss_mean=("final_loss_deadline", "mean"),
        loss_std=("final_loss_deadline", "std"),
        oh_mean=("overhead_ratio_bytes", "mean"),
        oh_std=("overhead_ratio_bytes", "std"),
        q_mean=("mean_queue_delay_ms", "mean"),
        q_std=("mean_queue_delay_ms", "std"),
        policy_r_mean=("mean_policy_r", "mean"),
        policy_r_std=("mean_policy_r", "std"),
        n=("final_loss_deadline", "count"),
    )
    .reset_index()
)

grouped["loss_ci"] = CONFIDENCE_Z * grouped["loss_std"].fillna(0) / np.sqrt(grouped["n"])
grouped["oh_ci"] = CONFIDENCE_Z * grouped["oh_std"].fillna(0) / np.sqrt(grouped["n"])
grouped["q_ci"] = CONFIDENCE_Z * grouped["q_std"].fillna(0) / np.sqrt(grouped["n"])
grouped["policy_r_ci"] = CONFIDENCE_Z * grouped["policy_r_std"].fillna(0) / np.sqrt(grouped["n"])

# Helpers
scenarios = sorted(df["scenario"].unique())
modes = ["static_flexfec", "adaptive_engine"]

def finish_plot(filename: str):
    plt.tight_layout()
    out = PLOTS_DIR / filename
    plt.savefig(out, dpi=200, bbox_inches="tight")
    if SHOW_PLOTS:
        plt.show()
    plt.close()
    print(f"saved: {out}")

def mode_label(mode: str) -> str:
    return "Static" if mode == "static_flexfec" else "Adaptive"

# Existing overview plots
# 0) Global Pareto Scatter
plt.figure()
for mode in modes:
    sub = df[df["mode"] == mode]
    plt.scatter(
        sub["overhead_ratio_bytes"],
        sub["final_loss_deadline"],
        alpha=0.4,
        label=mode,
    )

plt.xlabel("Overhead Ratio (bytes)")
plt.ylabel("Deadline Loss")
plt.title("Pareto Scatter: Overhead vs Deadline Loss")
plt.legend()
plt.grid(True)
finish_plot("pareto_overhead_vs_deadline_loss.png")

# 1) Deadline Loss
plt.figure()
x = np.arange(len(scenarios))
width = 0.35

for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").reindex(scenarios)
    offset = (-width / 2) if i == 0 else (width / 2)
    plt.bar(
        x + offset,
        sub["loss_mean"],
        width,
        yerr=sub["loss_ci"],
        capsize=5,
        label=mode,
        )

plt.xticks(x, scenarios, rotation=30)
plt.ylabel("Mean Deadline Loss")
plt.title("Deadline Loss (Mean ± 95% CI)")
plt.legend()
plt.grid(True, axis="y")
finish_plot("deadline_loss_mean_ci.png")

# 2) Overhead
plt.figure()
for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").reindex(scenarios)
    offset = (-width / 2) if i == 0 else (width / 2)
    plt.bar(
        x + offset,
        sub["oh_mean"],
        width,
        yerr=sub["oh_ci"],
        capsize=5,
        label=mode,
        )

plt.xticks(x, scenarios, rotation=30)
plt.ylabel("Mean Overhead Ratio (bytes)")
plt.title("Overhead (Mean ± 95% CI)")
plt.legend()
plt.grid(True, axis="y")
finish_plot("overhead_mean_ci.png")

# 3) Queue Delay
plt.figure()
for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").reindex(scenarios)
    offset = (-width / 2) if i == 0 else (width / 2)
    plt.bar(
        x + offset,
        sub["q_mean"],
        width,
        yerr=sub["q_ci"],
        capsize=5,
        label=mode,
        )

plt.xticks(x, scenarios, rotation=30)
plt.ylabel("Mean Queue Delay (ms)")
plt.title("Queue Delay (Mean ± 95% CI)")
plt.legend()
plt.grid(True, axis="y")
finish_plot("queue_delay_mean_ci.png")

# 4) Pareto scatter per scenario
for scenario in scenarios:
    plt.figure()
    sub_all = df[df["scenario"] == scenario]

    for mode in modes:
        sub = sub_all[sub_all["mode"] == mode]
        plt.scatter(
            sub["overhead_ratio_bytes"],
            sub["final_loss_deadline"],
            alpha=0.45,
            label=mode,
        )

        # mean marker
        if not sub.empty:
            plt.scatter(
                [sub["overhead_ratio_bytes"].mean()],
                [sub["final_loss_deadline"].mean()],
                marker="x",
                s=120,
                linewidths=2,
                label=f"{mode} mean",
            )

    plt.xlabel("Overhead Ratio (bytes)")
    plt.ylabel("Deadline Loss")
    plt.title(f"Pareto per Scenario: {scenario}")
    plt.legend()
    plt.grid(True)
    finish_plot(f"pareto_{scenario}.png")

# 5) Mean Policy R (Adaptive only, plus static reference line)
plt.figure()

adaptive = grouped[grouped["mode"] == "adaptive_engine"].set_index("scenario").reindex(scenarios)
x = np.arange(len(scenarios))

plt.bar(
    x,
    adaptive["policy_r_mean"],
    width=0.55,
    yerr=adaptive["policy_r_ci"],
    capsize=5,
    label="adaptive_engine",
)

# Static reference (StaticR = 2 in your current scenarios)
plt.axhline(2.0, linestyle="--", label="static reference r=2")

plt.xticks(x, scenarios, rotation=30)
plt.ylabel("Mean Policy R")
plt.title("Adaptive Mean Policy R (Mean ± 95% CI)")
plt.legend()
plt.grid(True, axis="y")
finish_plot("mean_policy_r_adaptive.png")

# 6) Time-series overlays for selected scenarios
# We look for files like:
# results/timeseries/<scenario>__<mode>__seed<seed>.csv
# and plot the first matching seed for each mode
def find_timeseries_file(scenario: str, mode: str):
    if not TIMESERIES_DIR.exists():
        return None
    pattern = f"{scenario}__{mode}__seed*.csv"
    matches = sorted(TIMESERIES_DIR.glob(pattern))
    return matches[0] if matches else None

def load_ts(path: Path):
    ts = pd.read_csv(path)
    # expected columns from CSVRecorder
    # t_ms, loss_window, target_bwe_bps, media_rate_bps, capacity_bps, current_bitrate_bps,
    # queue_delay_ms, policy_enabled, policy_k, policy_r, policy_overhead, ...
    for c in [
        "t_ms",
        "loss_window",
        "target_bwe_bps",
        "media_rate_bps",
        "capacity_bps",
        "current_bitrate_bps",
        "queue_delay_ms",
        "policy_r",
    ]:
        if c in ts.columns:
            ts[c] = pd.to_numeric(ts[c], errors="coerce")
    return ts

timeseries_scenarios = [s for s in ["bwe_bottleneck", "loss_steps", "gilbert_burst"] if s in scenarios]

for scenario in timeseries_scenarios:
    static_file = find_timeseries_file(scenario, "static_flexfec")
    adaptive_file = find_timeseries_file(scenario, "adaptive_engine")

    # Skip if not available
    if static_file is None and adaptive_file is None:
        continue

    static_ts = load_ts(static_file) if static_file else None
    adaptive_ts = load_ts(adaptive_file) if adaptive_file else None

    # 6a) Capacity vs Current Bitrate
    plt.figure()
    if static_ts is not None:
        plt.plot(static_ts["t_ms"], static_ts["capacity_bps"], label="static capacity_bps")
        plt.plot(static_ts["t_ms"], static_ts["current_bitrate_bps"], label="static current_bitrate_bps")
    if adaptive_ts is not None:
        plt.plot(adaptive_ts["t_ms"], adaptive_ts["capacity_bps"], label="adaptive capacity_bps")
        plt.plot(adaptive_ts["t_ms"], adaptive_ts["current_bitrate_bps"], label="adaptive current_bitrate_bps")
    plt.xlabel("Time (ms)")
    plt.ylabel("Bitrate (bps)")
    plt.title(f"{scenario}: Capacity vs Current Bitrate")
    plt.legend()
    plt.grid(True)
    finish_plot(f"{scenario}_timeseries_bitrate.png")

    # 6b) Queue Delay
    plt.figure()
    if static_ts is not None:
        plt.plot(static_ts["t_ms"], static_ts["queue_delay_ms"], label="static queue_delay_ms")
    if adaptive_ts is not None:
        plt.plot(adaptive_ts["t_ms"], adaptive_ts["queue_delay_ms"], label="adaptive queue_delay_ms")
    plt.xlabel("Time (ms)")
    plt.ylabel("Queue Delay (ms)")
    plt.title(f"{scenario}: Queue Delay over Time")
    plt.legend()
    plt.grid(True)
    finish_plot(f"{scenario}_timeseries_queue_delay.png")

    # 6c) Policy R
    plt.figure()
    if static_ts is not None:
        plt.plot(static_ts["t_ms"], static_ts["policy_r"], label="static policy_r")
    if adaptive_ts is not None:
        plt.plot(adaptive_ts["t_ms"], adaptive_ts["policy_r"], label="adaptive policy_r")
    plt.xlabel("Time (ms)")
    plt.ylabel("Policy R")
    plt.title(f"{scenario}: Policy R over Time")
    plt.legend()
    plt.grid(True)
    finish_plot(f"{scenario}_timeseries_policy_r.png")

    # 6d) Loss Window
    plt.figure()
    if static_ts is not None:
        plt.plot(static_ts["t_ms"], static_ts["loss_window"], label="static loss_window")
    if adaptive_ts is not None:
        plt.plot(adaptive_ts["t_ms"], adaptive_ts["loss_window"], label="adaptive loss_window")
    plt.xlabel("Time (ms)")
    plt.ylabel("Loss Window")
    plt.title(f"{scenario}: Loss Window over Time")
    plt.legend()
    plt.grid(True)
    finish_plot(f"{scenario}_timeseries_loss_window.png")