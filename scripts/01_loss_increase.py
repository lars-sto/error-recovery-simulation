import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv("../simdata/01_loss_increase.csv")

plt.figure()
plt.plot(df["loss"], df["overhead"], marker=".")
plt.xlabel("Loss rate")
plt.ylabel("FEC overhead")
plt.title("Loss vs FEC Overhead")
plt.grid(True)
plt.show()

plt.figure()
plt.plot(df["time"], df["overhead"], marker=".")
plt.xlabel("Time step")
plt.ylabel("FEC overhead")
plt.title("Time vs FEC Overhead")
plt.grid(True)
plt.show()

plt.figure()
plt.step(df["time"], df["fec_enabled"].astype(int), where="post")
plt.xlabel("Time step")
plt.ylabel("FEC enabled")
plt.title("FEC Enable State")
plt.ylim(-0.1, 1.1)
plt.grid(True)
plt.show()