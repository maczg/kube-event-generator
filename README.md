# kube-event-generator (keg)

[![ci](https://github.com/maczg/kube-event-generator/actions/workflows/ci.yml/badge.svg)](https://github.com/maczg/kube-event-generator/actions/workflows/ci.yml)

keg replays a timed workload, pods that arrive, run for a given time and leave, against the Kubernetes scheduler, and records how the scheduler placed them. Its purpose is to compare scheduler configurations, in particular the weights of the score plugins, on exactly the same workload.

keg is a research prototype. The [status table](#status) lists what works today and what does not.

## Why

For every pending pod, kube-scheduler filters the nodes that can host it and then ranks the feasible ones with a set of score plugins. The final score of a node is a weighted sum,

```
score(node) = Σ_i  w_i · s_i(node),    s_i ∈ [0, 100]
```

where `s_i` is the score of plugin `i` and `w_i` its weight in the scheduler profile. The default profile weights `TaintToleration` 3, `NodeAffinity`, `PodTopologySpread` and `InterPodAffinity` 2, and `NodeResourcesFit`, `NodeResourcesBalancedAllocation` and `ImageLocality` 1. The plugins pursue different goals (`NodeResourcesFit` with the default `LeastAllocated` strategy spreads load, `NodeResourcesBalancedAllocation` keeps the CPU and memory usage of each node in proportion), so the weights decide which goal wins when they disagree.

The weights are fixed for the lifetime of the scheduler process, and the right values depend on the workload. Whether a different set of weights would have served a workload better can only be answered by running the same workload, with the same arrival times, run times and requests, under each candidate configuration and comparing the outcome. Doing that by hand is slow and hard to repeat. keg makes the workload a file: a scenario lists timed pod events and timed weight changes, keg replays it against a cluster and exports the scheduler-side metrics (pending times, queue length, per-node allocation) as CSV.

## Status

| Feature | Status | Notes |
|---|---|---|
| Pod create and delete events with arrival and run time | Works | The run time starts when the pod is observed `Running`, so time spent pending is not counted. |
| Scheduler weight events | Works with kube-scheduler-simulator | Applied through the simulator's `/api/v1/schedulerconfiguration` API. The simulator applies a configuration by restarting the scheduler. On the local environment a pod created right after a change is bound in 0.8–0.9 s instead of 0.1 s (5 measurements each); a production scheduler with leader election would take longer. A stock kube-scheduler has no API for this. |
| Scenario generation | Works | Poisson arrivals, Weibull run times and requests, one pod class per scenario, seeded. `podCpuFactor` is not used. |
| Metrics export | Works | CSV per run under `results/`: pending and running time per pod, pending-queue length, per-node allocation, allocation ratio and free resources over time. CPU in millicores, memory in bytes. |
| Cluster definition in the scenario (`cluster.nodes`) | Not implemented | The field is parsed and ignored: create nodes with `kubectl apply`, as in `scenarios/nodes.example.yaml`. |
| Node events (add, remove, resize) | Not implemented | |
| `keg cluster reset --nodes` | Not implemented | Pods and scheduler weights can be reset. |
| Container image | Not available | There is no Dockerfile and no published image yet. |
| Integration tests | Not implemented | CI runs the unit tests and golangci-lint. |

Known limitations:

- Events are dispatched by a loop that ticks every 10 ms, so an arrival is applied within 10 ms of its scheduled time.
- The simulation ends when the last pod with a run time is deleted. A pod that can never be scheduled keeps it running: use `--timeout`.
- A weight event changes a plugin in the `multiPoint` section of the first profile that lists it; other profiles keep their weights.

## Installation

From source, with Go 1.24 or later:

```bash
git clone https://github.com/maczg/kube-event-generator.git
cd kube-event-generator
make build          # produces bin/keg
```

## Quick start

keg works against any cluster reachable through a kubeconfig, but scheduler weight events need [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator). The repository ships a local environment with the simulator and a [KWOK](https://github.com/kubernetes-sigs/kwok) cluster:

```bash
make local-env                                       # docker compose up in docker/
export KUBECONFIG=$PWD/docker/kubeconfig.local.yaml
kubectl apply -f scenarios/nodes.example.yaml       # four simulated nodes
./bin/keg cluster status
```

A scenario lists pod events and, optionally, scheduler events. Times are relative to the start of the simulation:

```yaml
metadata:
  name: basic
events:
  scheduler:
    - name: favour-balance
      arrivalTime: 0s
      weights:
        NodeResourcesFit: 1
        NodeResourcesBalancedAllocation: 10
  pods:
    - name: web
      arrivalTime: 20s     # submitted 20 s after the start
      evictTime: 30s       # deleted 30 s after it starts running
      podSpec:
        metadata:
          name: web
          namespace: default
        spec:
          containers:
            - name: web
              image: nginx
              resources:
                requests:
                  cpu: 500m
                  memory: 256Mi
```

Run it, and find the CSV files under `results/<simulation id>/`:

```bash
./bin/keg simulation start --scenario scenario.yaml --timeout 10m
```

`simulation start` accepts `--scenario`, `--timeout`, `--metrics` (CSV export, on by default) and `--scheduler-url` (default `http://localhost:1212/api/v1/schedulerconfiguration`). The global flags are `--kubeconfig`, `--verbose`, `--log-format` and `--log-file`.

### Generating a workload

`keg scenario generate` draws pod events from statistical distributions configured in a YAML file ([`config.yaml`](config.yaml) is an example):

```bash
./bin/keg scenario generate --config config.yaml --name my-workload --output-dir scenarios
```

Inter-arrival times are exponential, run times and CPU and memory requests follow Weibull distributions. Set `seed` in the configuration to obtain the same workload again; the seed is recorded in the scenario description.

### Resetting the cluster

```bash
./bin/keg cluster reset --force                    # deletes pods outside system namespaces, resets weights
./bin/keg cluster reset --force --scheduler=false  # pods only
```

`reset` acts on the cluster selected by `--kubeconfig` or `KUBECONFIG`: check the current context before running it.

## Architecture

```
scenario.yaml ──► simulation ──► event scheduler (priority queue by arrival time)
                     │                 │
                     │                 ├── pod events ──────► Kubernetes API (client-go)
                     │                 └── scheduler events ─► kube-scheduler-simulator API
                     │
                     └── cache (pod and node informers) ──► CSV metrics
```

- **Event scheduler** (`pkg/scheduler`): a priority queue ordered by arrival time; each due event runs in its own goroutine.
- **Events** (`pkg/simulation`): a pod event creates the pod, waits for it to run and schedules its deletion; a scheduler event sets plugin weights.
- **Cache** (`pkg/cache`): informers that track pending pods and per-node allocation, and export the metrics.
- **Distributions** (`pkg/distribution`): exponential and Weibull samplers used by the generator.

## Development

```bash
make test       # unit tests with the race detector
make lint       # golangci-lint, configured in .golangci.yml
```

```
cmd/
├── cluster/        # keg cluster status | reset
├── scenario/       # keg scenario generate
└── simulation/     # keg simulation start
pkg/
├── cache/          # informer-based state tracking and CSV export
├── distribution/   # statistical distributions
├── kubeutils/      # client, object factory, scheduler configuration API client
├── logger/
├── scheduler/      # event queue and dispatch loop
├── simulation/     # scenario model, pod and scheduler events, simulation lifecycle
└── util/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow.

## Roadmap

Next, in order:

1. Apply `cluster.nodes` from the scenario, so that a scenario fully describes an experiment.
2. Node events: add, remove and resize nodes during a run.
3. Integration tests against the local environment in CI.
4. A Dockerfile and a published image.

Open questions this tool is meant to help answer: how sensitive scheduling outcomes are to the score weights across workload classes, whether weights adapted at run time beat the best static weights once the cost of applying them is counted, and how to apply new weights without restarting the scheduler.

## Related projects

- [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator): simulator for kube-scheduler, used here to change weights at run time.
- [KWOK](https://github.com/kubernetes-sigs/kwok): Kubernetes without kubelet, used here to simulate nodes.

## License

Apache License 2.0, see [LICENSE](LICENSE).
