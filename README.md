# kube-event-generator

A tool for generating and apply timed events to Kubernetes clusters.

It allows to:
- Define scenarios in YAML format (CLI). Given a number of pods to submit, 
pods arrival times follow Exponential distribution (Poisson) and have duration defined by Weibull distribution. 
Pod request resources are generated following Weibull distribution. See [docs](docs/scenario_gen.md) for more details.
- Run a scenario applying events ordered by time in the cluster.
- Collect and export metrics at the end of the scenario.

Currently, it supports:
- `Pod` events (creation, deletion). The deletion is scheduled only after the pod transit in `Running` state.
- `SchedulerEvent` events ATM change the Kubernetes Scheduler Plugin weights (e.g `NodeResourceFit` from 1 to 5).

> [!WARNING]
> KubeSchedulerPlugin events works only using [kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator)
> SchedulerEvent execution function will call the simulator to reload the scheduler configuration and restarting the scheduler.


## Prerequisites

- Go 1.19+
- Access to a Kubernetes cluster
- `kubectl` configured with cluster access
- `kube-scheduler-simulator` for SchedulerEvent (see [docker](docker/docker-compose.yaml) to spin up a local cluster with the simulator)

## Installation

### Using Go
```bash
go install github.com/maczg/kube-event-generator/cmd/keg@latest
```
### Clone the repo
```bash
git clone https://github.com/maczg/kube-event-generator.git
cd kube-event-generator
go build ./cmd/keg
```

## Usage



### ⚠️ Setup KWOK + KubeSchedulerSimulator

> [!WARNING]
> keg will use the kubeconfig in $HOME/.kube/config by default to connect and work to the cluster.
> be sure to set the correct context in the kubeconfig file.

```bash
cp ~/.kube/config ~/.kube/config.bak
docker-compose -f docker/docker-compose.yaml up -d
export KUBECONFIG=~/.kube/config:$(pwd)/docker/kubeconfig.local.yaml
kubectl config view --flatten > ~/.kube/config
kubectl config use-context simulator
```

### Generate a scenario
```bash
keg scenario generate  -c config.yaml
```
by default it will generate a scenario.yaml file in the scenarios/ directory.

See [config.yaml](config.yaml) for an example of the configuration file.

### Start a scenario
```bash
keg simulation start -s scenario.yaml
```

Start scenario flags: 
- `--scenario` (required) path to the scenario.yaml file.
- `--cluster-reset` ⚠️ KWOK only. It will reset the cluster to the initial state before starting the scenario.
If `cluster: nodes: []` is set in the scenario.yaml, it creates a new cluster with the nodes defined in the scenario.
- `--save-metrics (default true)` will save the metrics in results/ directory.
- `--output-dir` (default results/) path to the directory where the metrics will be saved.

### Compare results

[!WARNING]
> WIP. [analyzer](analyzer) is not yet ready.


### Other commands
see 
```bash
keg cluster --help
```