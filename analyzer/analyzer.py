import os
import glob
import logging
import re

import pandas as pd
import numpy as np
from datetime import datetime
from matplotlib import pyplot as plt
from scipy.stats import variation

from util import scale_column, parse_cpu, parse_mem

SCALE_MEMORY_FACTOR = (1 / 1024 / 1024 / 1024)  # Scale memory from bytes to GB
SCALE_RATIO_FACTOR = 100  # Scale ratio from 0-1 to 0-100
logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")


class Analyzer:
    def __init__(self, data_dir):
        self.data_dir = data_dir
        self.run_name = data_dir.split("/")[-1]
        self.output_dir = os.path.join(data_dir, "report")
        self.resource_free_df = None
        self.resource_usage_df = None
        self.resource_usage_ratio_df = None
        self.pod_pending_duration_df = None
        self.pod_queue_length_df = None
        self.pod_running_times_df = None
        self.timeline_df = None
        self.report = {}
        self.load_data()

    def load_data(self):
        """
        Load node usage and ratio data from CSV files in the specified directory.
        It loads all CSV files that start with "node-" and contain either "usage" or "ratio" in their names.
        Data are index by timestamp, cleaned and scaled. (e.g. memory in MB, ratio in %)

        :return:
        """
        if not os.path.exists(self.data_dir):
            raise FileNotFoundError(f"Input directory {self.data_dir} does not exist.")

        resource_files = glob.glob(f"{self.data_dir}/node-*.csv")
        resource_files.sort()
        min_time = datetime.max
        max_time = datetime.min
        ratio_dfs = []
        usage_dfs = []
        free_dfs = []
        for file in resource_files:
            node_name = file.split("/")[-1].split("_")[0]
            df = pd.read_csv(file)
            df["timestamp"] = pd.to_datetime(df["timestamp"])
            df.set_index("timestamp", inplace=True)
            df = df[~df.index.duplicated(keep='first')]
            for cl in df.columns:
                if "pods" in cl:
                    df.drop(columns=[cl], inplace=True)
                    continue
                df.rename(columns={cl: f"{node_name.replace('-', '_')}_{cl}"}, inplace=True)
            min_time = min(min_time, df.index.min())
            max_time = max(max_time, df.index.max())
            if "ratio" in file:
                ratio_dfs.append(df)
            elif "free" not in file:
                usage_dfs.append(df)
            else:
                free_dfs.append(df)

        self.resource_usage_ratio_df = pd.concat(ratio_dfs, axis=1) if ratio_dfs else pd.DataFrame()
        self.resource_usage_df = pd.concat(usage_dfs, axis=1) if usage_dfs else pd.DataFrame()
        self.resource_free_df = pd.concat(free_dfs, axis=1) if free_dfs else pd.DataFrame()

        self.resource_usage_ratio_df.ffill(inplace=True)
        self.resource_usage_df.ffill(inplace=True)
        self.resource_free_df.ffill(inplace=True)

        for col in self.resource_usage_df.columns:
            if "memory" in col:
                scale_column(self.resource_usage_df, col, SCALE_MEMORY_FACTOR)

        for col in self.resource_free_df.columns:
            if "memory" in col:
                scale_column(self.resource_free_df, col, SCALE_MEMORY_FACTOR)

        for col in self.resource_usage_ratio_df.columns:
            scale_column(self.resource_usage_ratio_df, col, SCALE_RATIO_FACTOR)

        self.resource_usage_df = self.resource_usage_df.round(2)
        self.resource_free_df = self.resource_free_df.round(2)
        self.resource_usage_ratio_df = self.resource_usage_ratio_df.round(2)
        self.pod_pending_duration_df = pd.read_csv(os.path.join(self.data_dir, 'pod_pending_durations.csv'))
        self.pod_queue_length_df = pd.read_csv(os.path.join(self.data_dir, 'pod_queue_length.csv'))
        self.pod_running_times_df = pd.read_csv(os.path.join(self.data_dir, 'pod_running_durations.csv'))
        self.pod_queue_length_df['timestamp'] = pd.to_datetime(self.pod_queue_length_df['timestamp'])
        self.timeline_df = pd.read_csv(os.path.join(self.data_dir, 'event_history.csv'))
        self.timeline_df['timestamp'] = pd.to_datetime(self.timeline_df['timestamp'])
        self.timeline_df.set_index("timestamp", inplace=True)
        self.timeline_df = self.timeline_df[~self.timeline_df.index.duplicated(keep='first')]
        # Extract CPU/memory requests, converting from e.g. "421m" and "1044Mi" to floats
        self.timeline_df['cpu_req'] = self.timeline_df['cpu_req'].apply(parse_cpu)
        self.timeline_df['mem_req'] = self.timeline_df['mem_req'].apply(parse_mem)

        cpu_cols = [col for col in self.resource_usage_df.columns if "cpu" in col.lower()]
        mem_cols = [col for col in self.resource_usage_df.columns if "mem" in col.lower()]

        self.resource_usage_df["total_cpu"] = self.resource_usage_df[cpu_cols].sum(axis=1).round(2)
        self.resource_usage_df["total_memory"] = self.resource_usage_df[mem_cols].sum(axis=1).round(2)
        self.resource_usage_df["elapsed_seconds"] = (
                self.resource_usage_df.index - self.resource_usage_df.index[0]).total_seconds()

        self.resource_free_df["total_cpu"] = self.resource_free_df[cpu_cols].sum(axis=1).round(2)
        self.resource_free_df["total_memory"] = self.resource_free_df[mem_cols].sum(axis=1).round(2)
        self.resource_free_df["elapsed_seconds"] = (
                self.resource_free_df.index - self.resource_free_df.index[0]).total_seconds()

    def fragmentation_index(self):
        cpu_ref = self.timeline_df["cpu_req"].mean()
        mem_ref = self.timeline_df["mem_req"].mean()
        print(f"cpu_ref: {cpu_ref}, mem_ref: {mem_ref}")
        node_pattern = re.compile(r"node_(.*?)_(cpu|memory)")
        nodes = {}
        for col in self.resource_free_df.columns:
            m = node_pattern.fullmatch(col)
            if m:
                nid, resource = m.groups()
                nodes.setdefault(nid, {})[resource] = col  # {'tg96x': {'cpu':'node_tg96x_cpu', ...}, ...}
        for nid, cols in nodes.items():
            cpu_cap = (self.resource_free_df[cols["cpu"]] // cpu_ref).astype(int)
            mem_cap = (self.resource_free_df[cols["memory"]] // mem_ref).astype(int)
            self.resource_free_df[f"allocation_index_node_{nid}"] = np.minimum(cpu_cap, mem_cap)

        alloc_cols = [c for c in self.resource_free_df.columns if c.startswith("allocation_index_node_")]
        self.resource_free_df["max_allocation_index"] = self.resource_free_df[alloc_cols].max(axis=1)
        if isinstance(self.resource_free_df.index, pd.DatetimeIndex):
            dt = self.resource_free_df.index.to_series().diff().dt.total_seconds().fillna(0)
        else:
            dt = self.resource_free_df["elapsed_seconds"].diff().fillna(0)

        y = self.resource_free_df["max_allocation_index"].shift(fill_value=self.resource_free_df["max_allocation_index"].iloc[0])
        self.resource_free_df["alloc_index_integral"] = (y * dt).cumsum()
        total_integral = self.resource_free_df["alloc_index_integral"].iloc[-1].round(2)
        logger.info(f"allocation index integral: {total_integral}")



