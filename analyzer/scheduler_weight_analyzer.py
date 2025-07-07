import os
import json
import logging
import pandas as pd
import numpy as np
from typing import Dict, List, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime
import matplotlib.pyplot as plt
import seaborn as sns
from scipy import stats

from analyzer import Analyzer
from util import _exist_or_create_dir

logger = logging.getLogger(__name__)


@dataclass
class SimulationMetrics:
    """Metrics extracted from a single simulation run."""
    run_id: str
    weights: Dict[str, int]  # Scheduler plugin weights
    fragmentation_index: float
    avg_pending_time: float
    avg_running_time: float
    max_queue_length: int
    cpu_utilization: float
    memory_utilization: float
    pod_success_rate: float
    resource_balance_score: float  # How evenly resources are distributed
    scheduling_efficiency: float  # Combined metric


class SchedulerWeightAnalyzer:
    """Analyzes the impact of scheduler plugin weights on simulation outcomes."""
    
    def __init__(self, base_results_dir: str = "results"):
        self.base_results_dir = base_results_dir
        self.simulations: List[SimulationMetrics] = []
        self.comparison_df: Optional[pd.DataFrame] = None
        
    def load_simulation_results(self, simulation_dirs: List[str]) -> None:
        """Load results from multiple simulation directories."""
        for sim_dir in simulation_dirs:
            try:
                full_path = os.path.join(self.base_results_dir, sim_dir)
                if not os.path.exists(full_path):
                    logger.warning(f"Simulation directory not found: {full_path}")
                    continue
                    
                # Load scheduler weights from scenario or config
                weights = self._extract_scheduler_weights(full_path)
                
                # Use existing Analyzer to load metrics
                analyzer = Analyzer(full_path)
                
                # Calculate derived metrics
                metrics = self._calculate_simulation_metrics(analyzer, sim_dir, weights)
                self.simulations.append(metrics)
                
            except Exception as e:
                logger.error(f"Error loading simulation {sim_dir}: {str(e)}")
                
    def _extract_scheduler_weights(self, sim_dir: str) -> Dict[str, int]:
        """Extract scheduler weights from simulation configuration."""
        # Look for scenario file or configuration
        scenario_files = [
            os.path.join(sim_dir, "scenario.yaml"),
            os.path.join(sim_dir, "config.json"),
            os.path.join(sim_dir, "scheduler_config.json")
        ]
        
        weights = {}
        for file_path in scenario_files:
            if os.path.exists(file_path):
                try:
                    with open(file_path, 'r') as f:
                        if file_path.endswith('.json'):
                            data = json.load(f)
                        else:
                            import yaml
                            data = yaml.safe_load(f)
                    
                    # Extract weights based on file structure
                    # This needs to be adapted based on actual file format
                    if 'scheduler' in data and 'weights' in data['scheduler']:
                        weights = data['scheduler']['weights']
                    elif 'events' in data and 'scheduler' in data['events']:
                        # Extract from scheduler events
                        for event in data['events']['scheduler']:
                            if 'weights' in event:
                                weights.update(event['weights'])
                                
                except Exception as e:
                    logger.warning(f"Could not parse {file_path}: {str(e)}")
        
        # Default weights if not found
        if not weights:
            weights = {
                "NodeResourceFit": 1,
                "NodeAffinity": 1,
                "PodTopologySpread": 1
            }
            
        return weights
        
    def _calculate_simulation_metrics(self, analyzer: Analyzer, run_id: str, 
                                    weights: Dict[str, int]) -> SimulationMetrics:
        """Calculate comprehensive metrics from analyzer data."""
        # Basic metrics
        avg_pending = analyzer.pod_pending_duration_df['duration'].mean()
        avg_running = analyzer.pod_running_times_df['duration'].mean()
        max_queue = analyzer.pod_queue_length_df['queue_length'].max()
        
        # Resource utilization
        cpu_util = analyzer.resource_usage_ratio_df.filter(like='cpu').mean().mean()
        mem_util = analyzer.resource_usage_ratio_df.filter(like='memory').mean().mean()
        
        # Success rate
        total_pods = len(analyzer.timeline_df[analyzer.timeline_df['event'] == 'pod_add'])
        successful_pods = len(analyzer.timeline_df[analyzer.timeline_df['event'] == 'pod_running'])
        success_rate = successful_pods / total_pods if total_pods > 0 else 0
        
        # Resource balance score (coefficient of variation)
        cpu_balance = self._calculate_resource_balance(analyzer.resource_usage_ratio_df, 'cpu')
        mem_balance = self._calculate_resource_balance(analyzer.resource_usage_ratio_df, 'memory')
        balance_score = (cpu_balance + mem_balance) / 2
        
        # Fragmentation index
        analyzer.fragmentation_index()
        frag_index = analyzer.resource_free_df.get('alloc_index_integral', [0])[-1]
        
        # Combined efficiency score
        efficiency = self._calculate_scheduling_efficiency(
            success_rate, avg_pending, cpu_util, mem_util, balance_score
        )
        
        return SimulationMetrics(
            run_id=run_id,
            weights=weights,
            fragmentation_index=frag_index,
            avg_pending_time=avg_pending,
            avg_running_time=avg_running,
            max_queue_length=max_queue,
            cpu_utilization=cpu_util,
            memory_utilization=mem_util,
            pod_success_rate=success_rate,
            resource_balance_score=balance_score,
            scheduling_efficiency=efficiency
        )
        
    def _calculate_resource_balance(self, df: pd.DataFrame, resource_type: str) -> float:
        """Calculate how evenly resources are distributed across nodes."""
        resource_cols = [col for col in df.columns if resource_type in col.lower()]
        if not resource_cols:
            return 0.0
            
        # Calculate coefficient of variation for each timestamp
        cv_values = []
        for _, row in df.iterrows():
            values = row[resource_cols].values
            if len(values) > 1 and values.mean() > 0:
                cv = stats.variation(values)
                cv_values.append(cv)
                
        # Return inverse of average CV (higher score = better balance)
        avg_cv = np.mean(cv_values) if cv_values else 1.0
        return 1.0 / (1.0 + avg_cv)
        
    def _calculate_scheduling_efficiency(self, success_rate: float, avg_pending: float,
                                       cpu_util: float, mem_util: float, 
                                       balance_score: float) -> float:
        """Calculate overall scheduling efficiency metric."""
        # Normalize pending time (lower is better)
        pending_score = 1.0 / (1.0 + avg_pending / 60.0)  # Normalize to minutes
        
        # Combine metrics with weights
        efficiency = (
            0.3 * success_rate +
            0.2 * pending_score +
            0.2 * cpu_util / 100.0 +
            0.2 * mem_util / 100.0 +
            0.1 * balance_score
        )
        
        return efficiency
        
    def create_comparison_dataframe(self) -> pd.DataFrame:
        """Create a DataFrame for easy comparison of simulations."""
        if not self.simulations:
            raise ValueError("No simulations loaded")
            
        data = []
        for sim in self.simulations:
            row = {
                'run_id': sim.run_id,
                'fragmentation_index': sim.fragmentation_index,
                'avg_pending_time': sim.avg_pending_time,
                'avg_running_time': sim.avg_running_time,
                'max_queue_length': sim.max_queue_length,
                'cpu_utilization': sim.cpu_utilization,
                'memory_utilization': sim.memory_utilization,
                'pod_success_rate': sim.pod_success_rate,
                'resource_balance_score': sim.resource_balance_score,
                'scheduling_efficiency': sim.scheduling_efficiency
            }
            
            # Add weight columns
            for plugin, weight in sim.weights.items():
                row[f'weight_{plugin}'] = weight
                
            data.append(row)
            
        self.comparison_df = pd.DataFrame(data)
        return self.comparison_df
        
    def analyze_weight_impact(self) -> Dict[str, pd.DataFrame]:
        """Analyze the impact of each weight on metrics."""
        if self.comparison_df is None:
            self.create_comparison_dataframe()
            
        weight_cols = [col for col in self.comparison_df.columns if col.startswith('weight_')]
        metric_cols = ['fragmentation_index', 'avg_pending_time', 'cpu_utilization',
                      'memory_utilization', 'resource_balance_score', 'scheduling_efficiency']
        
        correlations = {}
        for weight_col in weight_cols:
            plugin_name = weight_col.replace('weight_', '')
            
            # Calculate correlations with metrics
            corr_data = []
            for metric in metric_cols:
                corr, p_value = stats.pearsonr(
                    self.comparison_df[weight_col],
                    self.comparison_df[metric]
                )
                corr_data.append({
                    'metric': metric,
                    'correlation': corr,
                    'p_value': p_value,
                    'significant': p_value < 0.05
                })
                
            correlations[plugin_name] = pd.DataFrame(corr_data)
            
        return correlations
        
    def find_optimal_weights(self, target_metric: str = 'scheduling_efficiency') -> Dict[str, int]:
        """Find the weight configuration that optimizes the target metric."""
        if self.comparison_df is None:
            self.create_comparison_dataframe()
            
        # Find simulation with best target metric
        best_idx = self.comparison_df[target_metric].idxmax()
        best_sim = self.simulations[best_idx]
        
        return best_sim.weights
        
    def generate_report(self, output_dir: str = "analysis_report") -> None:
        """Generate comprehensive analysis report."""
        _exist_or_create_dir(output_dir)
        
        # Create comparison DataFrame if not exists
        if self.comparison_df is None:
            self.create_comparison_dataframe()
            
        # 1. Save comparison data
        self.comparison_df.to_csv(os.path.join(output_dir, "simulation_comparison.csv"), index=False)
        
        # 2. Analyze weight impacts
        weight_impacts = self.analyze_weight_impact()
        for plugin, impact_df in weight_impacts.items():
            impact_df.to_csv(os.path.join(output_dir, f"weight_impact_{plugin}.csv"), index=False)
            
        # 3. Generate visualizations
        self._generate_visualizations(output_dir)
        
        # 4. Write summary report
        self._write_summary_report(output_dir, weight_impacts)
        
    def _generate_visualizations(self, output_dir: str) -> None:
        """Generate visualization plots."""
        # 1. Heatmap of weight impacts
        weight_cols = [col for col in self.comparison_df.columns if col.startswith('weight_')]
        metric_cols = ['fragmentation_index', 'avg_pending_time', 'scheduling_efficiency']
        
        if weight_cols and metric_cols:
            corr_matrix = self.comparison_df[weight_cols + metric_cols].corr()
            
            plt.figure(figsize=(10, 8))
            sns.heatmap(corr_matrix.loc[weight_cols, metric_cols], 
                       annot=True, cmap='coolwarm', center=0)
            plt.title('Correlation between Scheduler Weights and Metrics')
            plt.tight_layout()
            plt.savefig(os.path.join(output_dir, 'weight_impact_heatmap.png'))
            plt.close()
            
        # 2. Scatter plots for key relationships
        for weight_col in weight_cols[:3]:  # Top 3 weights
            fig, axes = plt.subplots(2, 2, figsize=(12, 10))
            axes = axes.ravel()
            
            metrics_to_plot = ['scheduling_efficiency', 'fragmentation_index', 
                             'avg_pending_time', 'resource_balance_score']
            
            for idx, metric in enumerate(metrics_to_plot):
                if idx < len(axes):
                    axes[idx].scatter(self.comparison_df[weight_col], 
                                    self.comparison_df[metric])
                    axes[idx].set_xlabel(weight_col)
                    axes[idx].set_ylabel(metric)
                    axes[idx].set_title(f'{weight_col} vs {metric}')
                    
            plt.tight_layout()
            plt.savefig(os.path.join(output_dir, f'{weight_col}_analysis.png'))
            plt.close()
            
    def _write_summary_report(self, output_dir: str, weight_impacts: Dict[str, pd.DataFrame]) -> None:
        """Write a text summary of the analysis."""
        report_path = os.path.join(output_dir, "analysis_summary.txt")
        
        with open(report_path, 'w') as f:
            f.write("Scheduler Weight Analysis Report\n")
            f.write("=" * 50 + "\n\n")
            
            f.write(f"Total Simulations Analyzed: {len(self.simulations)}\n\n")
            
            # Best performing configurations
            f.write("Best Performing Configurations:\n")
            f.write("-" * 30 + "\n")
            
            for metric in ['scheduling_efficiency', 'fragmentation_index', 'avg_pending_time']:
                if metric == 'avg_pending_time':
                    best_idx = self.comparison_df[metric].idxmin()
                else:
                    best_idx = self.comparison_df[metric].idxmax()
                    
                best_sim = self.simulations[best_idx]
                f.write(f"\nBest for {metric}:\n")
                f.write(f"  Run ID: {best_sim.run_id}\n")
                f.write(f"  Value: {getattr(best_sim, metric):.4f}\n")
                f.write(f"  Weights: {best_sim.weights}\n")
                
            # Weight impact summary
            f.write("\n\nWeight Impact Summary:\n")
            f.write("-" * 30 + "\n")
            
            for plugin, impact_df in weight_impacts.items():
                f.write(f"\n{plugin}:\n")
                significant = impact_df[impact_df['significant'] == True]
                if not significant.empty:
                    for _, row in significant.iterrows():
                        f.write(f"  - {row['metric']}: correlation = {row['correlation']:.3f} (p={row['p_value']:.3f})\n")
                else:
                    f.write("  - No significant correlations found\n")
                    
            # Recommendations
            f.write("\n\nRecommendations:\n")
            f.write("-" * 30 + "\n")
            
            optimal_weights = self.find_optimal_weights()
            f.write(f"Optimal weights for scheduling efficiency: {optimal_weights}\n")