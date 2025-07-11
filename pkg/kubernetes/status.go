package kubernetes

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ClusterStatus(ctx context.Context, log *logger.Logger) error {
	clientset, err := GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}
	log.Debug("Gathering cluster information...")

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get pods.
	podList, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var totalCPU, totalMemory, totalAllocatablePods int64

	for _, node := range nodes.Items {
		if cpu, ok := node.Status.Allocatable[v1.ResourceCPU]; ok {
			totalCPU += cpu.MilliValue()
		}

		if mem, ok := node.Status.Allocatable[v1.ResourceMemory]; ok {
			totalMemory += mem.Value()
		}

		if pods, ok := node.Status.Allocatable[v1.ResourcePods]; ok {
			totalAllocatablePods += pods.Value()
		}
	}

	fmt.Printf("\nCluster Information:\n")
	fmt.Printf("===================\n")
	fmt.Printf("Kubernetes Version: %s\n", version.GitVersion)
	fmt.Printf("Platform: %s/%s\n", version.Platform, version.GoVersion)
	fmt.Printf("\nNodes: %d\n", len(nodes.Items))
	fmt.Printf("Total Allocatable Resources:\n")
	fmt.Printf("  CPU: %.2f cores\n", float64(totalCPU)/1000)
	fmt.Printf("  Memory: %.2f GB\n", float64(totalMemory)/(1024*1024*1024))
	fmt.Printf("  Pods: %d\n", totalAllocatablePods)
	fmt.Printf("\nNode Details:\n")
	for _, node := range nodes.Items {
		fmt.Printf("\n  %s:\n", node.Name)
		fmt.Printf("    Status: %s\n", getNodeStatus(&node))
		fmt.Printf("    Roles: %s\n", getNodeRoles(&node))
		fmt.Printf("    Version: %s\n", node.Status.NodeInfo.KubeletVersion)
		fmt.Printf("    OS: %s/%s\n", node.Status.NodeInfo.OperatingSystem, node.Status.NodeInfo.Architecture)
	}
	fmt.Printf("===================\n")
	fmt.Printf("\nPods: %d\n", len(podList.Items))
	statusCount := make(map[v1.PodPhase]int)
	for _, pod := range podList.Items {
		statusCount[pod.Status.Phase]++
	}
	for phase, count := range statusCount {
		fmt.Printf("  - %s: %d\n", phase, count)
	}

	return nil
}

// getNodeStatus returns the status of a node.
func getNodeStatus(node *v1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			if condition.Status == v1.ConditionTrue {
				return "Ready"
			}

			return "NotReady"
		}
	}

	return "Unknown"
}

// getNodeRoles returns the roles of a node.
func getNodeRoles(node *v1.Node) string {
	roles := []string{}

	for label := range node.Labels {
		if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
			roles = append(roles, "control-plane")
		} else if label == "node-role.kubernetes.io/worker" {
			roles = append(roles, "worker")
		}
	}

	if len(roles) == 0 {
		return "none"
	}

	return fmt.Sprintf("%v", roles)
}

// isSystemNamespace checks if a namespace is a system namespace.
func isSystemNamespace(name string) bool {
	systemNs := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
	}

	for _, ns := range systemNs {
		if name == ns {
			return true
		}
	}
	return false
}
