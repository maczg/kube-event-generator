import argparse
from datetime import datetime
import glob
import os
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates

output_dir = "plots"
data_dir = "data"


def scale_column(df, col, factor):
    """
    Multiplies the given column by 'factor'.
    """
    df[col] = df[col] * factor


def plot(df_usage, df_ratios):
    fig, axs = plt.subplots(2, 1, figsize=(12, 12), layout='constrained')
    fig.suptitle("Resource Usage and Ratio", fontsize=16)
    fig.autofmt_xdate()
    axs[0].set_title("Resource Usage")
    axs[1].set_title("Ratio")
    axs[0].set_ylabel("Usage")
    axs[1].set_ylabel("Ratio")
    axs[1].set_xlabel("Time")
    axs[0].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))
    axs[1].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))

    for label in axs[0].get_xticklabels():
        label.set_rotation(45)
        label.set_horizontalalignment('right')

    for label in axs[1].get_xticklabels():
        label.set_rotation(45)
        label.set_horizontalalignment('right')

    for cl in df_usage.columns:
        if cl != "timestamp":
            axs[0].step(df_usage.index, df_usage[cl], where='post', marker='o', label=cl)
            axs[0].legend(loc="upper left")
            axs[0].grid(True)
    for cl in df_ratios.columns:
        if cl != "timestamp":
            axs[1].step(df_ratios.index, df_ratios[cl], where='post', marker='o', label=cl)
            axs[1].legend(loc="upper left")
            axs[1].grid(True)
    plt.tight_layout()
    plt.savefig(f"{args.result_dir}/{output_dir}/node_usage_ratio.png")
    plt.close()


def load_csv(result_dir):
    files = glob.glob(f"{result_dir}/node-*.csv")
    files.sort()
    ratio_dfs = []
    usage_dfs = []
    min_time = datetime.max
    max_time = datetime.min

    for file in files:
        # extract node name
        node_name = file.split("/")[-1].split("_")[0]
        _df = pd.read_csv(file)
        _df["timestamp"] = pd.to_datetime(_df["timestamp"])
        _df.set_index("timestamp", inplace=True)
        for cl in _df.columns:
            _df.rename(columns={cl: f"{node_name.replace('-', '_')}_{cl}"}, inplace=True)
        min_time = min(min_time, _df.index.min())
        max_time = max(max_time, _df.index.max())
        if "ratio" in file:
            ratio_dfs.append(_df)
        else:
            usage_dfs.append(_df)

    if ratio_dfs:
        ratio_dfs = pd.concat(ratio_dfs, axis=1)
    else:
        ratio_dfs = pd.DataFrame()

    if usage_dfs:
        usage_dfs = pd.concat(usage_dfs, axis=1)
    else:
        usage_dfs = pd.DataFrame()

    ratio_dfs.ffill(inplace=True)
    usage_dfs.ffill(inplace=True)
    return {"ratios": ratio_dfs, "usage": usage_dfs}



def plot_rolling(df_usage_rolling, df_ratios_rolling):
    fig, axs = plt.subplots(2, 1, figsize=(12, 12), layout='constrained')
    fig.suptitle("Rolling Resource Usage and Ratio", fontsize=16)
    fig.autofmt_xdate()
    axs[0].set_title("Rolling Resource Usage")
    axs[1].set_title("Rolling Ratio")
    axs[0].set_ylabel("Usage")
    axs[1].set_ylabel("Ratio")
    axs[1].set_xlabel("Time")
    axs[0].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))
    axs[1].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))
    for label in axs[0].get_xticklabels():
        label.set_rotation(45)
        label.set_horizontalalignment('right')
    for label in axs[1].get_xticklabels():
        label.set_rotation(45)
        label.set_horizontalalignment('right')
    for cl in df_usage_rolling.columns:
        if cl != "timestamp":
            axs[0].step(df_usage_rolling.index, df_usage_rolling[cl], where='post', marker='o', label=cl)
            axs[0].legend(loc="upper left")
            axs[0].grid(True)
    for cl in df_ratios_rolling.columns:
        if cl != "timestamp":
            axs[1].step(df_ratios_rolling.index, df_ratios_rolling[cl], where='post', marker='o', label=cl)
            axs[1].legend(loc="upper left")
            axs[1].grid(True)
    plt.tight_layout()
    plt.savefig(f"{args.result_dir}/{output_dir}/node_usage_ratio_rolling.png")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--result-dir", type=str, required=True)
    args = parser.parse_args()

    if not os.path.exists(f"{args.result_dir}/{output_dir}"):
        os.makedirs(f"{args.result_dir}/{output_dir}")

    dfs = load_csv(f"{args.result_dir}/{data_dir}")

    df_usage = dfs["usage"]
    df_ratios = dfs["ratios"]

    # Scale the columns in df_usage
    for col in df_usage.columns:
        if "memory" in col:
            scale_column(df_usage, col, 1 / 1024 / 1024 / 1024)

    for col in df_ratios.columns:
        scale_column(df_ratios, col, 100)

    # save to csv
    df_usage.to_csv(f"{args.result_dir}/{output_dir}/usage.csv")
    df_ratios.to_csv(f"{args.result_dir}/{output_dir}/ratios.csv")

    plot(df_usage, df_ratios)

    stats = df_usage.describe(include="all")
    # save stats to text
    with open(f"{args.result_dir}/{output_dir}/stats_usage.txt", "w") as f:
        f.write(str(stats))

    stats = df_ratios.describe(include="all")
    with open(f"{args.result_dir}/{output_dir}/stats_ratios.txt", "w") as f:
        f.write(str(stats))
