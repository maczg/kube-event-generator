package simulation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/metrics"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/simulator"
	"github.com/maczg/kube-event-generator/pkg/util"
)

// Options holds command options.
type Options struct {
	ScenarioFile    string
	OutputDir       string
	MetricsFormat   string
	SchedulerClient string
	Timeout         time.Duration
	SaveMetrics     bool
	ClusterReset    bool
	DryRun          bool
	Watch           bool
}

// NewCommand creates the simulation command.
func NewCommand(log *logger.Logger) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:     "simulation",
		Aliases: []string{"sim"},
		Short:   "Simulation management commands",
		Long:    "Commands for running and managing Kubernetes event simulations",
	}

	// Add sub-commands.
	cmd.AddCommand(
		newStartCommand(opts, log),
		newStatusCommand(opts, log),
		newStopCommand(opts, log),
		newRandomCommand(opts, log),
	)

	return cmd
}

// newStartCommand creates the start sub-command.
func newStartCommand(opts *Options, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a simulation",
		Long: `Start a simulation using a scenario file.
The simulation will create pods and other events according to the scenario definition.`,
		Example: `  # Start a simulation
  keg simulation start -s scenario.yaml
  
  # Start with cluster reset
  keg simulation start -s scenario.yaml --cluster-reset
  
  # Start with custom output directory
  keg simulation start -s scenario.yaml -o ./my-results`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), opts, log)
		},
	}

	cmd.Flags().StringVarP(&opts.ScenarioFile, "scenario", "s", "", "Path to scenario file (required)")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", "results", "Output directory for metrics")
	cmd.Flags().BoolVarP(&opts.SaveMetrics, "save-metrics", "m", true, "Save simulation metrics")
	cmd.Flags().StringVar(&opts.MetricsFormat, "metrics-format", "csv", "Metrics format (csv, json)")
	cmd.Flags().BoolVar(&opts.ClusterReset, "cluster-reset", false, "Reset cluster before simulation")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "Simulation timeout (0 for no timeout)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Validate scenario without running")
	cmd.Flags().BoolVar(&opts.Watch, "watch", false, "Watch simulation progress")
	cmd.Flags().StringVar(&opts.SchedulerClient, "scheduler-url", "http://localhost:1212/api/v1/schedulerconfiguration", "Scheduler simulator URL")

	cmd.MarkFlagRequired("scenario")

	return cmd
}

// newStatusCommand creates the status sub-command.
func newStatusCommand(opts *Options, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check simulation status",
		Long:  "Display the current status of running simulations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.Context(), opts, log)
		},
	}
}

// newStopCommand creates the stop sub-command.
func newStopCommand(opts *Options, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop running simulations",
		Long:  "Stop all currently running simulations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd.Context(), opts, log)
		},
	}
}

// newRandomCommand creates the random sub-command.
func newRandomCommand(opts *Options, log *logger.Logger) *cobra.Command {
	var numPods int

	var duration time.Duration

	cmd := &cobra.Command{
		Use:   "random",
		Short: "Run a random simulation",
		Long:  "Run a quick simulation with randomly generated pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRandom(cmd.Context(), opts, numPods, duration, log)
		},
	}

	cmd.Flags().IntVar(&numPods, "num-pods", 10, "Number of pods to create")
	cmd.Flags().DurationVar(&duration, "duration", 30*time.Second, "Simulation duration")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", "results", "Output directory for metrics")
	cmd.Flags().BoolVarP(&opts.SaveMetrics, "save-metrics", "m", true, "Save simulation metrics")

	return cmd
}

// runStart executes the start command.
func runStart(ctx context.Context, opts *Options, log *logger.Logger) error {
	// Add request ID to context.
	ctx = logger.WithRequestID(ctx)

	// Load scenario.
	log.WithContext(ctx).WithFields(map[string]interface{}{
		"scenario_file": opts.ScenarioFile,
	}).Info("Loading scenario")

	scn, err := scenario.LoadFromYaml(opts.ScenarioFile)
	if err != nil {
		return fmt.Errorf("failed to load scenario: %w", err)
	}

	// Add simulation ID to context.
	simulationID := fmt.Sprintf("%s-%s", scn.Metadata.Name, time.Now().Format("2006-01-02_15-04-05"))
	ctx = logger.WithSimulationID(ctx, simulationID)

	log.WithContext(ctx).WithFields(map[string]interface{}{
		"scenario_name":    scn.Metadata.Name,
		"pod_events":       len(scn.Events.Pods),
		"scheduler_events": len(scn.Events.SchedulerConfigs),
	}).Info("Scenario loaded successfully")

	// Dry run mode.
	if opts.DryRun {
		log.WithContext(ctx).Info("Dry run mode - validating scenario only")
		return validateScenario(scn, log)
	}

	// Create output directory.
	outputPath := filepath.Join(opts.OutputDir, simulationID)
	if opts.SaveMetrics {
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		log.WithContext(ctx).WithFields(map[string]interface{}{
			"path": outputPath,
		}).Debug("Created output directory")
	}

	// Get clientset.
	clientset, err := util.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Reset cluster if requested.
	if opts.ClusterReset {
		log.WithContext(ctx).Warn("Resetting cluster state")

		if err := resetCluster(ctx, scn, clientset, log); err != nil {
			return fmt.Errorf("failed to reset cluster: %w", err)
		}
	}

	// Create scheduler.
	sched := scheduler.New()

	// Setup scheduler client for scheduler events.
	if len(scn.Events.SchedulerConfigs) > 0 && opts.SchedulerClient != "" {
		client := scenario.NewHTTPSchedulerClient(opts.SchedulerClient, 10*time.Second)
		for _, event := range scn.Events.SchedulerConfigs {
			event.SetClient(client)
		}

		log.WithContext(ctx).WithFields(map[string]interface{}{
			"url": opts.SchedulerClient,
		}).Debug("Configured scheduler client")
	}

	// Configure pod events.
	for _, event := range scn.Events.Pods {
		event.SetClientset(clientset)
		event.SetScheduler(sched)
	}

	// Create metrics collector.
	var collector metrics.Collector
	if opts.SaveMetrics {
		collector = metrics.NewMemoryCollector()
	}

	// Create simulation.
	simulation := simulator.NewSimulation(scn, clientset, sched)
	simulation.LoadEvents()

	// Setup context with timeout.
	simCtx := ctx

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		simCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Start simulation.
	log.WithContext(ctx).Info("Starting simulation")

	startTime := time.Now()

	if err := simulation.Start(simCtx); err != nil {
		return fmt.Errorf("simulation failed: %w", err)
	}

	duration := time.Since(startTime)
	log.WithContext(ctx).WithFields(map[string]interface{}{
		"duration": duration,
	}).Info("Simulation completed")

	// Save metrics.
	if opts.SaveMetrics && collector != nil {
		log.WithContext(ctx).WithFields(map[string]interface{}{
			"format": opts.MetricsFormat,
			"path":   outputPath,
		}).Info("Saving metrics")

		// Get stats from simulation and convert to metrics.
		stats := simulation.GetStats()
		if err := stats.ExportCSV(outputPath); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to export legacy metrics")
		}

		// Export new metrics format if collector was used.
		if err := collector.ExportMetrics(ctx, opts.MetricsFormat, outputPath); err != nil {
			return fmt.Errorf("failed to export metrics: %w", err)
		}
	}

	return nil
}

// runStatus executes the status command.
func runStatus(ctx context.Context, opts *Options, log *logger.Logger) error {
	// TODO: Implement simulation status tracking.
	log.Info("Checking simulation status...")
	fmt.Println("No running simulations found")

	return nil
}

// runStop executes the stop command.
func runStop(ctx context.Context, opts *Options, log *logger.Logger) error {
	// TODO: Implement simulation stop functionality.
	log.Info("Stopping simulations...")
	fmt.Println("No running simulations to stop")

	return nil
}

// runRandom executes the random command.
func runRandom(ctx context.Context, opts *Options, numPods int, duration time.Duration, log *logger.Logger) error {
	simulationID := fmt.Sprintf("random-%s", time.Now().Format("2006-01-02_15-04-05"))
	ctx = logger.WithSimulationID(ctx, simulationID)

	log.WithContext(ctx).WithFields(map[string]interface{}{
		"num_pods": numPods,
		"duration": duration,
	}).Info("Starting random simulation")

	// Get clientset.
	clientset, err := util.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create scenario.
	scn := scenario.NewScenario(scenario.WithName("random-simulation"))
	sched := scheduler.New()

	// Generate random pod events.
	for i := 0; i < numPods; i++ {
		pod := util.PodFactory.NewPod(
			util.PodWithMetadata(
				fmt.Sprintf("random-pod-%d", i),
				"default",
				nil,
				nil,
			),
			util.PodWithContainer(
				"server",
				"nginx",
				fmt.Sprintf("%dm", rand.IntnRange(100, 1000)),
				fmt.Sprintf("%dMi", rand.IntnRange(128, 1024)),
			),
		)

		arrivalTime := time.Duration(rand.IntnRange(0, int(duration.Seconds()/2))) * time.Second
		podDuration := time.Duration(rand.IntnRange(5, 30)) * time.Second

		event := scenario.NewPodEvent(pod.Name, arrivalTime, podDuration, pod, clientset, sched)
		scn.Events.Pods = append(scn.Events.Pods, event)
	}

	// Create simulation.
	simulation := simulator.NewSimulation(scn, clientset, sched)
	simulation.LoadEvents()

	// Run with timeout.
	simCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	if err := simulation.Start(simCtx); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("simulation failed: %w", err)
	}

	log.WithContext(ctx).Info("Random simulation completed")

	// Save metrics if requested.
	if opts.SaveMetrics {
		outputPath := filepath.Join(opts.OutputDir, simulationID)
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		stats := simulation.GetStats()
		if err := stats.ExportCSV(outputPath); err != nil {
			return fmt.Errorf("failed to export metrics: %w", err)
		}

		log.WithContext(ctx).WithFields(map[string]interface{}{
			"path": outputPath,
		}).Info("Metrics saved")
	}

	return nil
}

// resetCluster resets the cluster state.
func resetCluster(ctx context.Context, scn *scenario.Scenario, clientset *kubernetes.Clientset, log *logger.Logger) error {
	log.WithContext(ctx).Info("Resetting cluster")

	// Delete all pods.
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Skip system pods.
		if isSystemPod(&pod) {
			continue
		}

		if err := clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			log.WithContext(ctx).WithFields(map[string]interface{}{
				"pod":       pod.Name,
				"namespace": pod.Namespace,
			}).WithError(err).Warn("Failed to delete pod")
		}
	}

	// Delete all nodes (for KWOK environments).
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if err := clientset.CoreV1().Nodes().Delete(ctx, node.Name, metav1.DeleteOptions{}); err != nil {
			log.WithContext(ctx).WithFields(map[string]interface{}{
				"node": node.Name,
			}).WithError(err).Warn("Failed to delete node")
		}
	}

	// Create cluster from scenario if defined.
	if scn.Cluster != nil {
		log.WithContext(ctx).Info("Creating cluster from scenario")
		return scn.Cluster.Create(clientset)
	}

	log.WithContext(ctx).Debug("Cluster reset completed")

	return nil
}

// validateScenario validates a scenario without running it.
func validateScenario(scn *scenario.Scenario, log *logger.Logger) error {
	log.Info("Validating scenario")

	// Check for events.
	if len(scn.Events.Pods) == 0 && len(scn.Events.SchedulerConfigs) == 0 {
		return fmt.Errorf("scenario has no events")
	}

	// Validate pod events.
	for _, event := range scn.Events.Pods {
		if event.Pod == nil {
			return fmt.Errorf("pod event %s has nil pod", event.Name)
		}

		if len(event.Pod.Spec.Containers) == 0 {
			return fmt.Errorf("pod %s has no containers", event.Pod.Name)
		}
	}

	log.WithFields(map[string]interface{}{
		"pod_events":       len(scn.Events.Pods),
		"scheduler_events": len(scn.Events.SchedulerConfigs),
	}).Info("Scenario is valid")

	return nil
}

// isSystemPod checks if a pod is a system pod.
func isSystemPod(pod *v1.Pod) bool {
	systemNamespaces := []string{"kube-system", "kube-public", "kube-node-lease"}
	for _, ns := range systemNamespaces {
		if pod.Namespace == ns {
			return true
		}
	}

	return false
}

// Cmd is deprecated, use NewCommand instead.
var Cmd = NewCommand(logger.Default())
