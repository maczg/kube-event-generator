package simulation

import (
	"github.com/maczg/kube-event-generator/cmd/simulation/generate"
	"github.com/spf13/cobra"
)

var SimulationCmd = &cobra.Command{
	Use:     "simulation",
	Aliases: []string{"sim"},
	Short:   "Simulation commands",
	Long:    `Simulation commands`,
}

func init() {
	SimulationCmd.AddCommand(generate.Cmd)
}
