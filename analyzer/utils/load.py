import glob
import pandas as pd
from datetime import datetime

SCALE_MEMORY_FACTOR = (1 / 1024 / 1024 / 1024)  # Scale memory from bytes to GB
SCALE_RATIO_FACTOR = 100  # Scale ratio from 0-1 to 0-100


def scale_column(df, col, factor):
    """
    Multiplies the given column by 'factor'.
    """
    df[col] = df[col] * factor


def load_node_usage(result_dir: str) -> dict[str, pd.DataFrame]:
    """
    Load node usage and ratio data from CSV files in the specified directory.
    It loads all CSV files that start with "node-" and contain either "usage" or "ratio" in their names.
    It return a dictionary with three keys:
    - "ratios": DataFrame containing the ratio data
    - "usage": DataFrame containing the usage data
    - "free": DataFrame containing the free data
    The DataFrames are indexed by timestamp and have columns named after the node names.

    :param result_dir:
    :return:
    """
    files = glob.glob(f"{result_dir}/node-*.csv")
    files.sort()
    ratio_dfs = []
    usage_dfs = []
    free_dfs = []

    min_time = datetime.max
    max_time = datetime.min

    for file in files:
        # extract node name
        node_name = file.split("/")[-1].split("_")[0]
        _df = pd.read_csv(file)
        _df["timestamp"] = pd.to_datetime(_df["timestamp"])
        _df.set_index("timestamp", inplace=True)
        for cl in _df.columns:
            if "pods" in cl:
                # drop
                _df.drop(columns=[cl], inplace=True)
                continue
            _df.rename(columns={cl: f"{node_name.replace('-', '_')}_{cl}"}, inplace=True)
        min_time = min(min_time, _df.index.min())
        max_time = max(max_time, _df.index.max())

        if "ratio" in file:
            ratio_dfs.append(_df)
        elif "free" not in file:
            usage_dfs.append(_df)
        else:
            free_dfs.append(_df)

    if ratio_dfs:
        ratio_dfs = pd.concat(ratio_dfs, axis=1)
    else:
        ratio_dfs = pd.DataFrame()

    if usage_dfs:
        usage_dfs = pd.concat(usage_dfs, axis=1)
    else:
        usage_dfs = pd.DataFrame()

    if free_dfs:
        free_dfs = pd.concat(free_dfs, axis=1)
    else:
        free_dfs = pd.DataFrame()

    ratio_dfs.ffill(inplace=True)
    usage_dfs.ffill(inplace=True)
    free_dfs.ffill(inplace=True)

    for col in usage_dfs.columns:
        if "memory" in col:
            scale_column(usage_dfs, col, SCALE_MEMORY_FACTOR)

    for col in free_dfs.columns:
        if "memory" in col:
            scale_column(usage_dfs, col, SCALE_MEMORY_FACTOR)

    for col in ratio_dfs.columns:
        scale_column(ratio_dfs, col, SCALE_RATIO_FACTOR)

    ratio_dfs = ratio_dfs.round(2)
    usage_dfs = usage_dfs.round(2)
    free_dfs = free_dfs.round(2)
    return {"ratios": ratio_dfs, "usage": usage_dfs, "free": free_dfs}
