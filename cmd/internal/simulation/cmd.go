package simulation

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/simulator"
	"github.com/maczg/kube-event-generator/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"time"
)

var (
	reset        bool
	scenarioFile string
	outputDir    string
	saveMetrics  bool
)

func init() {
	Cmd.AddCommand(startCmd)
	startCmd.Flags().BoolVarP(&reset, "reset", "r", false, "Reset the cluster before starting the simulation")
	startCmd.Flags().StringVarP(&scenarioFile, "scenario", "s", "", "Path to the scenario file")
	startCmd.Flags().BoolVarP(&saveMetrics, "save-metrics", "m", true, "Save metrics to the output directory")
	startCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "results", "Path to the output directory")
	startCmd.MarkFlagRequired("scenario")
}

var Cmd = &cobra.Command{
	Use:     "simulation",
	Aliases: []string{"sim"},
	Short:   "Simulation commands",
	Long:    `Simulation commands`,
}

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start a simulation",
	Long:    `Start a simulation`,
	Example: `simulation start`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var clientset *kubernetes.Clientset
		var scn *scenario.Scenario
		dirName := fmt.Sprintf("%s/%s_%s", outputDir, scn.Metadata.Name, time.Now().Format("2006-01-02_15-04-05"))
		if err := util.CreateDir(dirName); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		if s, err := scenario.LoadFromYaml(scenarioFile); err != nil {
			return fmt.Errorf("failed to load scenario: %v", err)
		} else {
			scn = s
		}
		if c, err := util.GetClientset(); err != nil {
			return fmt.Errorf("failed to get clientset: %v", err)
		} else {
			clientset = c
		}

		if reset {
			if err := resetCluster(scn, clientset); err != nil {
				return fmt.Errorf("failed to reset cluster: %v", err)
			}
		}

		scn.Describe()

		scdl := scheduler.New()
		for _, event := range scn.Events.Pods {
			event.SetClientset(clientset)
			event.SetScheduler(scdl)
		}

		simulation := simulator.NewSimulation(scn, clientset, scdl)
		simulation.LoadEvents()

		if err := simulation.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start simulation: %v", err)
		}

		if saveMetrics {
			logrus.Infof("saving metrics to %s", outputDir)
			stats := simulation.GetStats()
			if err := stats.ExportCSV(dirName); err != nil {
				return fmt.Errorf("failed to export metrics: %v", err)
			}
		}
		return nil
	},
}

func resetCluster(scn *scenario.Scenario, clientset *kubernetes.Clientset) error {
	logrus.Warn("requesting cluster reset")
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}
	for _, pod := range pods.Items {
		if err := clientset.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
			logrus.Errorf("failed to delete pod %s: %v", pod.Name, err)
		}
	}
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}
	for _, node := range nodes.Items {
		if err = clientset.CoreV1().Nodes().Delete(context.Background(), node.Name, metav1.DeleteOptions{}); err != nil {
			logrus.Errorf("failed to delete node %s: %v", node.Name, err)
		}
	}

	if scn.Cluster != nil {
		return scn.Cluster.Create(clientset)
	} else {
		logrus.Warn("cluster is not set in scenario, skipping cluster creation")
	}
	return nil
}

var randomRun = &cobra.Command{
	Use:   "start-random",
	Short: "Start a random simulation",
	Long:  `Start a random simulation`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		clientset, err := util.GetClientset()
		if err != nil {
			return err
		}
		scn := scenario.NewScenario()
		scn.Metadata.Name = "random-run"
		scdl := scheduler.New()

		for i := 0; i < 10; i++ {
			pod := util.PodFactory.NewPod(
				util.PodWithMetadata(fmt.Sprintf("pod-%d", i), "default", nil, nil),
				util.PodWithContainer("server", "nginx", fmt.Sprintf("%dm", rand.IntnRange(100, 1000)), fmt.Sprintf("%dMi", rand.IntnRange(100, 1000))))
			event := scenario.NewPodEvent(pod.Name, time.Duration(rand.IntnRange(5, 10))*time.Second, time.Duration(rand.IntnRange(5, 10)), pod, clientset, scdl)
			scn.Events.Pods = append(scn.Events.Pods, event)
		}
		simulation := simulator.NewSimulation(scn, clientset, scdl)
		ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
		if err := simulation.Start(ctx); err != nil {
			return fmt.Errorf("failed to start simulation: %v", err)
		}
		return nil
	},
}
