import argparse
import os

import pandas as pd

from utils.load import load_node_usage

POD_RUNNING_DURATION_FILE = "pod_running_durations.csv"
POD_PENDING_DURATION_FILE = "pod_pending_durations.csv"
PENDING_QUEUE_LENGTH_FILE = "pod_queue_length.csv"

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--result-dir", type=str, required=True)
    args = parser.parse_args()

    input_dir = f"{args.result_dir}/data"
    output_dir = f"{args.result_dir}/cleaned"

    if not os.path.exists(input_dir):
        raise FileNotFoundError(f"Input directory {input_dir} does not exist.")

    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    dfs = load_node_usage(input_dir)

    df_usage = dfs["usage"]
    df_ratios = dfs["ratios"]
    df_free = dfs["free"]

    cpu_cols = [col for col in df_usage.columns if "cpu" in col.lower()]
    mem_cols = [col for col in df_usage.columns if "mem" in col.lower()]

    df_usage["total_cpu"] = df_usage[cpu_cols].sum(axis=1)
    df_usage["total_mem"] = df_usage[mem_cols].sum(axis=1)
    df_usage["elapsed_seconds"] = (df_usage.index - df_usage.index[0]).total_seconds()

    pod_running_duration = pd.read_csv(f"{input_dir}/{POD_RUNNING_DURATION_FILE}")
    pod_pending_duration = pd.read_csv(f"{input_dir}/{POD_PENDING_DURATION_FILE}")
    pending_queue_duration = pd.read_csv(f"{input_dir}/{PENDING_QUEUE_LENGTH_FILE}")
    df_usage.to_csv(f"{output_dir}/resource_usage.csv")
    df_ratios.to_csv(f"{output_dir}/resource_usage_ratios.csv")
    df_free.to_csv(f"{output_dir}/resource_free.csv")
    pod_running_duration.to_csv(f"{output_dir}/pod_running_durations.csv")
    pod_pending_duration.to_csv(f"{output_dir}/pod_pending_durations.csv")
    pending_queue_duration.to_csv(f"{output_dir}/pod_queue_length.csv")
