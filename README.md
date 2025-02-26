# Overview
This repository provides a framework to simulate Kubernetes (atm mainly scheduling) events in a controlled manner. 

For exampe, it allows you to define when Pods are created, how long they run, and when they are evicted. 
By running these scenarios against a Kubernetes cluster (or simulator), you can gather insights (into scheduling efficiency), performance trade-offs,
and the impact of different (scoring) strategies when the cluster is subject to different workloads/conditions.

# Features

- Scenario-Driven: Define all events steps in YAML, making test orchestration straightforward.
- Easy Node & Pod Generation: Node and Pod factories let you programmatically configure resource capacities, labels, and container resource requests for each scenario.
- Event Queue: An internal event queue executes tasks in chronological order, modeling real-time behavior (create, run, then evict).

# Getting Started
## Prerequisites
- Go 1.19+
- A Kubernetes cluster (currently KWOK is recommended, but any cluster should work except for the node management)
- A valid kubeconfig file so that client-go can communicate with your cluster

## Usage

### Running a Scenario

1. Define a scenario in the current directory (see `examples/scenario.yaml` for an example).
```yaml
events:
  - type: create
    podSpec:
      name: "test-pod"
      namespace: "default"
      image: "nginx:latest"
      resources:
        cpu: "100m"
        memory: "128Mi"
    delayAfter: 0s
    duration: 30s

  - type: create
    podSpec:
      name: "another-pod"
      namespace: "default"
      image: "busybox"
      resources:
        cpu: "200m"
        memory: "256Mi"
    delayAfter: 5s
    duration: 20s
```

2. Run.

**WIP**
```bash
go run main.go
```