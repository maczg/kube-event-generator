package scenario

import (
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/cobra"
)

// NewCommand creates the scenario command.
func NewCommand(log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scenario",
		Aliases: []string{"scn"},
		Short:   "Scenario management operations",
		Long:    "Commands for managing and executing scenarios in Kubernetes",
	}
	// Add sub-commands.
	cmd.AddCommand(
		newGenerateScenarioCommand(log),
	)
	return cmd
}
