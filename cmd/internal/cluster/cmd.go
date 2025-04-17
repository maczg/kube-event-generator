package cluster

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/cache"
	"github.com/maczg/kube-event-generator/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var exportMetrics bool

func init() {
	Cmd.AddCommand(PodResetCmd)
	Cmd.AddCommand(NodeResetCmd)
	Cmd.AddCommand(WatchCmd)
	WatchCmd.Flags().BoolVarP(&exportMetrics, "export-metrics", "e", false, "Export metrics to CSV")
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Cluster commands",
	Long:  `Cluster commands`,
}

var PodResetCmd = &cobra.Command{
	Use:   "pod-reset",
	Short: "Reset the cluster pods",
	Long:  `Reset the cluster pods `,
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset, err := util.GetClientset()
		if err != nil {
			return err
		}
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pod := range pods.Items {
			if err := clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
				logrus.Errorf("failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
		return nil

	},
}

// NodeResetCmd is a command to reset the cluster nodes
var NodeResetCmd = &cobra.Command{
	Use:   "node-reset",
	Short: "Reset the cluster nodes",
	Long:  `Reset the cluster nodes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset, err := util.GetClientset()
		if err != nil {
			return err
		}
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, node := range nodes.Items {
			if err := clientset.CoreV1().Nodes().Delete(context.TODO(), node.Name, metav1.DeleteOptions{}); err != nil {
				logrus.Errorf("failed to delete node %s: %v", node.Name, err)
			}
		}
		return nil
	},
}

var WatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch the cluster events",
	Long:  `Watch the cluster events`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		clientset, err := util.GetClientset()
		if err != nil {
			return
		}
		store := cache.NewStore(clientset)
		store.Start()
		go func() {
			store.WatchEvery(1)
		}()
		util.WaitStopAndExecute(func() {
			store.Stop()
			logrus.Info("stop watch cluster events")
			if exportMetrics {
				stats := store.GetStats()
				dir := fmt.Sprintf("output/stats/%s", time.Now().Format("2006-01-02_15-04-05"))
				if err := stats.ExportCSV(dir); err != nil {
					logrus.Errorf("failed to export metrics: %v", err)
				}
			}
		})
		return
	},
}
