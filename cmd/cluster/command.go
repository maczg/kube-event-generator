package cluster

import (
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
