import glob
import logging
import os
from datetime import datetime

import numpy as np
from matplotlib import pyplot as plt
from scipy.stats import variation

import pandas as pd

SCALE_MEMORY_FACTOR = (1 / 1024 / 1024 / 1024)  # Scale memory from bytes to GB
SCALE_RATIO_FACTOR = 100  # Scale ratio from 0-1 to 0-100

logger = logging.getLogger(__name__)


def scale_column(df, col, factor):
    """
    Multiplies the given column by 'factor'.
    """
    df[col] = df[[col]] * factor


def parse_cpu(cpu_str: str) -> float:
    """
    Parse a CPU string like '421m' into a float (e.g., 421).
    If the string has no 'm' suffix, interpret it directly as a float.
    """
    if isinstance(cpu_str, str) and cpu_str.endswith('m'):
        return float(cpu_str.replace('m', ''))
    else:
        return float(cpu_str)


def parse_mem(mem_str: str) -> float:
    """
    Parse a memory string like '1044Mi' into a float (e.g., 1044).
    If the string has no 'Mi' suffix, interpret it directly as a float.
    """
    if isinstance(mem_str, str) and 'Mi' in mem_str:
        return float(mem_str.replace('Mi', ''))
    else:
        return float(mem_str)


def _exist_or_create_dir(dir_path):
    """
    Check if the directory exists, if not, create it.
    """
    if not os.path.exists(dir_path):
        os.makedirs(dir_path)


class Analyzer:
    def __init__(self, run_dir):
        self.run_dir = run_dir
        self.run_name = run_dir.split("/")[-1]
        self.data_dir = os.path.join(run_dir, "data")
        self.output_dir = os.path.join(run_dir, "report")
        self.free_df = None
        self.total_df = None
        self.ratio_df = None
        self.usage_df = None
        self.pod_pending_df = None
        self.pod_queue_df = None
        self.pod_running_df = None
        self.fragmentation_df = None
        self.timeline_df = None
        self.report = {}

    def make_report(self):
        report_dict = {
            "run_dir": self.run_dir,
            "run_name": self.run_name,
            "fragmentation": self.calculate_fragmentation_indexes(),
            "avg_pod_fit": self.calculate_avg_pod_fit(),
        }
        self.report.update(report_dict)
        _exist_or_create_dir(self.output_dir)
        import json
        with open(os.path.join(self.output_dir, "report.json"), 'w') as fp:
            json.dump(self.report, fp, indent=4)

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

        self.ratio_df = pd.concat(ratio_dfs, axis=1) if ratio_dfs else pd.DataFrame()
        self.usage_df = pd.concat(usage_dfs, axis=1) if usage_dfs else pd.DataFrame()
        self.free_df = pd.concat(free_dfs, axis=1) if free_dfs else pd.DataFrame()

        self.ratio_df.ffill(inplace=True)
        self.usage_df.ffill(inplace=True)
        self.free_df.ffill(inplace=True)

        for col in self.usage_df.columns:
            if "memory" in col:
                scale_column(self.usage_df, col, SCALE_MEMORY_FACTOR)

        for col in self.free_df.columns:
            if "memory" in col:
                scale_column(self.free_df, col, SCALE_MEMORY_FACTOR)

        for col in self.ratio_df.columns:
            scale_column(self.ratio_df, col, SCALE_RATIO_FACTOR)

        self.usage_df = self.usage_df.round(2)
        self.free_df = self.free_df.round(2)
        self.ratio_df = self.ratio_df.round(2)
        self.pod_pending_df = pd.read_csv(os.path.join(self.data_dir, 'pod_pending_durations.csv'))
        self.pod_queue_df = pd.read_csv(os.path.join(self.data_dir, 'pod_queue_length.csv'))
        self.pod_running_df = pd.read_csv(os.path.join(self.data_dir, 'pod_running_durations.csv'))
        self.pod_queue_df['timestamp'] = pd.to_datetime(self.pod_queue_df['timestamp'])
        self.timeline_df = pd.read_csv(os.path.join(self.data_dir, 'event_timeline.csv'))
        self.timeline_df['timestamp'] = pd.to_datetime(self.timeline_df['timestamp'])
        self.timeline_df.set_index("timestamp", inplace=True)
        self.timeline_df = self.timeline_df[~self.timeline_df.index.duplicated(keep='first')]

        cpu_cols = [col for col in self.usage_df.columns if "cpu" in col.lower()]
        mem_cols = [col for col in self.usage_df.columns if "mem" in col.lower()]

        self.usage_df["total_cpu"] = self.usage_df[cpu_cols].sum(axis=1).round(2)
        self.usage_df["total_memory"] = self.usage_df[mem_cols].sum(axis=1).round(2)
        self.usage_df["elapsed_seconds"] = (self.usage_df.index - self.usage_df.index[0]).total_seconds()

        self.free_df["total_cpu"] = self.free_df[cpu_cols].sum(axis=1).round(2)
        self.free_df["total_memory"] = self.free_df[mem_cols].sum(axis=1).round(2)
        self.free_df["elapsed_seconds"] = (self.free_df.index - self.free_df.index[0]).total_seconds()

    def plot_timeline(self, mdates=None):
        if self.timeline_df is None:
            raise ValueError("Timeline data is not loaded. Please load the data first.")

        # Import matplotlib.dates if not provided
        if mdates is None:
            import matplotlib.dates as mdates
        import matplotlib.patches as mpatches

        df = self.timeline_df.reset_index().copy()
        df['timestamp'] = pd.to_datetime(df['timestamp'])

        # Filter only ADDED and DELETED events
        df = df[df['event_type'].isin(['ADDED', 'DELETED'])].copy()
        df['pod_name'] = df['pod'].str.extract(r'([a-zA-Z0-9-]+)')
        t0 = df['timestamp'].min()
        df['time_seconds'] = (df['timestamp'] - t0).dt.total_seconds()
        df['order'] = range(len(df))

        unique_pods = df['pod_name'].unique()
        cmap = plt.get_cmap('tab20')
        colors = {pod: cmap(i % cmap.N) for i, pod in enumerate(unique_pods)}

        # Create the bar chart
        fig, ax = plt.subplots(figsize=(12, 4))
        bar_width = 1.5  # fixed width for each bar in seconds; adjust based on your data resolution

        # For each pod, plot its ADDED and DELETED events using the same color.
        for pod in unique_pods:
            pod_df = df[df['pod_name'] == pod]
            # ADDED events: plot bars that extend upward (height = +1)
            added = pod_df[pod_df['event_type'] == 'ADDED']
            if not added.empty:
                ax.bar(added['time_seconds'], height=1, width=bar_width, color=colors[pod], align='center')
            # DELETED events: plot bars that extend downward (height = -1)
            deleted = pod_df[pod_df['event_type'] == 'DELETED']
            if not deleted.empty:
                ax.bar(deleted['time_seconds'], height=-1, width=bar_width, color=colors[pod], align='center')

        # Draw a horizontal reference line at y=0.
        ax.axhline(0, color='gray', linestyle='--', linewidth=0.5)

        # Label and format the axes
        ax.set_xlabel('Duration (seconds)')
        ax.set_ylabel('Event Type')
        ax.set_title('Pod Event Timeline')
        ax.set_ylim(-1.5, 1.5)
        ax.set_yticks([-1, 0, 1])
        ax.set_yticklabels(['DELETED', '', 'ADDED'])

        # Create a custom legend mapping pod names to their color.
        legend_handles = [mpatches.Patch(color=colors[pod], label=pod) for pod in unique_pods]
        ax.legend(handles=legend_handles, title="Pod Name", bbox_to_anchor=(1.05, 1), loc='upper left')

        plt.tight_layout()
        _exist_or_create_dir(self.output_dir)
        output_filepath = os.path.join(self.output_dir, "pod_event_timeline.png")
        plt.savefig(output_filepath, dpi=300)
        plt.close(fig)

    def save_data(self, output_dir=None):
        if output_dir is None:
            output_dir = self.output_dir
        if not os.path.exists(output_dir):
            os.makedirs(output_dir)
        self.usage_df.to_csv(os.path.join(output_dir, "resource_usage.csv"))
        self.ratio_df.to_csv(os.path.join(output_dir, "resource_usage_ratios.csv"))
        self.free_df.to_csv(os.path.join(output_dir, "resource_free.csv"))
        self.pod_pending_df.to_csv(os.path.join(output_dir, "pod_pending_durations.csv"))
        self.pod_queue_df.to_csv(os.path.join(output_dir, "pod_queue_length.csv"))
        self.pod_running_df.to_csv(os.path.join(output_dir, "pod_running_durations.csv"))

    def calculate_avg_pod_fit(self) -> float:
        """
        1) Reads the pod timeline CSV and parses each unique pod's CPU/memory requests.
        2) Reads the cluster resource free CSV to find total free CPU/memory (last row).
        3) Returns how many times the average pod fits into the cluster's total free resources.
           fitCount = min( (clusterCPU / avgPodCPU), (clusterMem / avgPodMem) ).

        Parameters:
        -----------
        timeline_csv : str
            Path to the CSV with columns: [timestamp, pod, node, status, event_type, request_cpu, request_memory, value].
        resource_free_csv : str
            Path to the CSV with columns: [timestamp, node_0_cpu, node_0_memory, node_1_cpu, ..., total_cpu, total_mem, ...].

        Returns:
        --------
        float
            The number of average pods that fit in the cluster resources.
        """
        # 1) Read and parse the timeline CSV
        timeline_df = self.timeline_df.reset_index().copy()
        # Filter for ADDED or the first time each pod appears if 'ADDED' isn't consistent
        # For safety, we'll just group by pod and take the first row.
        # If you specifically want only 'ADDED', uncomment the line below:
        # timeline_df = timeline_df[timeline_df['event_type'] == 'ADDED'].copy()

        # In case the CSV logs the same pod repeatedly, we'll deduplicate by the first appearance.
        # That ensures each pod is counted once in the average.
        timeline_df.sort_values("timestamp", inplace=True)
        timeline_df = timeline_df.drop_duplicates(subset=["pod"], keep="first")

        # Extract CPU/memory requests, converting from e.g. "421m" and "1044Mi" to floats
        timeline_df['request_cpu_m'] = timeline_df['request_cpu'].apply(parse_cpu)
        timeline_df['request_mem_mi'] = timeline_df['request_memory'].apply(parse_mem)

        # 2) Compute average CPU and Memory across pods
        avg_cpu_m = timeline_df['request_cpu_m'].mean()  # e.g. "m" (milliCPU) units
        avg_mem_mi = timeline_df['request_mem_mi'].mean()  # in Mi

        if pd.isna(avg_cpu_m) or pd.isna(avg_mem_mi):
            raise ValueError("No valid CPU or Memory data found in the timeline CSV. Check your data.")

        # 3) Read the resource free CSV and get the last row's total free CPU/memory
        free_df = self.free_df.reset_index().copy()
        # We assume the last row in the CSV is the final resource state:
        last_row = free_df.iloc[-1]
        cluster_cpu_m = float(last_row['total_cpu'])  # in milliCPU if your data is consistent
        cluster_mem_mi = float(last_row['total_memory'])  # in Mi, if consistent

        # 4) Compute how many times the average pod (CPU & Memory) fits in total cluster resources
        # Use the limiting resource (CPU or Memory).
        fit_cpu = cluster_cpu_m / avg_cpu_m
        fit_mem = cluster_mem_mi / avg_mem_mi
        fit_count = min(fit_cpu, fit_mem)

        # Print and return the result
        print(f"Average pod CPU (m): {avg_cpu_m:.2f}")
        print(f"Average pod Mem (Mi): {avg_mem_mi:.2f}")
        print(f"Cluster CPU Capacity (m): {cluster_cpu_m:.2f}")
        print(f"Cluster Memory Capacity (Mi): {cluster_mem_mi:.2f}")
        print(f"Number of average pods that fit in the cluster = {fit_count:.2f}")
        return float(fit_count)

    def calculate_fragmentation_indexes(self) -> dict:
        """
        Calculate cluster resource fragmentation index based on node resource allocations.
        The fragmentation index is calculated using coefficient of variation (CV) for both CPU and memory
        :return: dict containing fragmentation indexes
        """
        if self.fragmentation_df is None:
            self.fragmentation_df = self.calculate_fragmentation()

        df = self.fragmentation_df.copy()
        cpu_mean = df["cpu_fragmentation"].mean()
        mem_mean = df["memory_fragmentation"].mean()
        comb_mean = df["combined_fragmentation"].mean()

        cpu_std = df["cpu_fragmentation"].std()
        mem_std = df["memory_fragmentation"].std()
        comb_std = df["combined_fragmentation"].std()

        def safe_cv(std_val, mean_val):
            return float(std_val / mean_val) if mean_val > 0 else 0

        indexes = {
            "cpu_fragmentation_mean": float(cpu_mean),
            "memory_fragmentation_mean": float(mem_mean),
            "combined_fragmentation_mean": float(comb_mean),

            "cpu_fragmentation_max": float(df["cpu_fragmentation"].max()),
            "memory_fragmentation_max": float(df["memory_fragmentation"].max()),
            "combined_fragmentation_max": float(df["combined_fragmentation"].max()),

            "cpu_fragmentation_std": float(cpu_std),
            "memory_fragmentation_std": float(mem_std),
            "combined_fragmentation_std": float(comb_std),

            "cpu_fragmentation_cv": safe_cv(cpu_std, cpu_mean),
            "memory_fragmentation_cv": safe_cv(mem_std, mem_mean),
            "combined_fragmentation_cv": safe_cv(comb_std, comb_mean),

            # Integrate over the index for each fragmentation series
            "cpu_fragmentation_auc": float(np.trapezoid(df["cpu_fragmentation"],
                                                        df.index)),
            "memory_fragmentation_auc": float(np.trapezoid(df["memory_fragmentation"],
                                                           df.index)),
            "combined_fragmentation_auc": float(np.trapezoid(df["combined_fragmentation"],
                                                             df.index))
        }
        return indexes

    def calculate_fragmentation(self):
        """
        Calculate cluster resource fragmentation index based on node resource allocations.
        The fragmentation index is calculated using coefficient of variation (CV) for both CPU and memory
        resources across nodes. A higher value indicates more fragmentation (uneven distribution).

        Returns: DataFrame timeseries of fragmentation indices
        --------
        DataFrame containing timestamp and fragmentation indices
        """
        # Initialize lists to store results
        timestamps = []
        cpu_fragmentation = []
        memory_fragmentation = []
        combined_fragmentation = []

        if self.usage_df is None:
            raise ValueError("Usage data is not loaded. Please load the data first.")

        df = self.usage_df.copy()

        # Process each timestamp
        for timestamp, group in df.groupby('timestamp'):
            # Extract CPU and memory columns
            cpu_cols = [col for col in group.columns if ('cpu' in col.lower() and 'total' not in col.lower())]
            mem_cols = [col for col in group.columns if ('memory' in col.lower() and 'total' not in col.lower())]

            # Get CPU and memory values
            cpu_values = group[cpu_cols].values.flatten()
            mem_values = group[mem_cols].values.flatten()
            cpu_values = cpu_values[cpu_values > 0]
            mem_values = mem_values[mem_values > 0]

            # Calculate coefficient of variation (standard deviation / mean)
            # This measures the relative variability and is a good indicator of fragmentation
            # If all nodes have similar usage, CV will be low (less fragmentation)
            # If usage is highly variable across nodes, CV will be high (more fragmentation)
            cpu_frag = variation(cpu_values) if len(cpu_values) > 1 else 0
            mem_frag = variation(mem_values) if len(mem_values) > 1 else 0
            # Calculate combined fragmentation (weighted average of CPU and memory fragmentation)
            # We can adjust weights based on which resource is more critical
            combined_frag = (cpu_frag + mem_frag) / 2

            timestamps.append(timestamp)
            cpu_fragmentation.append(cpu_frag)
            memory_fragmentation.append(mem_frag)
            combined_fragmentation.append(combined_frag)

        fragmentation_df = pd.DataFrame({
            'timestamp': timestamps,
            'cpu_fragmentation': cpu_fragmentation,
            'memory_fragmentation': memory_fragmentation,
            'combined_fragmentation': combined_fragmentation
        })
        return fragmentation_df.set_index("timestamp")


def compare_fragmentation(run_dirs, plot_dir="./results/comparison"):
    """
    Compare fragmentation metrics across multiple experiment runs.

    Parameters:
    -----------
    run_dirs : list of str
        List of directories containing experiment run data

    Returns:
    --------
    tuple: (summary_df, figures)
        - summary_df: DataFrame with comparison statistics
        - figures: dictionary of matplotlib figures for visualization
    """
    metrics = ['cpu_fragmentation', 'memory_fragmentation', 'combined_fragmentation']
    runs_fragmentation = {}

    for run_dir in run_dirs:
        analyzer = Analyzer(run_dir)
        analyzer.load_data()
        run_name = run_dir.split("/")[-1]
        runs_fragmentation[run_name] = analyzer.calculate_fragmentation()

    summary = {}
    for metric in metrics:
        for run_name, strategy_df in runs_fragmentation.items():
            strategy_df[metric] = strategy_df[metric].astype(float)
            strategy_df[metric] = strategy_df[metric].round(2)

            # Calculate statistics
            summary[f"{run_name}_{metric}_mean"] = strategy_df[metric].mean()
            summary[f"{run_name}_{metric}_max"] = strategy_df[metric].max()
            summary[f"{run_name}_{metric}_std"] = strategy_df[metric].std()
            summary[f"{run_name}_{metric}_cv"] = strategy_df[metric].std() / strategy_df[metric].mean() if strategy_df[
                                                                                                               metric].mean() > 0 else 0
            # Calculate area under curve
            strategy_df['timestamp'] = pd.to_datetime(strategy_df['timestamp'])
            strategy_df['time_seconds'] = (strategy_df['timestamp'] - strategy_df['timestamp'].min()).dt.total_seconds()
            summary[f"{run_name}_{metric}_auc"] = np.trapezoid(strategy_df[metric], strategy_df['time_seconds'])

    # Calculate pairwise comparisons and improvements
    if len(run_dirs) > 1:
        for i, run1 in enumerate(run_dirs):
            run1_name = run1.split("/")[-1]
            for j, run2 in enumerate(run_dirs):
                run2_name = run2.split("/")[-1]
                if i < j:  # Only compare each pair once
                    for metric in metrics:
                        mean1 = summary[f"{run1_name}_{metric}_mean"]
                        mean2 = summary[f"{run2_name}_{metric}_mean"]

                        if mean1 > 0:  # Avoid division by zero
                            improvement = ((mean1 - mean2) / mean1) * 100
                            summary[f"{run1_name}_vs_{run2_name}_{metric}_improvement_percentage"] = improvement

    # Create visualization for each metric
    figures = {}
    for metric in metrics:
        fig, ax = plt.subplots(figsize=(12, 6))

        for run_name, df in runs_fragmentation.items():
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            ax.plot(df['timestamp'], df[metric], label=f"{run_name}", marker='o', alpha=0.7)

        # Add AUC values to legend
        handles, labels = ax.get_legend_handles_labels()
        new_labels = []
        for run_dir, label in zip(run_dirs, labels):
            run_name = run_dir.split("/")[-1]
            auc = summary[f"{run_name}_{metric}_auc"]
            mean = summary[f"{run_name}_{metric}_mean"]
            new_labels.append(f"{label} (Mean: {mean:.2f}, AUC: {auc:.2f})")

        ax.set_title(f"Comparison of {metric.replace('_', ' ').title()}")
        ax.set_xlabel('Time')
        ax.set_ylabel(f"{metric.replace('_', ' ').title()}")
        ax.legend(handles, new_labels)
        ax.grid(True, alpha=0.3)

        if len(run_dirs) == 2:
            run1, run2 = run_dirs
            run1_name = run1.split("/")[-1]
            run2_name = run2.split("/")[-1]
            improvement = summary.get(f"{run1_name}_vs_{run2_name}_{metric}_improvement_percentage", 0)
            better_run = run2_name if improvement > 0 else run1_name
            worse_run = run1_name if improvement > 0 else run2_name
            abs_improvement = abs(improvement)

            annotation = f"{better_run} has {abs_improvement:.2f}% lower {metric} than {worse_run}"
            ax.annotate(annotation, xy=(0.5, 0.01), xycoords='figure fraction',
                        ha='center', va='bottom', bbox=dict(boxstyle="round,pad=0.5",
                                                            fc="lightyellow", ec="orange", alpha=0.8))

        fig.tight_layout()
        figures[metric] = fig

    # Save the figures
    if not os.path.exists(plot_dir):
        os.makedirs(plot_dir)

    for metric, fig in figures.items():
        fig.savefig(os.path.join(plot_dir, f"{metric}.png"), dpi=300)
        plt.close(fig)
    # Create a summary DataFrame
    summary_df = pd.DataFrame([summary])
    summary_df.to_csv(os.path.join(plot_dir, "summary_statistics.csv"), index=False)

    return summary_df, figures


def compare_reports(run_dirs, plot_dir="./results/comparison"):
    reports = []
    for run_dir in run_dirs:
        report_file = f"{run_dir}/report/report.json"
        if not os.path.exists(report_file):
            logger.warning(f"report file {report_file} does not exist, creating")
            analyzer = Analyzer(run_dir)
            try:
                analyzer.load_data()
                analyzer.make_report()
                reports.append(analyzer.report)
            except Exception as e:
                logger.error(f"Failed to load data for {run_dir}, skipping")
            finally:
                continue

        with open(report_file, "r") as f:
            import json
            report = json.load(f)
            reports.append(report)


def main():
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--data-dir", type=str, help="Path to the run directory to analyze")
    parser.add_argument('--compare', nargs='+', help='List of run dir to compare')
    parser.add_argument('--plot', type=bool, default=False, help='Plot the timeline of events')

    args = parser.parse_args()
    # usage python3 analyzer.py ./result/experiment2 ./result/experiment3
    if args.compare:
        # compare_fragmentation(args.compare, "./results/comparison")
        compare_reports(args.compare, "./results/comparison")
        return
    analyzer = Analyzer(args.data_dir)
    analyzer.load_data()
    if args.plot:
        analyzer.plot_timeline()
        return
    # analyzer.calculate_fragmentation_index()

    if args.data_dir:
        analyzer.make_report()


if __name__ == "__main__":
    main()
