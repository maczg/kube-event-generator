package cmd

import (
	"github.com/maczg/kube-event-generator/cmd/run"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "kube-event-generator",
	Aliases: []string{"keg"},
	Short:   "k8s-event-generator is a tool to generate events in a k8s cluster",
	Long:    ``,
	// Run: func(cmd *cobra.Command, args []string) { },
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
	path := os.Getenv("HOME") + "/.kube/config"
	rootCmd.PersistentFlags().StringP("kubeconfig", "k", path, "Path to the kubeconfig file")
	rootCmd.AddCommand(run.Cmd)
}
