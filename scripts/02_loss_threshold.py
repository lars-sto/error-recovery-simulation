import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv("../simdata/02_loss_threshold_oscillation.csv")

plt.figure()
plt.plot(df["time"], df["loss"], marker=".")
plt.axhline(0.05, linestyle="--", label="Enable threshold")
plt.axhline(0.01, linestyle="--", label="Disable threshold")
plt.xlabel("Time step")
plt.ylabel("Loss rate")
plt.title("Loss Threshold Oscillation")
plt.legend()
plt.grid(True)
plt.show()

plt.figure()
plt.plot(df["time"], df["overhead"], marker=".")
plt.xlabel("Time step")
plt.ylabel("FEC overhead")
plt.title("Overhead under Loss Oscillation")
plt.grid(True)
plt.show()

plt.figure()
plt.step(df["time"], df["fec_enabled"].astype(int), where="post")
plt.xlabel("Time step")
plt.ylabel("FEC enabled")
plt.title("FEC Enable State (Hysteresis)")
plt.ylim(-0.1, 1.1)
plt.grid(True)
plt.show()