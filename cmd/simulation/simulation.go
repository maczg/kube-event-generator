package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/kubernetes"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/simulation"
	"github.com/spf13/cobra"
	"strings"
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

// newStartCommand create the start sub-command.
func newStartCommand(log *logger.Logger) *cobra.Command {
	var scenarioFile string
	var saveMetrics bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a simulation",
		Long:  "Start a simulation based on a predefined scenario",
		RunE: func(cmd *cobra.Command, args []string) error {

			scenario, err := simulation.LoadFromYaml(scenarioFile)
			if err != nil {
				log.Errorf("failed to load scenario from file %s: %v", scenarioFile, err)
				return err
			}

			clientset, err := kubernetes.GetClientset()
			if err != nil {
				return nil
			}
			sim := simulation.NewSimulation(scenario, clientset, kubernetes.NewHTTPKubeSchedulerManager("http://localhost:1212"), log)
			if err := sim.Start(cmd.Context()); err != nil {
				log.Errorf("failed to start simulation: %v", err)
				return err
			}
			if saveMetrics {
				stats := sim.GetStats()
				err = stats.ExportCSV("results/" + strings.ReplaceAll(strings.ToLower(sim.GetID()), " ", "_"))
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&scenarioFile, "scenario", "scenario.yaml", "Path to the scenario file (YAML format)")
	cmd.Flags().BoolVar(&saveMetrics, "metrics", true, "Enable metrics")
	return cmd
}
