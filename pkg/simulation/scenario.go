package simulation

import (
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	"os"
)

type Metadata struct {
	// Name is the name of the scenario
	Name string `yaml:"name" json:"name"`
	// Description provides a brief description of the scenario
	Description string `yaml:"description" json:"description"`
	// CreatedAt is the timestamp when the scenario was created
	CreatedAt string `yaml:"createdAt" json:"createdAt"`
}

type Events struct {
	Pods      []PodEvent           `yaml:"pods" json:"pods"`
	Scheduler []KubeSchedulerEvent `yaml:"scheduler" json:"scheduler"`
}

type Cluster struct {
	Nodes []*v1.Node `yaml:"nodes" json:"nodes"`
}

type Scenario struct {
	// Metadata contains information about the scenario
	Metadata Metadata `yaml:"metadata" json:"metadata"`
	// Cluster represents the cluster configuration for the scenario
	Cluster Cluster `yaml:"cluster" json:"cluster"`
	// Events contains the events that will be executed in the scenario
	Events Events `yaml:"events" json:"events"`
}

func Load(data []byte) (*Scenario, error) {
	var s Scenario
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func LoadFromYaml(filename string) (*Scenario, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Load(data)
}
