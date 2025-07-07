package scenario

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scenario struct {
	Cluster  *Cluster `yaml:"cluster" json:"cluster"`
	Events   *Events  `yaml:"events" json:"events"`
	Metadata Metadata `yaml:"metadata" json:"metadata"`
}

func (s *Scenario) Describe() {
	if s.Cluster != nil {
		for _, node := range s.Cluster.Nodes {
			logrus.Infof("node %s allocatable [cpu: %s,mem %s,pod s%s]", node.Name, node.Status.Allocatable.Cpu().String(), node.Status.Allocatable.Memory().String(), node.Status.Allocatable.Pods().String())
		}
	} else {
		logrus.Warn("cluster is not set")
	}

	if s.Events.SchedulerConfigs != nil {
		for _, event := range s.Events.SchedulerConfigs {
			logrus.Infof("scheduler config %s [from: %s]", event.ID(), event.ExecuteAfterDuration().String())
		}
	} else {
		logrus.Warn("scheduler config is not set")
	}

	if s.Events.Pods != nil {
		longest := s.Events.GetLongestEvent()
		biggestCpu := s.Events.GetLargerCpuRequest()
		biggestMem := s.Events.GetLargerMemRequest()

		logrus.Infof("longest pod event %s [from: %s]", longest.Name, longest.ExecuteAfterDuration().String())

		cpuMax := biggestCpu.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]
		memMax := biggestMem.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceMemory]

		logrus.Infof("biggest event by cpu %s [cpu: %s]", biggestCpu.Name, cpuMax.String())
		logrus.Infof("biggest event by mem %s [mem: %s]", biggestMem.Name, memMax.String())
	}
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

type Opt func(*Scenario)

func WithCluster(cluster *Cluster) Opt {
	return func(s *Scenario) {
		s.Cluster = cluster
	}
}
func WithPodEvents(events []*PodEvent) Opt {
	return func(s *Scenario) {
		s.Events.Pods = events
	}
}

func WithName(name string) Opt {
	return func(s *Scenario) {
		s.Metadata.Name = name
		s.Metadata.CreatedAt = time.Now()
	}
}

func NewScenario(opts ...Opt) *Scenario {
	s := &Scenario{
		Cluster: &Cluster{},
		Events:  &Events{},
	}
	for _, opt := range opts {
		opt(s)
	}

	if s.Metadata.Name == "" {
		s.Metadata.Name = fmt.Sprintf("default-%s", uuid.New().String()[0:3])
		s.Metadata.CreatedAt = time.Now()
	}

	return s
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

type Metadata struct {
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`
	Name      string    `yaml:"name" json:"name"`
}

// Cluster represents the cluster configuration.
type Cluster struct {
	Nodes []*v1.Node `yaml:"nodes" json:"nodes"`
}

func (c *Cluster) Create(clientset *kubernetes.Clientset) error {
	for _, node := range c.Nodes {
		_, err := clientset.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create node %s: %v", node.Name, err)
		}
	}

	return nil
}

func NewCluster() *Cluster {
	return &Cluster{
		Nodes: make([]*v1.Node, 0),
	}
}

type Events struct {
	Pods             []*PodEvent       `yaml:"pods" json:"pods"`
	SchedulerConfigs []*SchedulerEvent `yaml:"scheduler" json:"scheduler"`
}

func (e *Events) GetLargerCpuRequest() *PodEvent {
	if len(e.Pods) == 0 {
		return nil
	}

	largest := slices.MaxFunc(e.Pods, func(a, b *PodEvent) int {
		cpuA := a.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]
		cpuB := b.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]

		return cmp.Compare(cpuA.Value(), cpuB.Value())
	})

	return largest
}

func (e *Events) GetLargerMemRequest() *PodEvent {
	if len(e.Pods) == 0 {
		return nil
	}

	largest := slices.MaxFunc(e.Pods, func(a, b *PodEvent) int {
		memA := a.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceMemory]
		memB := b.Pod.Spec.Containers[0].Resources.Requests[v1.ResourceMemory]

		return cmp.Compare(memA.Value(), memB.Value())
	})

	return largest
}

func (e *Events) GetLongestEvent() *PodEvent {
	if len(e.Pods) == 0 {
		return nil
	}

	longest := slices.MaxFunc(e.Pods, func(a, b *PodEvent) int {
		return cmp.Compare(a.ExecuteAfterDuration(), b.ExecuteAfterDuration())
	})

	return longest
}

func NewEvents() *Events {
	return &Events{
		Pods:             make([]*PodEvent, 0),
		SchedulerConfigs: make([]*SchedulerEvent, 0),
	}
}
