package scenario

import v1 "k8s.io/api/core/v1"

type Cluster struct {
	// Nodes in the cluster
	Nodes []v1.Node `yaml:"nodes" json:"nodes"`
	// SchedulerWeights map of scheduler plugin name to weight
	SchedulerWeights map[string]int `yaml:"schedulerWeights" json:"schedulerWeights"`
}
