package cmd

import (
	"fmt"
	"github.com/maczg/kube-event-generator/cmd/cluster"
	"github.com/maczg/kube-event-generator/cmd/simulation"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"runtime"
	"strings"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "keg",
	Short: "A tool to generate Kubernetes events for testing purposes",
	Long:  `kube-event-generator is a tool to generate Kubernetes events for testing purposes.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05.000",
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				return "", fmt.Sprintf("%s:%d", strings.Split(f.File, "/")[len(strings.Split(f.File, "/"))-1], f.Line)
			},
		})
		logrus.SetReportCaller(true)
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
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
	rootCmd.AddCommand(cluster.Cmd)
	rootCmd.AddCommand(simulation.Cmd)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}
