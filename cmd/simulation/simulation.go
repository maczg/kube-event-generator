package simulation

import (
	"github.com/maczg/kube-event-generator/cmd/simulation/generate"
	"github.com/maczg/kube-event-generator/cmd/simulation/run"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "simulation",
	Aliases: []string{"sim"},
	Short:   "Simulation commands",
	Long:    `Simulation commands`,
}

func init() {
	Cmd.AddCommand(generate.Cmd)
	Cmd.AddCommand(run.Cmd)
}
