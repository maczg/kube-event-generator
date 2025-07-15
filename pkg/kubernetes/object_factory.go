package kubernetes

import (
	"fmt"
	"k8s.io/apiserver/pkg/storage/names"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/apiserver/pkg/storage/names"
)

type ObjectAdapter interface {
	metav1.Object
}
type ObjectOpt func(ObjectAdapter)

var ObjectFactory = &objectFactory{}

type objectFactory struct{}

func (f *objectFactory) NewPod(namespace string, opts ...PodOpt) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.SimpleNameGenerator.GenerateName("pod-"),
			Namespace: namespace,
			Labels:    make(map[string]string),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "main",
					Image: "nginx:latest",
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("200m"),
							v1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyAlways,
		},
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
	}

	for _, opt := range opts {
		opt(pod)
	}

	return pod
}

func (f *objectFactory) NewNode(name string, opts ...NodeOpt) *v1.Node {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: make(map[string]string),
		},
		Spec: v1.NodeSpec{
			Unschedulable: false,
		},
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4"),
				v1.ResourceMemory: resource.MustParse("16Gi"),
				v1.ResourcePods:   resource.MustParse("110"),
			},
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("3800m"),
				v1.ResourceMemory: resource.MustParse("15Gi"),
				v1.ResourcePods:   resource.MustParse("100"),
			},
			Phase: v1.NodeRunning,
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
					Reason: "KubeletReady",
				},
			},
		},
	}

	for _, opt := range opts {
		opt(node)
	}

	return node
}

type PodOpt func(*v1.Pod)

func WithPodLabels(labels map[string]string) PodOpt {
	return func(p *v1.Pod) {
		for k, v := range labels {
			p.Labels[k] = v
		}
	}
}

func WithPodAnnotations(annotations map[string]string) PodOpt {
	return func(p *v1.Pod) {
		if p.Annotations == nil {
			p.Annotations = make(map[string]string)
		}
		for k, v := range annotations {
			p.Annotations[k] = v
		}
	}
}

func WithPodNodeSelector(nodeSelector map[string]string) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.NodeSelector = nodeSelector
	}
}

func WithPodResources(cpu, memory string) PodOpt {
	return func(p *v1.Pod) {
		if len(p.Spec.Containers) == 0 {
			return
		}
		p.Spec.Containers[0].Resources = v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(cpu),
				v1.ResourceMemory: resource.MustParse(memory),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(cpu),
				v1.ResourceMemory: resource.MustParse(memory),
			},
		}
	}
}

func WithPodResourcesDetailed(reqCPU, reqMem, limCPU, limMem string) PodOpt {
	return func(p *v1.Pod) {
		if len(p.Spec.Containers) == 0 {
			return
		}
		p.Spec.Containers[0].Resources = v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(reqCPU),
				v1.ResourceMemory: resource.MustParse(reqMem),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(limCPU),
				v1.ResourceMemory: resource.MustParse(limMem),
			},
		}
	}
}

func WithPodImage(image string) PodOpt {
	return func(p *v1.Pod) {
		if len(p.Spec.Containers) == 0 {
			return
		}
		p.Spec.Containers[0].Image = image
	}
}

func WithPodContainers(containers ...v1.Container) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.Containers = containers
	}
}

func WithPodSchedulerName(schedulerName string) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.SchedulerName = schedulerName
	}
}

func WithPodPriorityClass(priorityClassName string, priority int32) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.PriorityClassName = priorityClassName
		p.Spec.Priority = &priority
	}
}

func WithPodTolerations(tolerations []v1.Toleration) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.Tolerations = tolerations
	}
}

func WithPodAffinity(affinity *v1.Affinity) PodOpt {
	return func(p *v1.Pod) {
		p.Spec.Affinity = affinity
	}
}

type NodeOpt func(*v1.Node)

func WithNodeLabels(labels map[string]string) NodeOpt {
	return func(n *v1.Node) {
		for k, v := range labels {
			n.Labels[k] = v
		}
	}
}

func WithNodeCapacity(cpu, memory, pods string) NodeOpt {
	return func(n *v1.Node) {
		n.Status.Capacity = v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(cpu),
			v1.ResourceMemory: resource.MustParse(memory),
			v1.ResourcePods:   resource.MustParse(pods),
		}
	}
}

func WithNodeAllocatable(cpu, memory, pods string) NodeOpt {
	return func(n *v1.Node) {
		n.Status.Allocatable = v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(cpu),
			v1.ResourceMemory: resource.MustParse(memory),
			v1.ResourcePods:   resource.MustParse(pods),
		}
	}
}

func WithNodeTaints(taints []v1.Taint) NodeOpt {
	return func(n *v1.Node) {
		n.Spec.Taints = taints
	}
}

func WithNodeUnschedulable(unschedulable bool) NodeOpt {
	return func(n *v1.Node) {
		n.Spec.Unschedulable = unschedulable
	}
}

func WithNodeConditions(conditions []v1.NodeCondition) NodeOpt {
	return func(n *v1.Node) {
		n.Status.Conditions = conditions
	}
}

func WithNodeProviderID(providerID string) NodeOpt {
	return func(n *v1.Node) {
		n.Spec.ProviderID = providerID
	}
}

func WithNodeAddresses(addresses []v1.NodeAddress) NodeOpt {
	return func(n *v1.Node) {
		n.Status.Addresses = addresses
	}
}

func (f *objectFactory) NewPodFromTemplate(template *v1.Pod, name string) *v1.Pod {
	pod := template.DeepCopy()
	pod.Name = name
	pod.ResourceVersion = ""
	pod.UID = ""
	pod.CreationTimestamp = metav1.Time{}
	pod.DeletionTimestamp = nil
	pod.Status = v1.PodStatus{Phase: v1.PodPending}
	return pod
}

func (f *objectFactory) NewNodeFromTemplate(template *v1.Node, name string) *v1.Node {
	node := template.DeepCopy()
	node.Name = name
	node.ResourceVersion = ""
	node.UID = ""
	node.CreationTimestamp = metav1.Time{}
	node.DeletionTimestamp = nil
	return node
}

func (f *objectFactory) GeneratePodName(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}

func (f *objectFactory) GenerateNodeName(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}
