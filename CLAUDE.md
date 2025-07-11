# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
The kube-event-generator (keg) is a Go-based tool for generating and applying timed events to Kubernetes clusters. It generates scenarios where events have an arrival and optionally departure time. An internal scheduler manages the lifecycle of these events that can be:
- Pod creation and deletion
- Kube-scheduler configuration changes
- Node resource updates

The generation of events is based on statistical distributions, allowing for realistic simulation of Kubernetes workloads.

## Common Commands

### Development Commands
```bash
# Build the project
make build

# Run tests
make test              # Run all tests
make test-unit        # Run unit tests only
make test-integration # Run integration tests
make test-coverage    # Generate coverage report

# Code quality
make lint             # Run golangci-lint
make fmt              # Format code
make check            # Run all quality checks
make security         # Run security checks with gosec

# Docker operations
make docker-build     # Build Docker image
make docker-push      # Push to quay.io/maczg/kube-event-generator
```

### Application Commands
```bash
# Run the application
./bin/keg --help

# Common flags
./bin/keg --verbose              # Enable verbose logging
./bin/keg --log-format json      # Use JSON log format
./bin/keg --kubeconfig <path>    # Specify kubeconfig file

# Cluster commands
./bin/keg cluster status         # Check cluster status
./bin/keg cluster reset          # Reset cluster to initial state
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

The local environment includes:
- KWOK cluster (port 3131) - Lightweight Kubernetes API simulation
- KubeSchedulerSimulator backend (port 1212)
- Web UI frontend (port 3000)
- Debuggable scheduler instance

## Architecture

### Core Components

**Event Scheduler (`pkg/scheduler/`)**
- Priority queue-based event scheduling system
- Supports events with arrival and departure times
- Timer-based execution with goroutine management
- Metrics tracking for event statistics

**Event Types**
- `PodEvent`: Creates/deletes pods with resource specifications
- `SchedulerEvent`: Updates scheduler configuration
- `NodeEvent`: Modifies node resources and labels

**Statistical Distributions (`pkg/distribution/`)**
- Interface-based design for extensibility
- Exponential distribution for Poisson processes
- Weibull distribution for flexible modeling
- Used for generating realistic event timing

**Kubernetes Integration (`pkg/kubernetes/`)**
- Full client-go integration
- Cluster status checking and validation
- Scheduler configuration management
- Safe cluster reset functionality

### Code Structure
```
cmd/                      # CLI implementation using Cobra
├── app.go               # Main application setup
├── root.go              # Root command and global flags
└── cluster/             # Cluster management subcommands

pkg/
├── distribution/        # Statistical distribution generators
├── kubernetes/          # K8s API client and operations
├── logger/             # Centralized logging (logrus wrapper)
├── scheduler/          # Event scheduling engine
└── util/               # Common utilities
```

## Configuration

### CLI Flags
- `--verbose, -v`: Enable verbose logging
- `--log-format`: Log format (text/json)
- `--log-output`: Log output destination
- `--kubeconfig`: Path to kubeconfig file

### Docker Environment Configuration
- `docker/config.yaml`: Simulator configuration
- `docker/scheduler.yaml`: Scheduler settings
- `docker/kubeconfig.yaml`: Kubernetes access configuration
- `docker/kwok.yaml`: KWOK cluster configuration

## Testing

### Test Requirements
- Unit tests use standard Go testing
- Integration tests require `-tags=integration`
- Race condition detection enabled by default
- Coverage reports available via `make test-coverage`

### Running Specific Tests
```bash
# Run a specific test
go test ./pkg/scheduler -run TestPriorityQueue

# Run tests with verbose output
go test -v ./...

# Run with race detection
go test -race ./...
```

## Special Considerations

1. **Event Timing**: All events use Unix timestamps for scheduling. The scheduler maintains precise timing using Go's time.Timer.

2. **Resource Management**: When creating pod events, ensure resource requests/limits are realistic for your test cluster.

3. **Concurrent Operations**: The scheduler uses proper synchronization with mutexes. Be careful when modifying the event queue during runtime.

4. **KWOK Limitations**: The local KWOK environment simulates Kubernetes API but doesn't actually run containers. Perfect for testing scheduling logic without resource overhead.

5. **Error Handling**: The codebase uses explicit error returns. Always check errors from Kubernetes API operations.

6. **Logging**: Use the centralized logger from `pkg/logger`. It supports structured logging and multiple output formats.