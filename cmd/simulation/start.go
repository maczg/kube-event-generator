package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/kubeutils"
	"github.com/maczg/kube-event-generator/pkg/logger"
	simpkg "github.com/maczg/kube-event-generator/pkg/simulation"
	"github.com/spf13/cobra"
	"strings"
)

// newStartCommand create the start sub-command.
func newStartCommand(log *logger.Logger) *cobra.Command {
	var scenarioFile string
	var saveMetrics bool
	var createCluster bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a simulation",
		Long:  "Start a simulation based on a predefined scenario",
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario, err := simpkg.LoadFromYaml(scenarioFile)
			if err != nil {
				log.Errorf("failed to load scenario from file %s: %v", scenarioFile, err)
				return err
			}

			clientset, err := kubeutils.GetClientset()
			if err != nil {
				return nil
			}

			if createCluster && (scenario.Cluster.Nodes != nil && len(scenario.Cluster.Nodes) > 0) {
				log.Info("creating a new cluster...")
				if err = kubeutils.ResetCluster(); err != nil {
					log.Errorf("failed to reset cluster: %v", err)
					return err
				}
				if err = kubeutils.CreateClusterWithNodes(scenario.Cluster.Nodes); err != nil {
					log.Errorf("failed to create cluster with nodes: %v", err)
					return err
				} else {
					log.Infof("cluster created with %d nodes.", len(scenario.Cluster.Nodes))
				}
				log.Info("Cluster created successfully.")
			}

			sim := simpkg.NewSimulation(scenario, clientset, kubeutils.NewHTTPKubeSchedulerManager("http://localhost:1212"), log)
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
	cmd.Flags().BoolVar(&createCluster, "create-cluster", false, "Create a new cluster before starting the simulation")
	return cmd
}
