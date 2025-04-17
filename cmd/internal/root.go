package internal

import (
	"fmt"
	"github.com/maczg/kube-event-generator/cmd/internal/cluster"
	"github.com/maczg/kube-event-generator/cmd/internal/scenario"
	"github.com/maczg/kube-event-generator/cmd/internal/simulation"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"runtime"
	"strings"
)

var verbose bool

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.AddCommand(cluster.Cmd)
	rootCmd.AddCommand(simulation.Cmd)
	rootCmd.AddCommand(scenario.Cmd)
}

var rootCmd = &cobra.Command{
	Use:              "keg",
	Aliases:          []string{"kube-event-generator", "keg"},
	Short:            "Kubernetes Event Generator",
	Long:             `Kubernetes Event Generator is a tool to simulate events in a Kubernetes cluster.`,
	PersistentPreRun: persistentPreRun,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func persistentPreRun(cmd *cobra.Command, args []string) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf("[%s:%d]", strings.Split(f.File, "/")[len(strings.Split(f.File, "/"))-1], f.Line)
		},
	})
	logrus.SetReportCaller(true)
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
