package run

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/simulation"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sync"
)

var (
	scenarioFile string
	resetCluster bool
)

var Cmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"r"},
	Short:   "Run a simulation scenario",
	Long:    `Run a simulation`,
	Run: func(cmd *cobra.Command, args []string) {
		scn, err := scenario.LoadYaml(scenarioFile)
		if err != nil {
			logrus.Fatalf("Failed to load scenario file: %v", err)
		}

		clientset, cfg, err := utils.MakeClientSet()
		if err != nil {
			logrus.Fatalf("Failed to create clientset: %v", err)
		}

		mgr := utils.NewKubernetesManager(clientset, cfg)
		sim := simulation.New(scn, mgr)

		if resetCluster {
			logrus.Infoln("resetting cluster")
			if err := sim.ResetCluster(); err != nil {
				logrus.Fatalf("Failed to reset cluster: %v", err)
			}
			if err := sim.MakeCluster(); err != nil {
				logrus.Fatalf("Failed to create cluster: %v", err)
			}
		}

		mgr.DescribeCluster()
		var wg sync.WaitGroup
		wg.Add(1)
		done := make(chan struct{})

		go func() {
			defer wg.Done()
			if err := sim.Start(); err != nil {
				logrus.Errorln(err)
			}
			close(done)
		}()

		// Wait for either simulation completion or interrupt signal
		select {
		case <-done:
			logrus.Info("simulation completed")
		case <-utils.GetStopChan():
			logrus.Info("simulation interrupted")
			sim.Stop()
		}

		wg.Wait()
	},
}

func init() {
	Cmd.Flags().StringVarP(&scenarioFile, "scenario", "s", "scenario.yaml", "Scenario file to run")
	Cmd.Flags().BoolVarP(&resetCluster, "reset", "r", false, "Reset the cluster before running the simulation")
}
