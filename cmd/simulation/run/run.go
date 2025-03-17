package run

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/simulation"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"r"},
	Short:   "Run a simulation scenario",
	Long:    `Run a simulation`,
	Run: func(cmd *cobra.Command, args []string) {
		scenarioFile, _ := cmd.Flags().GetString("scenario")
		scn, err := scenario.LoadYaml(scenarioFile)
		if err != nil {
			logrus.Fatalf("Failed to load scenario file: %v", err)
		}

		clientset, err := utils.MakeClientSet()
		if err != nil {
			logrus.Fatalf("Failed to create clientset: %v", err)
		}
		mgr := utils.NewKubernetesManager(clientset)
		sim := simulation.New(scn, mgr)

		logrus.Fatalln(sim.Start())

	},
}

func init() {
	Cmd.Flags().StringP("scenario", "s", "scenario.yaml", "Scenario file to run")
}
