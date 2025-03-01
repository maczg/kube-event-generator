package scenario

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"time"
)

type Scenario struct {
	Name          string         `yaml:"name" json:"name"`
	Deadline      time.Duration  `yaml:"deadline" json:"deadline"`
	Nodes         []Node         `yaml:"nodes" json:"nodes"`
	PluginConfigs map[string]int `yaml:"pluginsWeights" json:"pluginsWeights"`
	Events        []Event        `yaml:"events" json:"events"`
}

func (s *Scenario) Info() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Scenario: %s\n", s.Name))
	builder.WriteString("Plugin Configurations:\n")
	for plugin, weight := range s.PluginConfigs {
		builder.WriteString(fmt.Sprintf("  %s: %d\n", plugin, weight))
	}
	builder.WriteString("\nNodes:\n")
	for _, node := range s.Nodes {
		builder.WriteString(fmt.Sprintf("  Name: %s\n", node.Name))
		builder.WriteString(fmt.Sprintf("    CPU Allocatable: %s\n", node.CPUAllocatable))
		builder.WriteString(fmt.Sprintf("    Memory Allocatable: %s\n", node.MemAllocatable))
		builder.WriteString(fmt.Sprintf("    Pods Allocatable: %s\n", node.PodsAllocatable))
		builder.WriteString("    Labels:\n")
		for key, val := range node.Labels {
			builder.WriteString(fmt.Sprintf("      %s: %s\n", key, val))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("Events:\n")
	for i, event := range s.Events {
		builder.WriteString(fmt.Sprintf("  Event %d:\n", i+1))
		builder.WriteString(fmt.Sprintf("    After: %s\n", event.After.String()))
		builder.WriteString(fmt.Sprintf("    Duration: %s\n", event.Duration.String()))
		builder.WriteString(fmt.Sprintf("    PodSpec: %+v\n", event.PodSpec))
		builder.WriteString("\n")
	}
	return builder.String()
}

func Load(data []byte) (*Scenario, error) {
	//type s Scenario
	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return nil, err
	}
	return &scenario, nil
}

func LoadFromFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(data)
}
