package cluster

import (
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/spf13/cobra"
)

const (
	// DefaultKubeConfigPath is the default path to the kubeconfig file
	DefaultKubeConfigPath = "~/.kube/config"
)

var Cmd = &cobra.Command{
	Use:     "cluster",
	Aliases: []string{"c"},
	Short:   "Cluster operations",
	Long:    `Cluster operations`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var resetCmd = &cobra.Command{
	Use:     "reset",
	Aliases: []string{"r"},
	Short:   "Reset the cluster",
	Long:    `Reset the cluster`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(logger.LevelInfo, "cluster-reset")
		clientset, rest, err := utils.MakeClientSet()
		if err != nil {
			return err
		}
		logger.Info("Resetting the cluster %s", rest.Host)
		mgr := utils.NewKubernetesManager(clientset, rest)
		if err = mgr.ResetNodes(); err != nil {
			return err
		}
		logger.Info("Cluster reset")
		return nil
	},
}

func init() {
	Cmd.Flags().StringP("kube-config", "k", DefaultKubeConfigPath, "Path to kubeconfig file")
	Cmd.AddCommand(resetCmd)
}
