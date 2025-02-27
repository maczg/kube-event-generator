package run

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run the event generator",
	Long:  `Run the event generator`,
	Run: func(cmd *cobra.Command, args []string) {
		main(cmd)
	},
}

func main(cmd *cobra.Command) {
	scenarioPath, _ := cmd.Flags().GetString("scenario")
	sc, err := scenario.LoadFromFile(scenarioPath)
	if err != nil {
		logrus.Fatalln("error loading scenario file:", err)
	}
	logrus.Infof(sc.Info())

	km, _ := scheduler.NewKubeManager()
	sdl := scheduler.NewScheduler(
		scheduler.WithScenario(*sc),
		scheduler.WithKubeManager(km),
	)

	logrus.Infoln(sdl.Run())
}

func init() {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05",
		})
	Cmd.PersistentFlags().StringP("scenario", "s", "scenario.yaml", "Path to the scenario file")
	Cmd.PersistentFlags().StringP("deadline", "d", "30m", "Deadline for the scenario")
}
