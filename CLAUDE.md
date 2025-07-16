# CLAUDE.md

## Overview
kube-event-generator (keg) - Go tool for simulating timed Kubernetes events (pods, scheduler configs, node resources) using statistical distributions.

## Quick Start
```bash
# Build & test
make build              # Build binary
make test               # Run tests
make lint               # Run linter

# Local environment
make local-env          # Start KWOK + simulator
make local-env-stop     # Stop environment

# Run
./bin/keg --help
./bin/keg cluster status
./bin/keg cluster reset
```

## Project Structure
```
cmd/                    # CLI (Cobra)
pkg/
├── scheduler/          # Event scheduling (priority queue, timers)
├── distribution/       # Statistical distributions (exponential, weibull)
├── kubernetes/         # K8s client operations
└── logger/             # Structured logging
```

**Event Types**: PodEvent, SchedulerEvent, NodeEvent

## Key Files
- `docker/docker-compose.yaml` - Local environment setup
- `--verbose/-v` - Enable verbose logging
- `--kubeconfig` - K8s config path

## Testing
```bash
go test ./pkg/scheduler -run TestName  # Specific test
go test -race ./...                    # Race detection
make test-coverage                     # Coverage report
```

## Notes
- Events use Unix timestamps
- KWOK simulates K8s API without running containers
- Always check errors from K8s operations
- Use `pkg/logger` for logging