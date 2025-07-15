package cache

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// NodeStore It is updated by the Store.
type NodeStore struct {
	Node        *v1.Node
	Capacity    v1.ResourceList
	Allocatable v1.ResourceList

	// Pods represents the pods running on the node.
	RunningPods map[Key]*v1.Pod
	// Allocated represents the total amount of resources requested by all pods running on the node.
	Allocated      v1.ResourceList
	AllocatedRatio map[v1.ResourceName]float64
}

// NewNodeStore creates a new NodeStore object.
func NewNodeStore(node *v1.Node) *NodeStore {
	ni := &NodeStore{
		Node:           node,
		Capacity:       node.Status.Capacity,
		Allocatable:    node.Status.Allocatable,
		RunningPods:    make(map[Key]*v1.Pod),
		Allocated:      make(v1.ResourceList),
		AllocatedRatio: make(map[v1.ResourceName]float64),
	}
	ni.UpdateAllocated()

	return ni
}

func (ns *NodeStore) Copy() NodeStore {
	return *ns
}

func (ns *NodeStore) UpdateNodeSpec(newNode *v1.Node) {
	ns.Node = newNode
	ns.Allocatable = newNode.Status.Allocatable
	ns.Capacity = newNode.Status.Capacity
	ns.UpdateAllocated()
}

// UpdateAllocated updates the allocated resources based on the pods running on the node.
func (ns *NodeStore) UpdateAllocated() {
	requested := make(v1.ResourceList)

	for _, pod := range ns.RunningPods {
		for _, container := range pod.Spec.Containers {
			for resourceType, quantity := range container.Resources.Requests {
				if current, ok := requested[resourceType]; !ok {
					requested[resourceType] = quantity.DeepCopy()
				} else {
					current.Add(quantity)
					requested[resourceType] = current
				}
			}
		}
	}

	ns.Allocated = requested

	if len(ns.Allocated) == 0 {
		for _, res := range []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory} {
			ns.Allocated[res] = *resource.NewQuantity(0, resource.DecimalSI)
			ns.AllocatedRatio[res] = 0
		}

		return
	}

	for resourceType, quantity := range ns.Allocated {
		qty := quantity.DeepCopy()
		if allocatable, ok := ns.Allocatable[resourceType]; ok && allocatable.MilliValue() != 0 {
			ns.AllocatedRatio[resourceType] = float64(qty.MilliValue()) / float64(allocatable.MilliValue())
		}
	}
}

func (ns *NodeStore) GetAllocated(resource ...v1.ResourceName) v1.ResourceList {
	if len(resource) == 0 {
		return ns.Allocated
	}

	allocated := make(v1.ResourceList)

	for _, r := range resource {
		if _, ok := ns.Allocated[r]; !ok {
			continue
		}

		allocated[r] = ns.Allocated[r]
	}

	return allocated
}

// GetAllocatedRatio returns the ratio of allocated resources to allocatable resources.
func (ns *NodeStore) GetAllocatedRatio(resource ...v1.ResourceName) map[v1.ResourceName]float64 {
	if len(resource) == 0 {
		return ns.AllocatedRatio
	}

	allocatedRatio := make(map[v1.ResourceName]float64)

	for _, r := range resource {
		if _, ok := ns.Allocated[r]; !ok {
			continue
		}

		allocatedRatio[r] = ns.AllocatedRatio[r]
	}

	return allocatedRatio
}

// GetFree returns the total capacity of the node.
func (ns *NodeStore) GetFree() v1.ResourceList {
	free := make(v1.ResourceList)

	for resourceType, allocatable := range ns.Allocatable {
		if allocated, ok := ns.Allocated[resourceType]; ok {
			free[resourceType] = allocatable.DeepCopy()
			_free := free[resourceType]
			_free.Sub(allocated)
			free[resourceType] = _free
		} else {
			free[resourceType] = allocatable.DeepCopy()
		}
	}

	return free
}

func (ns *NodeStore) addPod(pod *v1.Pod) {
	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return
	}

	ns.RunningPods[NewKey(pod)] = pod
	ns.UpdateAllocated()
}

func (ns *NodeStore) deletePod(pod *v1.Pod) {
	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return
	}

	delete(ns.RunningPods, NewKey(pod))
	ns.UpdateAllocated()
}

func (ns *NodeStore) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[%s|cpu %s|mem %s] ", ns.Node.Name, ns.Allocated.Cpu().String(), ns.Allocated.Memory().String()))

	return builder.String()
}
