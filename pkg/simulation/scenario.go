package simulation

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"os"
	"time"
)

type Metadata struct {
	// Name is the name of the scenario
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Description provides a brief description of the scenario
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// CreatedAt is the timestamp when the scenario was created
	CreatedAt time.Time `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
}

type Events struct {
	Pods      []PodEvent           `yaml:"pods,omitempty" json:"pods,omitempty"`
	Scheduler []KubeSchedulerEvent `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
}

type Cluster struct {
	Nodes []*v1.Node `yaml:"nodes,omitempty" json:"nodes,omitempty"`
}

func NewCluster(nodes []*v1.Node) *Cluster {
	return &Cluster{
		Nodes: nodes,
	}
}

type Scenario struct {
	// Metadata contains information about the scenario
	Metadata Metadata `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	// Cluster represents the cluster configuration for the scenario
	Cluster *Cluster `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	// Events contains the events that will be executed in the scenario
	Events Events `yaml:"events,omitempty" json:"events,omitempty"`
}

type ScenarioOption func(*Scenario)

func ScenarioWithCluster(cluster *Cluster) ScenarioOption {
	return func(s *Scenario) {
		s.Cluster = cluster
	}
}
func ScenarioWithPodsEvents(events []PodEvent) ScenarioOption {
	return func(s *Scenario) {
		s.Events.Pods = events
	}
}

func ScenarioWithName(name string) ScenarioOption {
	return func(s *Scenario) {
		s.Metadata.Name = name
		s.Metadata.CreatedAt = time.Now()
	}
}

func NewScenario(opts ...ScenarioOption) *Scenario {
	s := &Scenario{
		Metadata: Metadata{
			Name:        fmt.Sprintf("scenario-%s", uuid.New().String()[0:3]),
			Description: "",
			CreatedAt:   time.Now(),
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

func (s *Scenario) ToYaml(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}
