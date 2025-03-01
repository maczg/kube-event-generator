package cmd

import (
	"github.com/maczg/kube-event-generator/cmd/run"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var kubeconfig string
var defaultKubeconfigPath = os.Getenv("HOME") + "/.kube/config"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "kube-event-generator",
	Aliases: []string{"keg"},
	Short:   "k8s-event-generator is a tool to generate events in a k8s cluster",
	Long:    ``,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Errorf("Error executing command: %s", err)
		os.Exit(1)
	}
}

func init() {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05",
		})
	rootCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", defaultKubeconfigPath, "Path to the kubeconfig file")
	rootCmd.AddCommand(run.Cmd)
}
