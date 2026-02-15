import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv("../simdata/03_BWE_bottleneck.csv")

plt.figure()
plt.plot(df["time"], df["target_bitrate"], marker=".")
plt.plot(df["time"], df["current_bitrate"], marker=".")
plt.xlabel("Time step")
plt.ylabel("Bitrate")
plt.title("BWE Bottleneck: Target vs Current Bitrate")
plt.legend(["target_bitrate", "current_bitrate"])
plt.grid(True)
plt.show()

plt.figure()
plt.plot(df["time"], df["overhead"], marker=".")
plt.xlabel("Time step")
plt.ylabel("FEC overhead")
plt.title("BWE Bottleneck: Overhead after BWE cap")
plt.grid(True)
plt.show()

plt.figure()
plt.step(df["time"], df["fec_enabled"].astype(int), where="post")
plt.xlabel("Time step")
plt.ylabel("FEC enabled")
plt.title("BWE Bottleneck: Enable/Disable behavior")
plt.ylim(-0.1, 1.1)
plt.grid(True)
plt.show()

plt.figure()
plt.plot(df["target_bitrate"], df["overhead"], marker=".")
plt.xlabel("Target bitrate")
plt.ylabel("FEC overhead")
plt.title("BWE Bottleneck: Target bitrate vs Overhead")
plt.grid(True)
plt.show()