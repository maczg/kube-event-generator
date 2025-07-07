# Running the Experiment Model

## Prerequisites

1. **Environment Setup**
   ```bash
   # Ensure KWOK and kube-scheduler-simulator are running
   docker-compose -f docker/docker-compose.yaml up -d
   
   # Set up kubeconfig
   export KUBECONFIG=~/.kube/config:$(pwd)/docker/kubeconfig.local.yaml
   kubectl config view --flatten > ~/.kube/config
   kubectl config use-context simulator
   
   # Install Python dependencies for analysis
   pip install -r analyzer/requirements.txt
   ```

2. **Verify Installation**
   ```bash
   # Check KEG is built
   make build
   
   # Verify analyzer works
   python analyzer/main.py --help
   ```

## Step 1: Define Experiment Parameters

### Create Parameter Configuration File

```yaml
# experiments/experiment_config.yaml
seeds: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

weight_vectors:
  - name: "default"
    weights:
      NodeResourcesFit: 1
      NodeAffinity: 1
      PodTopologySpread: 1
  
  - name: "resource_focused"
    weights:
      NodeResourcesFit: 10
      NodeAffinity: 1
      PodTopologySpread: 1
  
  - name: "affinity_focused"
    weights:
      NodeResourcesFit: 1
      NodeAffinity: 10
      PodTopologySpread: 1
  
  - name: "spread_focused"
    weights:
      NodeResourcesFit: 1
      NodeAffinity: 1
      PodTopologySpread: 10
  
  - name: "balanced_high"
    weights:
      NodeResourcesFit: 5
      NodeAffinity: 5
      PodTopologySpread: 5

parameter_vectors:
  - name: "low_load_small"
    arrival_scale: 5.0      # λ=0.2 (avg 5s between pods)
    duration_shape: 2.0
    duration_scale: 60.0    # avg 60s duration
    pod_cpu_shape: 1.0
    pod_mem_scale: 128.0    # avg 128Mi
  
  - name: "low_load_large"
    arrival_scale: 5.0
    duration_shape: 2.0
    duration_scale: 60.0
    pod_cpu_shape: 2.0
    pod_mem_scale: 512.0
  
  - name: "medium_load_small"
    arrival_scale: 2.0      # λ=0.5 (avg 2s between pods)
    duration_shape: 2.0
    duration_scale: 120.0
    pod_cpu_shape: 1.0
    pod_mem_scale: 128.0
  
  - name: "medium_load_large"
    arrival_scale: 2.0
    duration_shape: 2.0
    duration_scale: 120.0
    pod_cpu_shape: 2.0
    pod_mem_scale: 512.0
  
  - name: "high_load_small"
    arrival_scale: 1.0      # λ=1.0 (avg 1s between pods)
    duration_shape: 2.0
    duration_scale: 180.0
    pod_cpu_shape: 1.0
    pod_mem_scale: 128.0
  
  # ... add more parameter vectors to reach 15 total

cluster_config:
  nodes:
    - name: "node-1"
      cpu: 4
      memory: 32Gi
    - name: "node-2"
      cpu: 4
      memory: 32Gi
    - name: "node-3"
      cpu: 8
      memory: 64Gi
```

## Step 2: Create Experiment Runner Script

```bash
#!/bin/bash
# experiments/run_experiments.sh

EXPERIMENT_DIR="experiments"
RESULTS_DIR="results/experiment_$(date +%Y%m%d_%H%M%S)"
CONFIG_FILE="$EXPERIMENT_DIR/experiment_config.yaml"

mkdir -p "$RESULTS_DIR"

# Parse configuration (simplified - use proper YAML parser in production)
SEEDS=(1 2 3 4 5 6 7 8 9 10)
WEIGHT_NAMES=("default" "resource_focused" "affinity_focused" "spread_focused" "balanced_high")
PARAM_NAMES=("low_load_small" "low_load_large" "medium_load_small" "medium_load_large" "high_load_small")

# Generate base scenarios for each parameter vector
for param_idx in "${!PARAM_NAMES[@]}"; do
  param_name="${PARAM_NAMES[$param_idx]}"
  
  # Create base config for this parameter set
  cat > "$EXPERIMENT_DIR/config_${param_name}.yaml" <<EOF
scenario:
  name: "experiment-${param_name}"
  outputDir: "scenarios"
  generation:
    numPodEvents: 100
    arrivalScale: 2.0    # Will be overridden based on param
    durationScale: 120.0 # Will be overridden based on param
    # ... other parameters
EOF

  # Generate scenarios with different seeds
  for seed in "${SEEDS[@]}"; do
    echo "Generating scenario: ${param_name}_seed${seed}"
    ./bin/keg scenario generate \
      -c "$EXPERIMENT_DIR/config_${param_name}.yaml" \
      -o "$EXPERIMENT_DIR/scenario_${param_name}_seed${seed}.yaml" \
      --seed "$seed"
  done
done

# Run experiments
experiment_count=0
total_experiments=$((${#SEEDS[@]} * ${#WEIGHT_NAMES[@]} * ${#PARAM_NAMES[@]}))

for param_name in "${PARAM_NAMES[@]}"; do
  for weight_name in "${WEIGHT_NAMES[@]}"; do
    for seed in "${SEEDS[@]}"; do
      experiment_count=$((experiment_count + 1))
      run_name="${param_name}_${weight_name}_seed${seed}"
      
      echo "[$experiment_count/$total_experiments] Running: $run_name"
      
      # Modify scenario to include scheduler weight changes
      # This would need to inject scheduler events into the scenario
      
      # Run simulation
      ./bin/keg simulation start \
        -s "$EXPERIMENT_DIR/scenario_${param_name}_seed${seed}.yaml" \
        --output-dir "$RESULTS_DIR/$run_name" \
        --cluster-reset
      
      # Add metadata about weights used
      echo "$weight_name" > "$RESULTS_DIR/$run_name/weight_config.txt"
      
      # Optional: Add delay between runs to avoid overwhelming the system
      sleep 2
    done
  done
done

echo "All experiments completed! Results in: $RESULTS_DIR"
```

## Step 3: Analyze Results

```bash
# After all experiments complete, run analysis
python analyzer/compare_simulations.py \
  --simulations $(ls -d $RESULTS_DIR/*/) \
  --base-dir "$RESULTS_DIR" \
  --output "$RESULTS_DIR/analysis" \
  --target-metric scheduling_efficiency
```

## Step 4: Generate Report

Create a comprehensive analysis script:

```python
# experiments/analyze_experiment_results.py
import os
import pandas as pd
import numpy as np
from analyzer.scheduler_weight_analyzer import SchedulerWeightAnalyzer

def analyze_experiment_results(results_dir):
    """Analyze the full experiment results."""
    
    # Group results by parameter vector
    parameter_groups = {}
    
    # Load all simulations
    analyzer = SchedulerWeightAnalyzer(results_dir)
    all_sims = [d for d in os.listdir(results_dir) if os.path.isdir(os.path.join(results_dir, d))]
    analyzer.load_simulation_results(all_sims)
    
    # Create comparison DataFrame
    df = analyzer.create_comparison_dataframe()
    
    # Extract parameter and weight info from run names
    df['param_vector'] = df['run_id'].str.extract(r'(.*?)_.*?_seed\d+')[0]
    df['weight_config'] = df['run_id'].str.extract(r'.*?_(.*?)_seed\d+')[0]
    df['seed'] = df['run_id'].str.extract(r'seed(\d+)')[0].astype(int)
    
    # Aggregate results by parameter vector and weight config
    summary = df.groupby(['param_vector', 'weight_config']).agg({
        'scheduling_efficiency': ['mean', 'std'],
        'fragmentation_index': ['mean', 'std'],
        'avg_pending_time': ['mean', 'std'],
        'cpu_utilization': ['mean', 'std'],
        'memory_utilization': ['mean', 'std']
    }).round(3)
    
    # Find best weight config for each parameter vector
    best_configs = df.groupby('param_vector').apply(
        lambda x: x.groupby('weight_config')['scheduling_efficiency'].mean().idxmax()
    )
    
    return summary, best_configs

# Run analysis
results_dir = "results/experiment_20240107_120000"  # Update with actual directory
summary, best_configs = analyze_experiment_results(results_dir)

print("Summary Statistics:")
print(summary)
print("\nBest Weight Configurations by Workload:")
print(best_configs)
```

## Automation Tools

### 1. Experiment Configuration Generator
```python
# experiments/generate_configs.py
import yaml
import itertools

def generate_experiment_configs():
    """Generate all parameter combinations."""
    
    # Define parameter ranges
    arrival_scales = [1.0, 2.0, 5.0, 10.0]  # High to low load
    duration_scales = [60.0, 120.0, 180.0]  # Short to long
    cpu_factors = [0.5, 1.0, 2.0]           # Small to large
    mem_scales = [128.0, 256.0, 512.0]     # Small to large
    
    # Generate combinations
    param_vectors = []
    for idx, (arr, dur, cpu, mem) in enumerate(
        itertools.product(arrival_scales, duration_scales, cpu_factors, mem_scales)
    ):
        if idx >= 15:  # Limit to 15 vectors
            break
        param_vectors.append({
            'name': f'param_vector_{idx+1}',
            'arrival_scale': arr,
            'duration_scale': dur,
            'pod_cpu_factor': cpu,
            'pod_mem_scale': mem
        })
    
    return param_vectors
```

### 2. Results Aggregator
```python
# experiments/aggregate_results.py
def create_experiment_report(results_dir):
    """Create a comprehensive experiment report."""
    
    # Load and analyze results
    # Generate visualizations
    # Create recommendations
    # Export to PDF/HTML report
    pass
```

## Best Practices

1. **Run in Batches**: Don't run all 750 experiments at once
2. **Monitor Resources**: Watch cluster resource usage
3. **Checkpoint Progress**: Save intermediate results
4. **Validate Data**: Check for failed simulations
5. **Version Control**: Track scenario configurations

## Expected Outputs

1. **Raw Results**: 750 simulation directories with metrics
2. **Aggregated Data**: Summary statistics by parameter/weight combination
3. **Visualizations**: Heatmaps, scatter plots, performance curves
4. **Recommendations**: Optimal weights for different workload types
5. **Statistical Analysis**: Confidence intervals, significance tests