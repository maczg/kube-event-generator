# kube-event-generator (keg)

A tool for generating and applying timed events to Kubernetes clusters, designed to facilitate testing and simulation of complex scheduling scenarios.

## Overview

The kube-event-generator (keg) enables users to create realistic workload patterns in Kubernetes clusters by generating events with precise timing control. It's particularly useful for:

- Testing Kubernetes scheduler behavior under various load conditions
- Simulating production workload patterns in development environments
- Benchmarking cluster performance with reproducible scenarios
- Validating scheduler configurations and policies

### Key Features

- **Statistical Distribution Support**: Generate event timing using exponential and Weibull distributions for realistic workload patterns [WIP]
- **Multiple Event Types**: Support for pod creation/deletion, scheduler configuration changes, and node resource updates [WIP]
- **Integration with Simulators**: Native support for [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator) and [KWOK](https://github.com/kubernetes-sigs/kwok)
- **Resource Tracking**: Built-in cache system to track resource allocation and utilization over time
- **Flexible Scenarios**: YAML-based scenario definitions for reproducible testing

## Installation

### From Source

```bash
git clone https://github.com/yourusername/kube-event-generator.git
cd kube-event-generator
make build
```

### Using Docker

```bash
docker pull quay.io/maczg/kube-event-generator:latest
```

## Quick Start

### 1. Create a Scenario File

Create a `scenario.yaml` file describing your workload:

```yaml
metadata:
  name: "Basic Load Test"
  description: "Simple pod creation and deletion scenario"
events:
  pods:
    - name: web-workload
      arrivalTime: 5s
      evictTime: 30s
      podSpec:
        apiVersion: v1
        kind: Pod
        metadata:
          name: "web-pod"
          namespace: "default"
        spec:
          containers:
            - name: "web"
              image: "nginx:latest"
              resources:
                requests:
                  cpu: "100m"
                  memory: "128Mi"
                limits:
                  cpu: "200m"
                  memory: "256Mi"
```

### 2. Run the Simulation

```bash
# Check cluster status
./bin/keg cluster status

# Run the simulation
./bin/keg simulation run --scenario scenario.yaml --duration 5m

# View results
ls results/
```

## Architecture

keg follows a modular architecture with clear separation of concerns:

```
+--------------------------------------------------+
|                  CLI Interface                   |
|                   (Cobra)                        |
+----------------------+---------------------------+
                       |
+----------------------v---------------------------+
|             Event Scheduler                      |
|         (Priority Queue Based)                   |
+----------------------+---------------------------+
                       |
        +--------------+---------------+
        |              |               |
+-------v--------+ +---v--------+ +----v-----------+
|  Pod Events    | |Scheduler   | |  Node Events   |
|                | |Events      | |                |
+-------+--------+ +---+--------+ +----+-----------+
        |              |               |
        +--------------+---------------+
                       |
+----------------------v---------------------------+
|            Kubernetes Client                     |
|              (client-go)                         |
+--------------------------------------------------+
```

### Core Components

- **Event Scheduler**: Priority queue-based system for managing event timing
- **Event Types**: Extensible event system supporting pods, scheduler configs, and nodes
- **Statistical Distributions**: Pluggable distribution generators for realistic timing
- **Cache System**: Tracks cluster state and resource utilization
- **Kubernetes Integration**: Full client-go integration with informers and listers

## Usage Examples

### Basic Commands

```bash
# Run with verbose logging
./bin/keg simulation run --scenario scenario.yaml --verbose

# Use a specific kubeconfig
./bin/keg cluster status --kubeconfig ~/.kube/config

# Reset cluster state before simulation
./bin/keg cluster reset
./bin/keg simulation run --scenario scenario.yaml
```

### Working with Distributions

Generate events with exponential inter-arrival times:

```yaml
events:
  pods:
    - name: poisson-workload
      distribution:
        type: exponential
        rate: 0.5  # Average 2 seconds between events
      count: 100
      podSpec:
        # ... pod specification
```

### Local Development Environment

keg includes a complete local development environment using KWOK and kube-scheduler-simulator:

```bash
# Start local environment
make local-env

# Run simulation against local environment
export KUBECONFIG=~/.kube/config:$(pwd)/docker/kubeconfig.local.yaml
kubectl config use-context simulator
./bin/keg simulation run --scenario scenario.yaml

# Stop local environment
make local-env-stop
```

## Development

### Prerequisites

- Go 1.21 or later
- Docker (for local environment)
- Make

### Building

```bash
# Build binary
make build

# Run tests
make test

# Run linting
make lint

# Generate test coverage
make test-coverage
```

### Project Structure

```
cmd/                    # CLI implementation
├── app.go             # Application setup
├── root.go            # Root command
├── cluster/           # Cluster management commands
└── simulation/        # Simulation commands

pkg/
├── cache/             # Resource tracking and caching
├── distribution/      # Statistical distributions
├── kubernetes/        # Kubernetes client utilities
├── logger/           # Centralized logging
├── scheduler/        # Event scheduling engine
├── simulation/       # Simulation orchestration
└── util/             # Common utilities
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Reporting Issues

Please use GitHub Issues to report bugs or request features. Include:
- keg version (`keg version`)
- Kubernetes version
- Scenario file (if applicable)
- Full error output with `--verbose` flag

## Roadmap

- [ ] Support for more distribution types (normal, uniform, custom)
- [ ] Web UI for real-time visualization
- [ ] Integration with Prometheus metrics
- [ ] Scenario recorder to capture real cluster patterns
- [ ] Multi-cluster simulation support

## Related Projects

- [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator) - Simulator for kube-scheduler
- [KWOK](https://github.com/kubernetes-sigs/kwok) - Kubernetes WithOut Kubelet
- [cluster-api](https://github.com/kubernetes-sigs/cluster-api) - Declarative cluster creation

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is inspired by the need for better testing tools in the Kubernetes scheduling ecosystem. Special thanks to the Kubernetes sig-scheduling community for their continued work on improving cluster scheduling.