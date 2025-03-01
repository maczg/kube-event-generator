package run

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var scenarioPath string

var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run the event generator",
	Long:  `Run the event generator`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := main(cmd)
		return err
	},
}

func main(cmd *cobra.Command) error {
	sc, err := scenario.LoadFromFile(scenarioPath)
	if err != nil {
		return fmt.Errorf("error loading scenario file: %s", err)
	}
	logrus.Infof(sc.Info())

	km, _ := scheduler.NewKubeManager()

	sdl := scheduler.NewScheduler(
		scheduler.WithScenario(*sc),
		scheduler.WithKubeManager(km),
	)

	return sdl.Run()
}

func init() {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05",
		})
	Cmd.PersistentFlags().StringVarP(&scenarioPath, "scenario", "s", "scenario.yaml", "Path to the scenario file")
}
