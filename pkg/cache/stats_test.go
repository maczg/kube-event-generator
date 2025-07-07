package cache

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	nodeCpu    = resource.MustParse("1")
	nodeMemory = resource.MustParse("1Gi")

	pod1Cpu    = resource.MustParse("100m")
	pod1Memory = resource.MustParse("200Mi")
	pod2Cpu    = resource.MustParse("200m")
	pod2Memory = resource.MustParse("300Mi")
)

func createTestNode(name string, cpu, memory resource.Quantity) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(name + "uid"),
		},
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU:    cpu,
				v1.ResourceMemory: memory,
			},
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    cpu,
				v1.ResourceMemory: memory,
			},
		},
	}
}

func createTestPod(name, nodeName string, cpu, memory resource.Quantity) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(name + "uid"),
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
			Containers: []v1.Container{
				{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    cpu,
							v1.ResourceMemory: memory,
						},
					},
				},
			},
		},
	}
}

func TestClusterStats_RegisterAllocatedResourceSample(t *testing.T) {
	node := createTestNode("node1", nodeCpu, nodeMemory)
	nodeInfo := NewNodeStore(node)

	nodeInfo.addPod(createTestPod("pod1", "node1", pod1Cpu, pod1Memory))
	nodeInfo.addPod(createTestPod("pod2", "node1", pod2Cpu, pod2Memory))

	stats := NewStats()
	stats.UpdateHistory(*nodeInfo)

	key := NewKey(node)
	if len(stats.AllocationHistory[key]) == 0 {
		t.Errorf("expected AllocationHistory to have records, but got 0")
	}

	if len(stats.AllocationRatioHistory[key]) == 0 {
		t.Errorf("expected AllocationRatioHistory to have records, but got 0")
	}

	record := stats.AllocationHistory[key][0]
	expectedCpu := pod1Cpu.DeepCopy()
	expectedCpu.Add(pod2Cpu)

	expectedMemory := pod1Memory.DeepCopy()
	expectedMemory.Add(pod2Memory)

	if record.Value.Cpu().Cmp(expectedCpu) != 0 {
		t.Errorf("expected allocated CPU to be %s, but got %s", expectedCpu.String(), record.Value.Cpu().String())
	}

	if record.Value.Memory().Cmp(expectedMemory) != 0 {
		t.Errorf("expected allocated Memory to be %s, but got %s", expectedMemory.String(), record.Value.Memory().String())
	}

	ratioRecord := stats.AllocationRatioHistory[key][0]
	expectedCpuRatio := float64(expectedCpu.MilliValue()) / float64(nodeCpu.MilliValue())
	expectedMemRatio := float64(expectedMemory.MilliValue()) / float64(nodeMemory.MilliValue())

	if ratioRecord.Value[v1.ResourceCPU] != expectedCpuRatio {
		t.Errorf("expected allocated CPU ratio to be %f, but got %f", expectedCpuRatio, ratioRecord.Value[v1.ResourceCPU])
	}

	if ratioRecord.Value[v1.ResourceMemory] != expectedMemRatio {
		t.Errorf("expected allocated Memory ratio to be %f, but got %f", expectedMemRatio, ratioRecord.Value[v1.ResourceMemory])
	}
}
