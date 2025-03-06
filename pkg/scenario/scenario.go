package scenario

import (
	"github.com/ghodss/yaml"
	"os"
)

type Scenario struct {
	Name string `yaml:"name" json:"name"`
	// Cluster state
	Cluster Cluster `yaml:"cluster" json:"cluster"`
	// Events that will be applied to the cluster
	Events []Event `yaml:"events" json:"events"`
}

func Load(data []byte) (*Scenario, error) {
	var s Scenario
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func LoadYaml(filename string) (*Scenario, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Load(data)
}

func (s *Scenario) Dump(filename string) error {
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
