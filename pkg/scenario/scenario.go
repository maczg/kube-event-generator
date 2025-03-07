package scenario

import (
	"cmp"
	"github.com/ghodss/yaml"
	"github.com/jedib0t/go-pretty/v6/table"
	"io"
	"os"
	"slices"
)

type Metadata struct {
	Name      string `yaml:"name" json:"name"`
	CreatedAt string `yaml:"created_at" json:"created_at"`
}

type Scenario struct {
	//NodeName string `yaml:"name" json:"name"`
	Metadata Metadata `yaml:"metadata" json:"metadata"`
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

func (s *Scenario) Describe(writers ...io.Writer) {
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	t := table.NewWriter()
	t.SetOutputMirror(io.MultiWriter(writers...))

	t.AppendSeparator()
	t.AppendRow(table.Row{"Cluster", "", ""})
	t.AppendSeparator()
	t.AppendRow(table.Row{"node", "cpu", "memory"})
	for _, n := range s.Cluster.Nodes {
		t.AppendRow(table.Row{
			n.Name,
			n.Status.Capacity.Cpu(),
			n.Status.Capacity.Memory(),
		})
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{"Events", "", ""})
	t.AppendSeparator()
	t.AppendRow(table.Row{"# events", "longest", "duration"})
	numEvents := len(s.Events)
	// Using slices.MaxFunc to get the event with the longest duration (assumed to be the one with the maximum From value)
	longest := slices.MaxFunc(s.Events, func(a, b Event) int {
		return cmp.Compare(a.From, b.From)
	})

	t.AppendRow(table.Row{
		numEvents,
		longest.From,
		"",
	})
	t.Render()
}
