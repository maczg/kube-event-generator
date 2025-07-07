package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/util"
)

// Options holds command options.
type Options struct {
	Namespace  string
	OutputMode string
	Timeout    time.Duration
}

// NewCommand creates the cluster command.
func NewCommand(log *logger.Logger) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Cluster management operations",
		Long:  "Commands for interacting with and managing Kubernetes clusters",
	}

	// Add sub-commands.
	cmd.AddCommand(
		newStatusCommand(opts, log),
		newResetCommand(opts, log),
		newInfoCommand(opts, log),
	)

	// Common flags.
	cmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Command timeout")
	cmd.PersistentFlags().StringVarP(&opts.Namespace, "namespace", "n", "", "Namespace to operate in")

	return cmd
}

// newStatusCommand creates the status sub-command.
func newStatusCommand(opts *Options, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check cluster status",
		Long:  "Display the current status of the Kubernetes cluster including nodes and resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
			defer cancel()

			return runStatus(ctx, opts, log)
		},
	}
}

// newResetCommand creates the reset sub-command.
func newResetCommand(opts *Options, log *logger.Logger) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset cluster state",
		Long:  "Remove all pods and optionally other resources from the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
			defer cancel()

			if !force {
				log.Warn("This will delete all pods in the cluster. Use --force to confirm.")
				return nil
			}

			return runReset(ctx, opts, log)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force reset without confirmation")

	return cmd
}

// newInfoCommand creates the info sub-command.
func newInfoCommand(opts *Options, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display cluster information",
		Long:  "Show detailed information about the cluster including version, nodes, and capacity",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
			defer cancel()

			return runInfo(ctx, opts, log)
		},
	}

	cmd.Flags().StringVarP(&opts.OutputMode, "output", "o", "table", "Output format (table, json, yaml)")

	return cmd
}

// runStatus executes the status command.
func runStatus(ctx context.Context, opts *Options, log *logger.Logger) error {
	clientset, err := util.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	log.Info("Checking cluster status...")

	// Get nodes.
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get pods.
	podList, err := clientset.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Display status.
	fmt.Printf("\nCluster Status:\n")
	fmt.Printf("===============\n")
	fmt.Printf("Nodes: %d\n", len(nodes.Items))

	for _, node := range nodes.Items {
		status := getNodeStatus(&node)
		fmt.Printf("  - %s: %s\n", node.Name, status)
	}

	fmt.Printf("\nPods: %d\n", len(podList.Items))

	// Count pods by status.
	statusCount := make(map[v1.PodPhase]int)
	for _, pod := range podList.Items {
		statusCount[pod.Status.Phase]++
	}

	for phase, count := range statusCount {
		fmt.Printf("  - %s: %d\n", phase, count)
	}

	return nil
}

// runReset executes the reset command.
func runReset(ctx context.Context, opts *Options, log *logger.Logger) error {
	clientset, err := util.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	log.Info("Resetting cluster state...")

	// Delete all pods.
	deletePolicy := metav1.DeletePropagationForeground
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	if opts.Namespace != "" {
		log.Infof("Deleting pods in namespace: %s", opts.Namespace)
		err = clientset.CoreV1().Pods(opts.Namespace).DeleteCollection(ctx, deleteOpts, metav1.ListOptions{})
	} else {
		log.Info("Deleting all pods in cluster")

		namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list namespaces: %w", err)
		}

		for _, ns := range namespaces.Items {
			if isSystemNamespace(ns.Name) {
				log.Debugf("Skipping system namespace: %s", ns.Name)
				continue
			}

			log.Debugf("Deleting pods in namespace: %s", ns.Name)

			if err := clientset.CoreV1().Pods(ns.Name).DeleteCollection(ctx, deleteOpts, metav1.ListOptions{}); err != nil {
				log.Warnf("Failed to delete pods in namespace %s: %v", ns.Name, err)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to delete pods: %w", err)
	}

	log.Info("Cluster reset completed")

	return nil
}

// runInfo executes the info command.
func runInfo(ctx context.Context, opts *Options, log *logger.Logger) error {
	clientset, err := util.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	log.Debug("Gathering cluster information...")

	// Get version.
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	// Get nodes.
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// Calculate total capacity.
	var totalCPU, totalMemory int64

	var totalPods int64

	for _, node := range nodes.Items {
		if cpu, ok := node.Status.Allocatable[v1.ResourceCPU]; ok {
			totalCPU += cpu.MilliValue()
		}

		if mem, ok := node.Status.Allocatable[v1.ResourceMemory]; ok {
			totalMemory += mem.Value()
		}

		if pods, ok := node.Status.Allocatable[v1.ResourcePods]; ok {
			totalPods += pods.Value()
		}
	}

	// Display information.
	switch opts.OutputMode {
	case "json":
		// TODO: Implement JSON output.
		return fmt.Errorf("JSON output not yet implemented")
	case "yaml":
		// TODO: Implement YAML output.
		return fmt.Errorf("YAML output not yet implemented")
	default:
		fmt.Printf("\nCluster Information:\n")
		fmt.Printf("===================\n")
		fmt.Printf("Kubernetes Version: %s\n", version.GitVersion)
		fmt.Printf("Platform: %s/%s\n", version.Platform, version.GoVersion)
		fmt.Printf("\nNodes: %d\n", len(nodes.Items))
		fmt.Printf("Total Allocatable Resources:\n")
		fmt.Printf("  CPU: %.2f cores\n", float64(totalCPU)/1000)
		fmt.Printf("  Memory: %.2f GB\n", float64(totalMemory)/(1024*1024*1024))
		fmt.Printf("  Pods: %d\n", totalPods)

		fmt.Printf("\nNode Details:\n")

		for _, node := range nodes.Items {
			fmt.Printf("\n  %s:\n", node.Name)
			fmt.Printf("    Status: %s\n", getNodeStatus(&node))
			fmt.Printf("    Roles: %s\n", getNodeRoles(&node))
			fmt.Printf("    Version: %s\n", node.Status.NodeInfo.KubeletVersion)
			fmt.Printf("    OS: %s/%s\n", node.Status.NodeInfo.OperatingSystem, node.Status.NodeInfo.Architecture)
		}
	}

	return nil
}

// Helper functions.

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
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}

	for _, ns := range systemNamespaces {
		if name == ns {
			return true
		}
	}

	return false
}

// Cmd is deprecated, use NewCommand instead.
var Cmd = NewCommand(logger.Default())
