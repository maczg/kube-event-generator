package builder

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/maczg/kube-event-generator/pkg/scenario"
)

// ScenarioBuilder helps build scenarios with a fluent interface.
type ScenarioBuilder struct {
	scenario *scenario.Scenario
	errors   []error
}

// NewScenarioBuilder creates a new scenario builder.
func NewScenarioBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: &scenario.Scenario{
			Metadata: scenario.Metadata{
				Name:      name,
				CreatedAt: time.Now(),
			},
			Events: &scenario.Events{
				Pods:             make([]*scenario.PodEvent, 0),
				SchedulerConfigs: make([]*scenario.SchedulerEvent, 0),
			},
		},
		errors: make([]error, 0),
	}
}

// WithMetadata sets scenario metadata.
func (b *ScenarioBuilder) WithMetadata(name string, createdAt time.Time) *ScenarioBuilder {
	b.scenario.Metadata = scenario.Metadata{
		Name:      name,
		CreatedAt: createdAt,
	}

	return b
}

// AddPodEvent adds a pod event to the scenario.
func (b *ScenarioBuilder) AddPodEvent(name string, after, duration time.Duration, pod *v1.Pod) *ScenarioBuilder {
	if pod == nil {
		b.errors = append(b.errors, fmt.Errorf("pod cannot be nil for event %s", name))
		return b
	}

	event := &scenario.PodEvent{
		Name:            name,
		ExecuteAfter:    scenario.EventDuration(after),
		ExecuteDuration: scenario.EventDuration(duration),
		Pod:             pod,
	}

	b.scenario.Events.Pods = append(b.scenario.Events.Pods, event)

	return b
}

// AddSchedulerEvent adds a scheduler configuration event.
func (b *ScenarioBuilder) AddSchedulerEvent(name string, after time.Duration, weights map[string]int32) *ScenarioBuilder {
	if len(weights) == 0 {
		b.errors = append(b.errors, fmt.Errorf("weights cannot be empty for scheduler event %s", name))
		return b
	}

	event := &scenario.SchedulerEvent{
		Name:         name,
		ExecuteAfter: scenario.EventDuration(after),
		Weights:      weights,
	}

	b.scenario.Events.SchedulerConfigs = append(b.scenario.Events.SchedulerConfigs, event)

	return b
}

// AddNode adds a node to the cluster configuration.
func (b *ScenarioBuilder) AddNode(node *v1.Node) *ScenarioBuilder {
	if b.scenario.Cluster == nil {
		b.scenario.Cluster = &scenario.Cluster{}
	}

	if node == nil {
		b.errors = append(b.errors, fmt.Errorf("node cannot be nil"))
		return b
	}

	b.scenario.Cluster.Nodes = append(b.scenario.Cluster.Nodes, node)

	return b
}

// Build builds the scenario and returns any errors.
func (b *ScenarioBuilder) Build() (*scenario.Scenario, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("scenario build failed with %d errors: %v", len(b.errors), b.errors)
	}

	// Validate scenario.
	if len(b.scenario.Events.Pods) == 0 && len(b.scenario.Events.SchedulerConfigs) == 0 {
		return nil, fmt.Errorf("scenario must have at least one event")
	}

	return b.scenario, nil
}

// PodBuilder helps build pods with a fluent interface.
type PodBuilder struct {
	pod    *v1.Pod
	errors []error
}

// NewPodBuilder creates a new pod builder.
func NewPodBuilder(name, namespace string) *PodBuilder {
	return &PodBuilder{
		pod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1.PodSpec{
				Containers: make([]v1.Container, 0),
			},
		},
		errors: make([]error, 0),
	}
}

// WithLabels adds labels to the pod.
func (b *PodBuilder) WithLabels(labels map[string]string) *PodBuilder {
	b.pod.Labels = labels
	return b
}

// WithAnnotations adds annotations to the pod.
func (b *PodBuilder) WithAnnotations(annotations map[string]string) *PodBuilder {
	b.pod.Annotations = annotations
	return b
}

// AddContainer adds a container to the pod.
func (b *PodBuilder) AddContainer(name, image string, cpuMillis, memoryMi int64) *PodBuilder {
	container := v1.Container{
		Name:  name,
		Image: image,
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(cpuMillis, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(memoryMi*1024*1024, resource.BinarySI),
			},
		},
	}

	b.pod.Spec.Containers = append(b.pod.Spec.Containers, container)

	return b
}

// WithNodeSelector adds node selector to the pod.
func (b *PodBuilder) WithNodeSelector(selector map[string]string) *PodBuilder {
	b.pod.Spec.NodeSelector = selector
	return b
}

// WithTolerations adds tolerations to the pod.
func (b *PodBuilder) WithTolerations(tolerations []v1.Toleration) *PodBuilder {
	b.pod.Spec.Tolerations = tolerations
	return b
}

// Build builds the pod and returns any errors.
func (b *PodBuilder) Build() (*v1.Pod, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("pod build failed with %d errors: %v", len(b.errors), b.errors)
	}

	// Validate pod.
	if len(b.pod.Spec.Containers) == 0 {
		return nil, fmt.Errorf("pod must have at least one container")
	}

	return b.pod, nil
}

// NodeBuilder helps build nodes with a fluent interface.
type NodeBuilder struct {
	node   *v1.Node
	errors []error
}

// NewNodeBuilder creates a new node builder.
func NewNodeBuilder(name string) *NodeBuilder {
	return &NodeBuilder{
		node: &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Status: v1.NodeStatus{
				Conditions: []v1.NodeCondition{
					{
						Type:   v1.NodeReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		},
		errors: make([]error, 0),
	}
}

// WithCapacity sets the node capacity.
func (b *NodeBuilder) WithCapacity(cpuCores, memoryGi int64, maxPods int64) *NodeBuilder {
	b.node.Status.Capacity = v1.ResourceList{
		v1.ResourceCPU:    *resource.NewQuantity(cpuCores, resource.DecimalSI),
		v1.ResourceMemory: *resource.NewQuantity(memoryGi*1024*1024*1024, resource.BinarySI),
		v1.ResourcePods:   *resource.NewQuantity(maxPods, resource.DecimalSI),
	}

	// Set allocatable same as capacity by default.
	b.node.Status.Allocatable = b.node.Status.Capacity.DeepCopy()

	return b
}

// WithAllocatable sets the node allocatable resources.
func (b *NodeBuilder) WithAllocatable(cpuCores, memoryGi int64, maxPods int64) *NodeBuilder {
	b.node.Status.Allocatable = v1.ResourceList{
		v1.ResourceCPU:    *resource.NewQuantity(cpuCores, resource.DecimalSI),
		v1.ResourceMemory: *resource.NewQuantity(memoryGi*1024*1024*1024, resource.BinarySI),
		v1.ResourcePods:   *resource.NewQuantity(maxPods, resource.DecimalSI),
	}

	return b
}

// WithLabels adds labels to the node.
func (b *NodeBuilder) WithLabels(labels map[string]string) *NodeBuilder {
	b.node.Labels = labels
	return b
}

// WithTaints adds taints to the node.
func (b *NodeBuilder) WithTaints(taints []v1.Taint) *NodeBuilder {
	b.node.Spec.Taints = taints
	return b
}

// Build builds the node and returns any errors.
func (b *NodeBuilder) Build() (*v1.Node, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("node build failed with %d errors: %v", len(b.errors), b.errors)
	}

	// Validate node.
	if b.node.Status.Capacity == nil {
		return nil, fmt.Errorf("node must have capacity set")
	}

	return b.node, nil
}
