package cache

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var pod1Resource = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("100m"),
	v1.ResourceMemory: resource.MustParse("200Mi"),
}

var pod2Resource = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("200m"),
	v1.ResourceMemory: resource.MustParse("300Mi"),
}

var nodeResource = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("1"),
	v1.ResourceMemory: resource.MustParse("1Gi"),
	v1.ResourcePods:   resource.MustParse("110"),
}

var testNode = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "node1",
		UID:  "nodeuid1",
	},
	Status: v1.NodeStatus{
		Capacity:    nodeResource,
		Allocatable: nodeResource,
	},
}

var testPod1 = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "pod1",
		UID:  "uid1",
	},
	Spec: v1.PodSpec{
		NodeName: testNode.Name,
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: pod1Resource,
				},
			},
		},
	},
}

var testPod2 = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "pod2",
		UID:  "uid2",
	},
	Spec: v1.PodSpec{
		NodeName: testNode.Name,
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: pod2Resource,
				},
			},
		},
	},
}

func TestNodeStatus_Update(t *testing.T) {
	nodeStatus := NewNodeStore(testNode)
	nodeStatus.addPod(testPod1)
	nodeStatus.addPod(testPod2)

	expectedCPU := resource.MustParse("300m")
	expectedMemory := resource.MustParse("500Mi")

	allocatedCPU := nodeStatus.GetAllocated()[v1.ResourceCPU]
	allocatedMemory := nodeStatus.GetAllocated()[v1.ResourceMemory]

	if allocatedCPU.Cmp(expectedCPU) != 0 {
		t.Errorf("expected CPU %s, but got %s", expectedCPU.String(), nodeStatus.Allocated.Cpu().String())
	}

	if allocatedMemory.Cmp(expectedMemory) != 0 {
		t.Errorf("expected Memory %s, but got %s", expectedMemory.String(), nodeStatus.Allocated.Memory().String())
	}

	nodeCPUCapacity := nodeStatus.Allocatable[v1.ResourceCPU]
	nodeMemoryCapacity := nodeStatus.Allocatable[v1.ResourceMemory]

	expectedCPURatio := float64(expectedCPU.MilliValue()) / float64(nodeCPUCapacity.MilliValue())
	expectedMemoryRatio := float64(expectedMemory.MilliValue()) / float64(nodeMemoryCapacity.MilliValue())

	ratio := nodeStatus.GetAllocatedRatio()
	if ratio[v1.ResourceCPU] != expectedCPURatio {
		t.Errorf("expected CPU ratio %f, but got %f", expectedCPURatio, ratio[v1.ResourceCPU])
	}

	if ratio[v1.ResourceMemory] != expectedMemoryRatio {
		t.Errorf("expected Memory ratio %f, but got %f", expectedMemoryRatio, ratio[v1.ResourceMemory])
	}
}
