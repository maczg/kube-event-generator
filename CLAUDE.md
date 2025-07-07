# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
The kube-event-generator (keg) is a Go-based tool for generating and applying timed events to Kubernetes clusters. It simulates realistic workloads by generating pods with resource requests following statistical distributions, supports scheduler configuration changes, and collects metrics for performance analysis.

## Common Commands

### Development Commands
```bash
# Build the project
make build

# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run linter
make lint

# Format code
make fmt

# Run go vet
make vet

# Run all checks (format, vet, lint, tests)
make check

# Generate test coverage report
make test-coverage
```

### Application Commands
```bash
# Generate a scenario from config
./bin/keg scenario generate -c config.yaml

# Start a simulation
./bin/keg simulation start -s scenarios/scenario.yaml

# Reset cluster (KWOK only)
./bin/keg simulation start -s scenarios/scenario.yaml --cluster-reset

# Start with custom output directory
./bin/keg simulation start -s scenarios/scenario.yaml --output-dir custom-results/
```

### Local Development Environment
```bash
# Start local KWOK + KubeSchedulerSimulator environment
make local-env

# Stop local environment
make local-env-stop

# Manual setup
docker-compose -f docker/docker-compose.yaml up -d
cp ~/.kube/config ~/.kube/config.bak
export KUBECONFIG=~/.kube/config:$(pwd)/docker/kubeconfig.local.yaml
kubectl config view --flatten > ~/.kube/config
kubectl config use-context simulator
```

## Architecture

### Core Components
1. **Scenario Generation** (`pkg/scenario/`) - Creates simulation scenarios using statistical distributions
2. **Simulator** (`pkg/simulator/`) - Executes scenarios against Kubernetes clusters
3. **Scheduler** (`pkg/scheduler/`) - Internal event scheduler with priority queue
4. **Cache** (`pkg/cache/`) - Kubernetes resource caching and monitoring
5. **Metrics** (`pkg/metrics/`) - Performance metrics collection
6. **Distribution** (`pkg/distribution/`) - Statistical distribution functions (Exponential, Weibull)

### Event Types
- **Pod Events** - Pod creation/deletion with realistic resource requests
- **Scheduler Events** - Dynamic scheduler configuration changes (weights, policies)

### Key Packages
- `cmd/internal/` - CLI command structure using Cobra
- `pkg/config/` - Configuration management with Viper
- `pkg/errors/` - Custom error types for better error handling
- `pkg/logger/` - Structured logging with logrus
- `pkg/util/` - Utility functions for Kubernetes operations

### Statistical Distributions
- **Arrival Times**: Exponential (Poisson) distribution for pod inter-arrival times
- **Service Times**: Weibull distribution for pod execution durations
- **Resource Requests**: Weibull distribution for CPU/memory requests

## Configuration

### Main Config File
The application uses `config.yaml` for scenario generation parameters:
- `scenario.generation.numPodEvents` - Number of pod events to generate
- `scenario.generation.arrivalScale` - Exponential distribution lambda parameter
- `scenario.generation.durationScale/Shape` - Weibull distribution parameters
- `kubernetes.namespace` - Target namespace (default: "default")
- `scheduler.simulatorUrl` - URL for scheduler simulator API

### Environment Variables
- `SCHEDULER_SIM_URL` - Scheduler simulator URL (default: http://localhost:1212/api/v1/schedulerconfiguration)
- `KUBECONFIG` - Path to kubeconfig file

## Testing

### Test Structure
- Unit tests: `pkg/*/test.go` files
- Integration tests: `test/integration/`
- Mocks: `pkg/testing/mocks/`

### Test Requirements
- Integration tests require running Kubernetes cluster
- Use `//go:build integration` tags for integration tests
- Mock external dependencies (scheduler simulator, Kubernetes API)

## Special Considerations

### Scheduler Simulator Integration
- SchedulerEvent functionality requires [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator)
- Changes scheduler plugin weights dynamically (e.g., NodeResourceFit: 1’5)
- Simulator must be accessible via HTTP API

### KWOK Integration
- Supports KWOK (Kubernetes WithOut Kubelet) for lightweight testing
- Use `--cluster-reset` flag to reset cluster state before simulation
- Cluster nodes can be defined in scenario YAML files

### Output and Results
- Results saved to `results/` directory by default
- Metrics include: node allocation history, pod queue lengths, resource fragmentation
- Supports CSV and JSON output formats
- Python analyzer available in `analyzer/` directory

## Development Notes

### Error Handling
- Uses custom error types from `pkg/errors/`
- Implements error wrapping with context
- Validation errors for configuration parameters

### Concurrency
- Uses goroutines for event execution
- Mutex protection for shared state
- Context-based cancellation support

### Metrics Collection
- Real-time metrics during simulation
- Memory usage tracking
- Resource allocation monitoring
- Queue length statistics