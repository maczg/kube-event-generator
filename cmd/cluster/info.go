package cluster

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/spf13/cobra"
	"os"
)

var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"i"},
	Short:   "Get cluster info",
	Long:    `Get cluster info`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(logger.LevelInfo, "cluster-info")
		clientset, rest, err := utils.MakeClientSet()
		if err != nil {
			return err
		}
		logger.Info("Getting cluster info for %s", rest.Host)
		mgr := utils.NewKubernetesManager(clientset, rest)
		nodes, err := mgr.GetNodes(context.Background(), *clientset)
		if err != nil {
			logger.Error("Failed to get nodes: %v", err)
			return err
		}
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Node", "CPU (Capacity)", "Memory (Capacity)", "CPU (Allocatable)", "Memory (Allocatable)", "Conditions"})

		for _, node := range nodes.Items {
			conditions := ""
			for _, condition := range node.Status.Conditions {
				conditions += fmt.Sprintf("%s: %s\n", condition.Type, condition.Status)
			}
			t.AppendRow(table.Row{
				node.Name,
				fmt.Sprintf("%dm", node.Status.Capacity.Cpu().MilliValue()),
				fmt.Sprintf("%dMi", node.Status.Capacity.Memory().Value()/(1024*1024)),
				fmt.Sprintf("%dm", node.Status.Allocatable.Cpu().MilliValue()),
				fmt.Sprintf("%dMi", node.Status.Allocatable.Memory().Value()/(1024*1024)),
				conditions,
			})
			t.AppendSeparator()
		}

		t.Render()
		return nil
	},
}
