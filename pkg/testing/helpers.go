package testing

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestContext creates a test context with timeout.
func TestContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	t.Cleanup(func() {
		cancel()
	})

	return ctx, cancel
}

// NewFakeClientset creates a fake Kubernetes clientset for testing.
func NewFakeClientset() *fake.Clientset {
	return fake.NewSimpleClientset()
}

// NewTestPod creates a test pod with specified resources.
func NewTestPod(name, namespace string, cpuMillis, memoryMi int64) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    *resource.NewMilliQuantity(cpuMillis, resource.DecimalSI),
							v1.ResourceMemory: *resource.NewQuantity(memoryMi*1024*1024, resource.BinarySI),
						},
					},
				},
			},
		},
	}
}

// NewTestNode creates a test node with specified capacity.
func NewTestNode(name string, cpuCores int64, memoryGi int64) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(cpuCores, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(memoryGi*1024*1024*1024, resource.BinarySI),
				v1.ResourcePods:   *resource.NewQuantity(110, resource.DecimalSI),
			},
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(cpuCores, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(memoryGi*1024*1024*1024, resource.BinarySI),
				v1.ResourcePods:   *resource.NewQuantity(110, resource.DecimalSI),
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

// AssertEventually asserts that a condition is met within the timeout.
func AssertEventually(t *testing.T, condition func() bool, timeout, interval time.Duration, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(interval)
	}

	t.Fatalf("condition not met within %v: %s", timeout, msg)
}

// AssertNoError asserts that an error is nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError asserts that an error is not nil.
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error but got nil: %s", msg)
	}
}
