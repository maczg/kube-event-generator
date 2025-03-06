import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import os
import argparse


class SchedulerMetricsAnalyzer:
    def __init__(self, data_dir):
        self.data_dir = data_dir
        self.free_df = None
        self.ratios_df = None
        self.usage_df = None
        self.pod_pending_df = None
        self.pod_queue_df = None
        self.pod_running_df = None
        self.load_data()

    def load_data(self):
        """Load all CSV files from the data directory"""
        try:
            self.free_df = pd.read_csv(os.path.join(self.data_dir, 'resource_free.csv'))
            self.ratios_df = pd.read_csv(os.path.join(self.data_dir, 'resource_usage_ratios.csv'))
            self.usage_df = pd.read_csv(os.path.join(self.data_dir, 'resource_usage.csv'))
            self.pod_pending_df = pd.read_csv(os.path.join(self.data_dir, 'pod_pending_durations.csv'))
            self.pod_queue_df = pd.read_csv(os.path.join(self.data_dir, 'pod_queue_length.csv'))
            self.pod_running_df = pd.read_csv(os.path.join(self.data_dir, 'pod_running_durations.csv'))

            # Convert timestamps to datetime objects
            self.free_df['timestamp'] = pd.to_datetime(self.free_df['timestamp'])
            self.ratios_df['timestamp'] = pd.to_datetime(self.ratios_df['timestamp'])
            self.usage_df['timestamp'] = pd.to_datetime(self.usage_df['timestamp'])
            self.pod_queue_df['timestamp'] = pd.to_datetime(self.pod_queue_df['timestamp'])

            print("All data loaded successfully.")
        except Exception as e:
            print(f"Error loading data: {e}")
            raise

    def calculate_resource_fragmentation(self):
        """Calculate resource fragmentation across nodes"""
        # Get the initial capacity for each node
        initial_row = self.free_df.iloc[0]
        node_cpu_cols = [col for col in self.free_df.columns if '_cpu' in col]
        node_mem_cols = [col for col in self.free_df.columns if '_memory' in col]

        # Calculate total capacity
        total_cpu_capacity = sum([initial_row[col] for col in node_cpu_cols])
        total_mem_capacity = sum([initial_row[col] for col in node_mem_cols])

        # Find the timestamp with maximum resource usage
        max_usage_row = self.usage_df.loc[self.usage_df['total_cpu'].idxmax()]
        max_usage_time = max_usage_row['timestamp']

        # Get the corresponding free resources at that time
        closest_free_idx = self.free_df['timestamp'].sub(max_usage_time).abs().idxmin()
        free_at_max_usage = self.free_df.iloc[closest_free_idx]

        # Calculate fragmentation metrics
        node_cpu_usage = {}
        node_mem_usage = {}
        cpu_utilization = []
        mem_utilization = []

        for i, cpu_col in enumerate(node_cpu_cols):
            node_num = cpu_col.split('_')[1]
            mem_col = f"node_{node_num}_memory"

            initial_cpu = initial_row[cpu_col]
            initial_mem = initial_row[mem_col]
            free_cpu = free_at_max_usage[cpu_col]
            free_mem = free_at_max_usage[mem_col]

            used_cpu = initial_cpu - free_cpu
            used_mem = initial_mem - free_mem

            cpu_util = (used_cpu / initial_cpu) * 100 if initial_cpu > 0 else 0
            mem_util = (used_mem / initial_mem) * 100 if initial_mem > 0 else 0

            node_cpu_usage[f"node_{node_num}"] = used_cpu
            node_mem_usage[f"node_{node_num}"] = used_mem

            cpu_utilization.append(cpu_util)
            mem_utilization.append(mem_util)

        # Calculate variance and std deviation
        cpu_util_variance = np.var(cpu_utilization)
        cpu_util_std = np.std(cpu_utilization)
        mem_util_variance = np.var(mem_utilization)
        mem_util_std = np.std(mem_utilization)

        # Imbalance ratio (max/min utilization)
        cpu_imbalance = max(cpu_utilization) / (
                    min(cpu_utilization) + 0.001)  # Add small value to avoid division by zero
        mem_imbalance = max(mem_utilization) / (min(mem_utilization) + 0.001)

        return {
            "node_cpu_usage": node_cpu_usage,
            "node_mem_usage": node_mem_usage,
            "cpu_utilization_percent": cpu_utilization,
            "mem_utilization_percent": mem_utilization,
            "cpu_util_variance": cpu_util_variance,
            "cpu_util_std": cpu_util_std,
            "mem_util_variance": mem_util_variance,
            "mem_util_std": mem_util_std,
            "cpu_imbalance_ratio": cpu_imbalance,
            "mem_imbalance_ratio": mem_imbalance,
            "max_usage_timestamp": max_usage_time
        }

    def calculate_cluster_allocation(self):
        """Calculate allocated CPU/memory across the cluster over time"""
        # Extract peak resource usage
        peak_cpu_usage = self.usage_df['total_cpu'].max()
        peak_cpu_time = self.usage_df.loc[self.usage_df['total_cpu'].idxmax(), 'timestamp']

        peak_mem_usage = self.usage_df['total_mem'].max()
        peak_mem_time = self.usage_df.loc[self.usage_df['total_mem'].idxmax(), 'timestamp']

        # Calculate total capacity
        initial_row = self.free_df.iloc[0]
        node_cpu_cols = [col for col in self.free_df.columns if '_cpu' in col]
        node_mem_cols = [col for col in self.free_df.columns if '_memory' in col]

        total_cpu_capacity = sum([initial_row[col] for col in node_cpu_cols])
        total_mem_capacity = sum([initial_row[col] for col in node_mem_cols])

        # Calculate peak utilization percentages
        peak_cpu_percent = (peak_cpu_usage / total_cpu_capacity) * 100
        peak_mem_percent = (peak_mem_usage / total_mem_capacity) * 100

        # Duration of resource allocation
        start_time = self.usage_df['timestamp'].min()
        end_time = self.usage_df['timestamp'].max()
        total_duration_seconds = (end_time - start_time).total_seconds()

        # Calculate area under the curve (resource-time product)
        # This gives us a measure of total resource consumption over time
        cpu_time_product = np.trapezoid(self.usage_df['total_cpu'].values, self.usage_df['elapsed_seconds'].values)
        mem_time_product = np.trapezoid(self.usage_df['total_mem'].values, self.usage_df['elapsed_seconds'].values)

        # Average resource usage
        avg_cpu_usage = cpu_time_product / total_duration_seconds
        avg_mem_usage = mem_time_product / total_duration_seconds

        return {
            "peak_cpu_usage": peak_cpu_usage,
            "peak_cpu_time": peak_cpu_time,
            "peak_mem_usage": peak_mem_usage,
            "peak_mem_time": peak_mem_time,
            "total_cpu_capacity": total_cpu_capacity,
            "total_mem_capacity": total_mem_capacity,
            "peak_cpu_percent": peak_cpu_percent,
            "peak_mem_percent": peak_mem_percent,
            "total_duration_seconds": total_duration_seconds,
            "avg_cpu_usage": avg_cpu_usage,
            "avg_mem_usage": avg_mem_usage,
            "cpu_time_product": cpu_time_product,
            "mem_time_product": mem_time_product
        }

    def calculate_resource_saturation(self):
        """Calculate resource saturation per node"""
        # Find maximum utilization for each node
        node_nums = set()
        for col in self.usage_df.columns:
            if '_cpu' in col:
                parts = col.split('_')
                if len(parts) >= 2:
                    node_nums.add(parts[1])

        saturation_data = {}

        for node in node_nums:
            cpu_col = f"node_{node}_cpu"
            mem_col = f"node_{node}_memory"

            if cpu_col in self.usage_df.columns and mem_col in self.usage_df.columns:
                max_cpu_usage = self.usage_df[cpu_col].max()
                max_cpu_time = self.usage_df.loc[self.usage_df[cpu_col].idxmax(), 'timestamp']

                max_mem_usage = self.usage_df[mem_col].max()
                max_mem_time = self.usage_df.loc[self.usage_df[mem_col].idxmax(), 'timestamp']

                # Get the initial capacity for this node
                initial_cpu = self.free_df.iloc[0][f"node_{node}_cpu"]
                initial_mem = self.free_df.iloc[0][f"node_{node}_memory"]

                # Calculate saturation percentages
                cpu_saturation = (max_cpu_usage / initial_cpu) * 100 if initial_cpu > 0 else 0
                mem_saturation = (max_mem_usage / initial_mem) * 100 if initial_mem > 0 else 0

                saturation_data[f"node_{node}"] = {
                    "max_cpu_usage": max_cpu_usage,
                    "max_cpu_time": max_cpu_time,
                    "max_mem_usage": max_mem_usage,
                    "max_mem_time": max_mem_time,
                    "initial_cpu": initial_cpu,
                    "initial_mem": initial_mem,
                    "cpu_saturation_percent": cpu_saturation,
                    "mem_saturation_percent": mem_saturation
                }

        return saturation_data

    def calculate_queue_metrics(self):
        """Calculate average pod waiting time and queue length metrics"""
        # Pod waiting time statistics
        avg_wait_time = self.pod_pending_df['pending_time_milliseconds'].mean()
        min_wait_time = self.pod_pending_df['pending_time_milliseconds'].min()
        max_wait_time = self.pod_pending_df['pending_time_milliseconds'].max()
        std_wait_time = self.pod_pending_df['pending_time_milliseconds'].std()

        # Queue length statistics
        avg_queue_length = self.pod_queue_df['length'].mean()
        max_queue_length = self.pod_queue_df['length'].max()
        max_queue_time = self.pod_queue_df.loc[self.pod_queue_df['length'].idxmax(), 'timestamp']

        # Calculate queue duration (time between first pod added and last pod removed)
        queue_start = self.pod_queue_df['timestamp'].min()
        queue_end = self.pod_queue_df['timestamp'].max()
        queue_duration_seconds = (queue_end - queue_start).total_seconds()

        # Calculate area under the queue length curve (queue-time product)
        # This gives us a measure of total waiting in the queue
        queue_length_times = [(t - queue_start).total_seconds() for t in self.pod_queue_df['timestamp']]
        queue_time_product = np.trapezoid(self.pod_queue_df['length'].values, queue_length_times)

        return {
            "avg_wait_time_ms": avg_wait_time,
            "min_wait_time_ms": min_wait_time,
            "max_wait_time_ms": max_wait_time,
            "std_wait_time_ms": std_wait_time,
            "avg_queue_length": avg_queue_length,
            "max_queue_length": max_queue_length,
            "max_queue_time": max_queue_time,
            "queue_duration_seconds": queue_duration_seconds,
            "queue_time_product": queue_time_product
        }

    def calculate_pod_runtime_metrics(self):
        """Calculate pod runtime metrics"""
        avg_runtime = self.pod_running_df['running_time_milliseconds'].mean()
        min_runtime = self.pod_running_df['running_time_milliseconds'].min()
        max_runtime = self.pod_running_df['running_time_milliseconds'].max()
        std_runtime = self.pod_running_df['running_time_milliseconds'].std()

        return {
            "avg_runtime_ms": avg_runtime,
            "min_runtime_ms": min_runtime,
            "max_runtime_ms": max_runtime,
            "std_runtime_ms": std_runtime,
            "pod_count": len(self.pod_running_df)
        }

    def generate_analysis_report(self):
        """Generate a comprehensive analysis report"""
        fragmentation = self.calculate_resource_fragmentation()
        allocation = self.calculate_cluster_allocation()
        saturation = self.calculate_resource_saturation()
        queue_metrics = self.calculate_queue_metrics()
        runtime_metrics = self.calculate_pod_runtime_metrics()

        # Combine all metrics into a single report
        report = {
            "resource_fragmentation": fragmentation,
            "cluster_allocation": allocation,
            "resource_saturation": saturation,
            "queue_metrics": queue_metrics,
            "runtime_metrics": runtime_metrics
        }

        return report

    def print_report(self, report):
        """Print a human-readable summary of the analysis report"""
        print("\n===== KUBERNETES SCHEDULER METRICS ANALYSIS =====\n")

        # Resource Fragmentation
        print("RESOURCE FRAGMENTATION ACROSS NODES")
        print(f"Timestamp of peak usage: {report['resource_fragmentation']['max_usage_timestamp']}")

        print("\nNode CPU Utilization (%):")
        for i, util in enumerate(report['resource_fragmentation']['cpu_utilization_percent']):
            print(f"  Node {i}: {util:.2f}%")

        print(f"\nCPU Utilization Standard Deviation: {report['resource_fragmentation']['cpu_util_std']:.2f}%")
        print(f"CPU Imbalance Ratio (max/min): {report['resource_fragmentation']['cpu_imbalance_ratio']:.2f}")

        print("\nNode Memory Utilization (%):")
        for i, util in enumerate(report['resource_fragmentation']['mem_utilization_percent']):
            print(f"  Node {i}: {util:.2f}%")

        print(f"\nMemory Utilization Standard Deviation: {report['resource_fragmentation']['mem_util_std']:.2f}%")
        print(f"Memory Imbalance Ratio (max/min): {report['resource_fragmentation']['mem_imbalance_ratio']:.2f}")

        # Cluster Allocation
        print("\n\nALLOCATED RESOURCES ACROSS THE CLUSTER")
        print(f"Peak CPU Usage: {report['cluster_allocation']['peak_cpu_usage']:.2f} millicores "
              f"({report['cluster_allocation']['peak_cpu_percent']:.2f}% of total capacity)")
        print(f"Peak Memory Usage: {report['cluster_allocation']['peak_mem_usage']:.2f} MB "
              f"({report['cluster_allocation']['peak_mem_percent']:.2f}% of total capacity)")
        print(f"Average CPU Usage: {report['cluster_allocation']['avg_cpu_usage']:.2f} millicores")
        print(f"Average Memory Usage: {report['cluster_allocation']['avg_mem_usage']:.2f} MB")
        print(f"Total Duration: {report['cluster_allocation']['total_duration_seconds']:.2f} seconds")

        # Resource Saturation
        print("\n\nRESOURCE SATURATION PER NODE")
        for node, data in report['resource_saturation'].items():
            print(f"\n{node.upper()}:")
            print(f"  CPU Saturation: {data['cpu_saturation_percent']:.2f}%")
            print(f"  Memory Saturation: {data['mem_saturation_percent']:.2f}%")

        # Queue Metrics
        print("\n\nPOD QUEUE METRICS")
        print(f"Average Pod Waiting Time: {report['queue_metrics']['avg_wait_time_ms']:.2f} ms")
        print(f"Min/Max Pod Waiting Time: {report['queue_metrics']['min_wait_time_ms']:.2f} ms / "
              f"{report['queue_metrics']['max_wait_time_ms']:.2f} ms")
        print(f"Standard Deviation of Waiting Time: {report['queue_metrics']['std_wait_time_ms']:.2f} ms")
        print(f"Average Queue Length: {report['queue_metrics']['avg_queue_length']:.2f} pods")
        print(f"Maximum Queue Length: {report['queue_metrics']['max_queue_length']} pods "
              f"at {report['queue_metrics']['max_queue_time']}")
        print(f"Queue Duration: {report['queue_metrics']['queue_duration_seconds']:.2f} seconds")

        # Pod Runtime Metrics
        print("\n\nPOD RUNTIME METRICS")
        print(f"Total Pods: {report['runtime_metrics']['pod_count']}")
        print(f"Average Pod Runtime: {report['runtime_metrics']['avg_runtime_ms']:.2f} ms")
        print(f"Min/Max Pod Runtime: {report['runtime_metrics']['min_runtime_ms']:.2f} ms / "
              f"{report['runtime_metrics']['max_runtime_ms']:.2f} ms")
        print(f"Standard Deviation of Runtime: {report['runtime_metrics']['std_runtime_ms']:.2f} ms")

    def generate_visualizations(self, output_dir=None):
        """Generate visualizations of the metrics"""
        if output_dir is None:
            output_dir = "visualizations"

        os.makedirs(output_dir, exist_ok=True)

        # 1. Resource Utilization Over Time
        plt.figure(figsize=(12, 6))
        plt.plot(self.usage_df['timestamp'],
                 self.usage_df['total_cpu'] / self.calculate_cluster_allocation()['total_cpu_capacity'] * 100,
                 label='CPU Utilization (%)')
        plt.plot(self.usage_df['timestamp'],
                 self.usage_df['total_mem'] / self.calculate_cluster_allocation()['total_mem_capacity'] * 100,
                 label='Memory Utilization (%)')
        plt.xlabel('Time')
        plt.ylabel('Utilization (%)')
        plt.title('Cluster Resource Utilization Over Time')
        plt.legend()
        plt.grid(True)
        plt.xticks(rotation=45)
        plt.tight_layout()
        plt.savefig(os.path.join(output_dir, 'resource_utilization.png'))

        # 2. Queue Length Over Time
        plt.figure(figsize=(12, 6))
        plt.plot(self.pod_queue_df['timestamp'], self.pod_queue_df['length'])
        plt.xlabel('Time')
        plt.ylabel('Queue Length')
        plt.title('Pod Queue Length Over Time')
        plt.grid(True)
        plt.xticks(rotation=45)
        plt.tight_layout()
        plt.savefig(os.path.join(output_dir, 'queue_length.png'))

        # 3. Node Utilization at Peak
        report = self.generate_analysis_report()

        plt.figure(figsize=(12, 6))
        nodes = [f"Node {i}" for i in range(len(report['resource_fragmentation']['cpu_utilization_percent']))]
        cpu_utils = report['resource_fragmentation']['cpu_utilization_percent']
        mem_utils = report['resource_fragmentation']['mem_utilization_percent']

        x = np.arange(len(nodes))
        width = 0.35

        fig, ax = plt.subplots(figsize=(12, 6))
        rects1 = ax.bar(x - width / 2, cpu_utils, width, label='CPU')
        rects2 = ax.bar(x + width / 2, mem_utils, width, label='Memory')

        ax.set_ylabel('Utilization (%)')
        ax.set_title('Node Resource Utilization at Peak Usage')
        ax.set_xticks(x)
        ax.set_xticklabels(nodes)
        ax.legend()

        ax.bar_label(rects1, fmt='%.1f')
        ax.bar_label(rects2, fmt='%.1f')

        fig.tight_layout()
        plt.savefig(os.path.join(output_dir, 'node_utilization.png'))

        # 4. Pod Waiting Times
        plt.figure(figsize=(12, 6))
        pod_names = self.pod_pending_df['pod_name']
        waiting_times = self.pod_pending_df['pending_time_milliseconds']

        plt.bar(pod_names, waiting_times)
        plt.xlabel('Pod Name')
        plt.ylabel('Waiting Time (ms)')
        plt.title('Pod Waiting Times')
        plt.xticks(rotation=45)
        plt.grid(True, axis='y')
        plt.tight_layout()
        plt.savefig(os.path.join(output_dir, 'pod_waiting_times.png'))

        print(f"\nVisualizations saved to {output_dir} directory.")

    def export_metrics_for_comparison(self, output_file=None):
        """Export key metrics in a format suitable for comparing different simulations"""
        if output_file is None:
            output_file = "simulation_metrics.csv"

        # Generate the analysis report
        report = self.generate_analysis_report()

        # Extract key metrics for comparison
        metrics = {
            "cpu_utilization_std": report['resource_fragmentation']['cpu_util_std'],
            "mem_utilization_std": report['resource_fragmentation']['mem_util_std'],
            "cpu_imbalance_ratio": report['resource_fragmentation']['cpu_imbalance_ratio'],
            "mem_imbalance_ratio": report['resource_fragmentation']['mem_imbalance_ratio'],
            "peak_cpu_percent": report['cluster_allocation']['peak_cpu_percent'],
            "peak_mem_percent": report['cluster_allocation']['peak_mem_percent'],
            "avg_cpu_usage_percent": report['cluster_allocation']['avg_cpu_usage'] / report['cluster_allocation'][
                'total_cpu_capacity'] * 100,
            "avg_mem_usage_percent": report['cluster_allocation']['avg_mem_usage'] / report['cluster_allocation'][
                'total_mem_capacity'] * 100,
            "max_cpu_saturation": max(
                [data['cpu_saturation_percent'] for data in report['resource_saturation'].values()]),
            "max_mem_saturation": max(
                [data['mem_saturation_percent'] for data in report['resource_saturation'].values()]),
            "avg_wait_time_ms": report['queue_metrics']['avg_wait_time_ms'],
            "max_wait_time_ms": report['queue_metrics']['max_wait_time_ms'],
            "avg_queue_length": report['queue_metrics']['avg_queue_length'],
            "max_queue_length": report['queue_metrics']['max_queue_length'],
            "avg_runtime_ms": report['runtime_metrics']['avg_runtime_ms'],
            "total_duration_seconds": report['cluster_allocation']['total_duration_seconds']
        }

        # Create a DataFrame and save to CSV
        metrics_df = pd.DataFrame([metrics])
        metrics_df.to_csv(output_file, index=False)
        print(f"\nComparison metrics exported to {output_file}")

        return metrics


def compare_simulations(simulation_files, output_file=None):
    """Compare metrics from multiple simulation runs"""
    if output_file is None:
        output_file = "simulation_comparison.csv"

    results = []

    for sim_file in simulation_files:
        if os.path.exists(sim_file):
            df = pd.read_csv(sim_file)
            sim_name = os.path.basename(sim_file).replace('.csv', '')
            df['simulation'] = sim_name
            results.append(df)
        else:
            print(f"Warning: Simulation file not found: {sim_file}")

    if results:
        comparison_df = pd.concat(results)
        comparison_df.to_csv(output_file, index=False)
        print(f"Comparison saved to {output_file}")
        return comparison_df
    else:
        print("No valid simulation files found for comparison.")
        return None


def main():
    parser = argparse.ArgumentParser(description='Analyze Kubernetes scheduler metrics')
    parser.add_argument('--data_dir', type=str, default='.', help='Directory containing the CSV files')
    parser.add_argument('--output_dir', type=str, default='visualizations', help='Directory for output visualizations')
    parser.add_argument('--metrics_file', type=str, default='simulation_metrics.csv',
                        help='Output file for comparison metrics')
    parser.add_argument('--compare', nargs='+', help='List of metrics files to compare')
    parser.add_argument('--comparison_output', type=str, default='simulation_comparison.csv',
                        help='Output file for comparison results')

    args = parser.parse_args()

    if args.compare:
        compare_simulations(args.compare, args.comparison_output)
    else:
        try:
            analyzer = SchedulerMetricsAnalyzer(args.data_dir)
            report = analyzer.generate_analysis_report()
            analyzer.print_report(report)
            analyzer.generate_visualizations(args.output_dir)
            analyzer.export_metrics_for_comparison(args.metrics_file)
        except Exception as e:
            print(f"Error analyzing scheduler metrics: {e}")
            raise


if __name__ == "__main__":
    main()
