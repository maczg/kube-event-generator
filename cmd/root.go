package cmd

import (
	"fmt"
	"github.com/maczg/kube-event-generator/cmd/simulation"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var defaultSimulationName = func() string {
	name := fmt.Sprintf("sim-%s", time.Now().Format("15:04:05"))
	return name
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "kube-event-generator",
	Aliases: []string{"keg"},
	Short:   "A tool to generate Kubernetes events for testing purposes",
	Long:    `kube-event-generator is a tool to generate Kubernetes events for testing purposes.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(simulation.SimulationCmd)
	rootCmd.PersistentFlags().StringP("name", "n", defaultSimulationName(), "Name of the simulation")
}
