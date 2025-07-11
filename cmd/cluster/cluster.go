package cluster

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/kubernetes"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/cobra"
)

// NewCommand creates the cluster command.
func NewCommand(log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Cluster management operations",
		Long:  "Commands for interacting with and managing Kubernetes clusters",
	}

	// Add sub-commands.
	cmd.AddCommand(
		newStatusCommand(log),
		newResetCommand(log),
	)

	return cmd
}

// newStatusCommand creates the status sub-command.
func newStatusCommand(log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check cluster status",
		Long:  "Display the current status of the Kubernetes cluster including nodes, pods and resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			return kubernetes.ClusterStatus(ctx, log)
		},
	}
}

// newResetCommand creates the reset sub-command.
func newResetCommand(log *logger.Logger) *cobra.Command {
	var force bool
	var pods bool
	var nodes bool
	var scheduler bool
	var schedulerUrl string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset cluster state",
		Long:  "Remove all pods and optionally other resources from the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if !force {
				log.Warn("This will delete all pods in the cluster. Use --force to confirm.")
				return nil
			}
			return runReset(ctx, log, pods, nodes, scheduler, schedulerUrl)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force reset without confirmation")
	cmd.Flags().StringVar(&schedulerUrl, "scheduler-url", "http://localhost:1212/api/v1/schedulerconfiguration", "URL of the scheduler configuration API")
	cmd.Flags().BoolVar(&pods, "pods", true, "Reset pods in the cluster")
	cmd.Flags().BoolVar(&nodes, "nodes", false, "Reset nodes in the cluster (not implemented)")
	cmd.Flags().BoolVar(&scheduler, "scheduler", true, "Reset scheduler weights and configuration")

	return cmd
}

// runReset executes the reset command.
func runReset(ctx context.Context, log *logger.Logger, pods, nodes, scheduler bool, schedulerUrl string) error {
	if pods {
		if err := kubernetes.ResetPods(ctx, log); err != nil {
			log.Errorf("Failed to reset pods: %v", err)
			return err
		}
		log.Info("Successfully reset pods in the cluster.")
	}

	if nodes {
		log.Warn("Resetting nodes is not implemented yet.")
		return fmt.Errorf("resetting nodes is not implemented yet")
	}

	if scheduler {
		if err := kubernetes.ResetKubeSchedulerWeights(ctx, log, schedulerUrl); err != nil {
			log.Errorf("Failed to reset scheduler weights: %v", err)
		}
		log.Info("Successfully reset scheduler to defaults.")
	}
	return nil
}

// ClusterCmd is deprecated, use NewCommand instead.
var ClusterCmd = NewCommand(logger.Default())
