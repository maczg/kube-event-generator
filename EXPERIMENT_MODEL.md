# Experiment Model for Scheduler Weight Evaluation

## Overview
This experiment model evaluates the impact of different scheduler plugin weight configurations on cluster performance metrics under various workload patterns.

## Model Definition

### Parameters

**θ = [input_avg_rate, avg_duration, avg_mem, avg_cpu]**
- `input_avg_rate`: Average pod arrival rate (λ for Poisson distribution)
- `avg_duration`: Average pod execution duration
- `avg_mem`: Average memory request per pod
- `avg_cpu`: Average CPU request per pod

### Experimental Design

1. **Seeds**: S₁, S₂, ..., S₁₀
   - 10 different random seeds for statistical validity

2. **Weight Vectors**: w̄₁, w̄₂, ..., w̄₅
   - 5 different scheduler weight configurations
   - Each vector contains weights for scheduler plugins (e.g., NodeResourcesFit, NodeAffinity, PodTopologySpread)

3. **Parameter Vectors**: θ̄₁, θ̄₂, ..., θ̄₁₅
   - 15 different workload parameter combinations
   - Varying arrival rates, durations, and resource requirements

### Experiment Generation

For each combination:
```
g(θ̄ᵢ, s) where i ∈ {1...5}, j ∈ {1...15}
```

This generates:
- **Total experiments**: 10 seeds × 5 weight vectors × 15 parameter vectors = **750 experiments**

### Cluster Configuration
- Fixed cluster setup with predefined nodes
- Each node with specific CPU/memory capacity

### Performance Metrics

For each experiment, measure:
1. **Defragmentation Index**: Resource fragmentation over time
2. **Scheduling Efficiency**: Combined metric of success rate and resource utilization
3. **Average Pending Time**: Time pods wait to be scheduled
4. **Resource Balance**: Distribution of resources across nodes
5. **Queue Length**: Maximum pending pods

### Analysis Goals

1. Identify optimal weight configurations for different workload patterns
2. Understand trade-offs between different metrics
3. Develop recommendations for weight tuning based on workload characteristics

## Experimental Procedure

1. **Generate Scenarios**: For each θ̄ᵢ and seed, generate event sequence
2. **Apply Weights**: Run simulation with each weight vector w̄ⱼ
3. **Collect Metrics**: Record all performance metrics
4. **Analyze Results**: Compare performance across weight configurations
5. **Identify Patterns**: Find correlations between weights and metrics