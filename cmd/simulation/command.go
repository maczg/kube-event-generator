package simulation

import (
	"context"
	"time"

	"github.com/maczg/kube-event-generator/pkg/kubeutils"
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
	var schedulerURL string
	var timeout time.Duration

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

			clientset, err := kubeutils.GetClientset()
			if err != nil {
				log.Errorf("failed to create kubernetes client: %v", err)
				return err
			}

			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			sim := simulation.NewSimulation(scenario, clientset, kubeutils.NewHTTPKubeSchedulerManager(schedulerURL), log)
			if err := sim.Start(ctx); err != nil {
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
	cmd.Flags().BoolVar(&saveMetrics, "metrics", true, "Export metrics as CSV under results/")
	cmd.Flags().StringVar(&schedulerURL, "scheduler-url", "http://localhost:1212/api/v1/schedulerconfiguration", "URL of the kube-scheduler-simulator configuration API, used by scheduler events")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Stop the simulation after this wall-clock time even if pods are still pending (0 means no limit)")
	return cmd
}
