package pkg

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewNode(name string, cpu string, memory string, pod string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
				corev1.ResourcePods:   resource.MustParse(pod),
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
				corev1.ResourcePods:   resource.MustParse(pod),
			},
		},
	}
}

// NewNodeWithLabels creates a new node with the given name, cpu, memory, and labels
func NewNodeWithLabels(name string, cpu string, memory string, pod string, labels map[string]string) *corev1.Node {
	n := NewNode(name, cpu, memory, pod)
	n.Labels = labels
	return n
}

func NewNodeBatch(name string, cpu string, memory string, count int) []*corev1.Node {
	nodes := make([]*corev1.Node, count)
	for i := 0; i < count; i++ {
		_name := fmt.Sprintf("%s-%d", name, i)
		nodes[i] = NewNode(_name, cpu, memory, "110")
	}
	return nodes

}
