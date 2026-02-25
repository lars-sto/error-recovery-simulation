import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
from pathlib import Path

SUMMARY_PATH = Path("../results/summary.csv")
CONFIDENCE_Z = 1.96  # 95% CI

print(SUMMARY_PATH)

# Load
if not SUMMARY_PATH.exists():
    raise FileNotFoundError(f"{SUMMARY_PATH} not found")

df = pd.read_csv(SUMMARY_PATH)

# Ensure numeric columns
numeric_cols = [
    "final_loss_deadline",
    "overhead_ratio_bytes",
    "mean_queue_delay_ms",
]
for c in numeric_cols:
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
        n=("final_loss_deadline", "count"),
    )
    .reset_index()
)

grouped["loss_ci"] = CONFIDENCE_Z * grouped["loss_std"] / np.sqrt(grouped["n"])
grouped["oh_ci"] = CONFIDENCE_Z * grouped["oh_std"] / np.sqrt(grouped["n"])
grouped["q_ci"] = CONFIDENCE_Z * grouped["q_std"] / np.sqrt(grouped["n"])


# Helper for consistent ordering
scenarios = sorted(df["scenario"].unique())
modes = ["static_flexfec", "adaptive_engine"]


# Pareto Scatter: Overhead vs Deadline-Loss
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
plt.tight_layout()
plt.show()


# Bar Plot: Mean Deadline Loss + CI
plt.figure()

x = np.arange(len(scenarios))
width = 0.35

for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").loc[scenarios]
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
plt.tight_layout()
plt.show()


# Bar Plot: Mean Overhead + CI
plt.figure()

for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").loc[scenarios]
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
plt.tight_layout()
plt.show()


# Queue Delay Comparison
plt.figure()

for i, mode in enumerate(modes):
    sub = grouped[grouped["mode"] == mode].set_index("scenario").loc[scenarios]
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
plt.tight_layout()
plt.show()