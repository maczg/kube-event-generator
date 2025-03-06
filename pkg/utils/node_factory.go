package utils

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var NodeFactory = &nodeFactory{}

type nodeFactory struct{}

func (n *nodeFactory) CreateNode(ctx context.Context, client kubernetes.Clientset, node *corev1.Node) error {
	_, err := client.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
	return err
}

func (n *nodeFactory) DeleteNode(ctx context.Context, client kubernetes.Clientset, name string) error {
	return client.CoreV1().Nodes().Delete(ctx, name, metav1.DeleteOptions{})
}

func (n *nodeFactory) GetNode(ctx context.Context, client kubernetes.Clientset, name string) (*corev1.Node, error) {
	return client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (n *nodeFactory) GetNodes(ctx context.Context, client kubernetes.Clientset) (*corev1.NodeList, error) {
	return client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

type node corev1.Node

func (n *nodeFactory) NewNode(opts ...ResourceOption) *corev1.Node {
	newNode := &node{}

	for _, opt := range opts {
		opt(newNode)
	}
	return (*corev1.Node)(newNode)
}
func (n *nodeFactory) NewNodeBatch(name, cpu, memory, pods string, count int) []*corev1.Node {
	nodes := make([]*corev1.Node, count)
	for i := 0; i < count; i++ {
		_name := fmt.Sprintf("%s-%d", name, i)
		n := n.NewNode(NodeWithStatus(_name, cpu, memory, pods))
		nodes[i] = n
	}
	return nodes
}

// NodeWithStatus sets the status of the node
func NodeWithStatus(name string, cpu string, memory string, pod string) ResourceOption {
	return func(obj ResourceAdapter) {
		if innerNode, ok := obj.(*node); ok {
			innerNode.Name = name
			innerNode.Status.Allocatable = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
				corev1.ResourcePods:   resource.MustParse(pod),
			}
			innerNode.Status.Capacity = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
				corev1.ResourcePods:   resource.MustParse(pod),
			}
		}
	}
}
