package cluster

import (
	"context"
	"github.com/maczg/kube-event-generator/pkg/kubeutils"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/cobra"
)

// newStatusCommand creates the status sub-command.
func newStatusCommand(log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check cluster status",
		Long:  "Display the current status of the Kubernetes cluster including nodes, pods and resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			return kubeutils.ClusterStatus(ctx, log)
		},
	}
}
