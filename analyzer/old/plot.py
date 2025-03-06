import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import pandas as pd
import numpy as np


def plot_usage(df_usage, df_ratios, save_dir):
    fig, axs = plt.subplots(4, 1, figsize=(12, 12), layout='constrained')
    fig.suptitle("Resource Usage and Ratio", fontsize=16)
    fig.autofmt_xdate()

    axs[0].set_title("CPU Usage")
    axs[0].set_ylabel("CPU")
    axs[0].set_xlabel("Time")
    axs[1].set_title("Memory Usage")
    axs[1].set_ylabel("Memory \\(GB\\)")
    axs[1].set_xlabel("Time")
    axs[2].set_title("CPU Ratio")
    axs[2].set_ylabel("CPU")
    axs[2].set_xlabel("Time")
    axs[3].set_title("Memory Ratio")
    axs[3].set_ylabel("Memory \\(GB\\)")
    axs[3].set_xlabel("Time")

    for ax in axs:
        ax.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))
        for lbl in ax.get_xticklabels():
            lbl.set_rotation(45)
            lbl.set_horizontalalignment("right")
        ax.grid(True)

    # CPU usage
    for cl in df_usage.columns:
        if "cpu" in cl.lower():
            axs[0].step(df_usage.index, df_usage[cl], where="post", marker="o", label=cl)
            axs[0].legend(loc="upper left")

        if "mem" in cl.lower():
            axs[1].step(df_usage.index, df_usage[cl], where="post", marker="o", label=cl)
            axs[1].legend(loc="upper left")

    # Ratios
    for cl in df_ratios.columns:
        if "cpu" in cl.lower():
            axs[2].step(df_ratios.index, df_ratios[cl], where="post", marker="o", label=cl)
            axs[2].legend(loc="upper left")
        if "mem" in cl.lower():
            axs[3].step(df_ratios.index, df_ratios[cl], where="post", marker="o", label=cl)
            axs[3].legend(loc="upper left")

    # plt.tight_layout()
    plt.savefig(f"{save_dir}/node_usage.png")
    plt.close()


if __name__ == "__main__":
    import argparse
    import os

    parser = argparse.ArgumentParser()
    parser.add_argument("--run-name", type=str, required=True)
    args = parser.parse_args()

    input_dir = f"./data/{args.run_name}"
    output_dir = f"./data/{args.run_name}"

    # Example usage
    df_usage = pd.DataFrame()  # Replace with actual DataFrame
    df_ratios = pd.DataFrame()  # Replace with actual DataFrame
    plot_usage(df_usage, df_ratios, args.save_dir)

    stats = df_usage.describe(include="all")
    with open(f"{output_dir}/stats_usage.txt", "w") as f:
        f.write(str(stats))
    stats = df_ratios.describe(include="all")
    with open(f"{output_dir}/stats_ratios.txt", "w") as f:
        f.write(str(stats))



    df_usage = df_usage.round(2)
    df_ratios = df_ratios.round(2)

    # 2) Use trapezoidal rule for CPU and Memory
    total_cpu_area = np.trapezoid(df_usage["total_cpu"], x=df_usage["elapsed_seconds"]).round(2)
    total_mem_area = np.trapezoid(df_usage["total_mem"], x=df_usage["elapsed_seconds"]).round(2)
    print(f"\nIntegrated overall CPU usage (area): {total_cpu_area}")
    print(f"Integrated overall Memory usage (area): {total_mem_area}")
