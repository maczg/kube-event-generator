# KEG Analyzer Module

The analyzer module provides tools to analyze and compare simulation results from the Kubernetes Event Generator (KEG). It helps evaluate the impact of scheduler plugin weights on various performance metrics.

## Features

- **Multi-Simulation Comparison**: Compare metrics across multiple simulation runs
- **Scheduler Weight Analysis**: Analyze the impact of different scheduler plugin weights
- **Resource Fragmentation Analysis**: Calculate fragmentation indices
- **Comprehensive Metrics**: Evaluate scheduling efficiency, resource utilization, and balance
- **Visualization**: Generate charts and heatmaps for visual analysis
- **Flexible Export**: Export results in CSV, JSON, or HTML formats

## Installation

```bash
# Install Python dependencies
pip install -r analyzer/requirements.txt
```

## Usage

### Using the KEG CLI

```bash
# Compare multiple simulations
keg analyze compare -s sim1 sim2 sim3

# Compare with specific target metric
keg analyze compare -s sim1 sim2 -m fragmentation_index

# Analyze fragmentation for a single simulation
keg analyze fragmentation scenario-100-2025-07-07

# Generate detailed report
keg analyze report scenario-100-2025-07-07
```

### Using Python Scripts Directly

```bash
# Compare simulations
python analyzer/compare_simulations.py \
  --simulations sim1 sim2 sim3 \
  --base-dir results \
  --output analysis_report

# Analyze fragmentation
python analyzer/main.py --data-dir results/sim1
```

## Metrics Analyzed

### Performance Metrics
- **Scheduling Efficiency**: Combined metric of success rate, pending time, and resource utilization
- **Average Pending Time**: Time pods spend waiting to be scheduled
- **Average Running Time**: Pod execution duration
- **Pod Success Rate**: Percentage of successfully scheduled pods

### Resource Metrics
- **CPU Utilization**: Average CPU usage across nodes
- **Memory Utilization**: Average memory usage across nodes
- **Resource Balance Score**: How evenly resources are distributed (inverse coefficient of variation)
- **Fragmentation Index**: Measure of resource fragmentation over time

### Queue Metrics
- **Max Queue Length**: Maximum number of pending pods
- **Queue Length History**: Time series of pending pod count

## Scheduler Weight Analysis

The analyzer evaluates how different scheduler plugin weights affect performance:

1. **NodeResourcesFit**: Scores nodes based on resource availability
2. **NodeAffinity**: Considers node affinity constraints
3. **PodTopologySpread**: Ensures even pod distribution

### Correlation Analysis
The tool calculates Pearson correlations between weights and metrics to identify:
- Which weights have the strongest impact on performance
- Optimal weight configurations for specific goals
- Trade-offs between different metrics

## Output Files

### Comparison Report
- `simulation_comparison.csv`: Raw comparison data
- `weight_impact_*.csv`: Per-plugin correlation analysis
- `weight_impact_heatmap.png`: Visual correlation matrix
- `weight_*_analysis.png`: Scatter plots for each weight
- `analysis_summary.txt`: Text summary with recommendations

### Fragmentation Analysis
- Allocation index integral calculation
- Resource availability over time
- Node-level fragmentation metrics

## Example Workflow

1. **Run Multiple Simulations** with different scheduler weights:
```bash
# Simulation 1: Default weights
keg simulation start -s scenario1.yaml

# Simulation 2: High NodeResourcesFit weight
keg simulation start -s scenario2.yaml

# Simulation 3: Balanced weights
keg simulation start -s scenario3.yaml
```

2. **Compare Results**:
```bash
keg analyze compare -s scenario1-* scenario2-* scenario3-*
```

3. **Review Analysis**:
- Check `analysis_report/analysis_summary.txt` for recommendations
- Review visualizations for patterns
- Examine correlation data for insights

## Extending the Analyzer

To add new metrics or analysis features:

1. **Add Metric Calculation** in `SchedulerWeightAnalyzer._calculate_simulation_metrics()`
2. **Update SimulationMetrics** dataclass with new fields
3. **Include in Comparison** by updating `create_comparison_dataframe()`
4. **Add Visualizations** in `_generate_visualizations()`

## Troubleshooting

### Common Issues

1. **"No valid simulations found"**
   - Ensure simulation directories contain required CSV files
   - Check that base directory path is correct

2. **"Required Python packages not installed"**
   - Run: `pip install -r analyzer/requirements.txt`

3. **"Python3 not found"**
   - Ensure Python 3.x is installed and in PATH

### Debug Mode

For detailed logs:
```bash
export LOG_LEVEL=DEBUG
keg analyze compare -s sim1 sim2
```