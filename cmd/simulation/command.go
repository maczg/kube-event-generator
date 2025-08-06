package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/cobra"
)

// NewCommand creates the cluster command.
func NewCommand(log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "simulation",
		Aliases: []string{"sim", "s"},
		Short:   "Simulation commands",
		Long:    "Commands for managing and running simulations",
	}
	// sub-commands.
	cmd.AddCommand(
		newStartCommand(log),
	)
	return cmd
}
