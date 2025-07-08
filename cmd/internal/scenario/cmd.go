package scenario

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/maczg/kube-event-generator/pkg/builder"
	"github.com/maczg/kube-event-generator/pkg/distribution"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/util"
)

// Options holds command options.
type Options struct {
	ConfigFile string
	OutputDir  string
	Seed       int64
	DryRun     bool
}

// NewCommand creates the scenario command.
func NewCommand(log *logger.Logger) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:     "scenario",
		Aliases: []string{"scn"},
		Short:   "Scenario management commands",
		Long:    "Commands for generating and managing simulation scenarios",
	}

	// Add sub-commands.
	cmd.AddCommand(
		newGenerateCommand(opts, log),
		newValidateCommand(opts, log),
		newListCommand(opts, log),
	)

	return cmd
}

// newGenerateCommand creates the generate sub-command.
func newGenerateCommand(opts *Options, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "Generate a new scenario",
		Long: `Generate a new scenario based on configuration parameters.
The scenario will include pod events with arrival times following 
Poisson distribution and durations following Weibull distribution.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts, log)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigFile, "config", "c", "config.yaml", "Configuration file path")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", "", "Output directory (overrides config)")
	cmd.Flags().Int64Var(&opts.Seed, "seed", 0, "Random seed (0 for current time)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Print scenario without saving")

	return cmd
}

// newValidateCommand creates the validate sub-command.
func newValidateCommand(opts *Options, log *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "validate [scenario-file]",
		Short: "Validate a scenario file",
		Long:  "Validate that a scenario file is properly formatted and contains valid events",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args[0], log)
		},
	}
}

// newListCommand creates the list sub-command.
func newListCommand(opts *Options, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available scenarios",
		Long:  "List all scenarios in the scenarios directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, log)
		},
	}

	cmd.Flags().StringVarP(&opts.OutputDir, "dir", "d", "scenarios", "Directory to list scenarios from")

	return cmd
}

// runGenerate executes the generate command.
func runGenerate(opts *Options, log *logger.Logger) error {
	// Load configuration.
	cfg := GetConfig(opts.ConfigFile)

	// Override output directory if specified.
	if opts.OutputDir != "" {
		cfg.OutputDir = opts.OutputDir
	}

	// Set seed.
	if opts.Seed != 0 {
		cfg.Seed = opts.Seed
	} else if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
	}

	log.WithFields(map[string]interface{}{
		"config_file": opts.ConfigFile,
		"output_dir":  cfg.OutputDir,
		"seed":        cfg.Seed,
	}).Info("Generating new scenario")

	// Create output directory.
	if !opts.DryRun {
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Create scenario.
	scn := scenario.NewScenario(scenario.WithName(cfg.ScenarioName))

	// Generate events.
	events, err := generatePodEvents(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to generate pod events: %w", err)
	}

	scn.Events.Pods = events

	// Add scheduler events if configured.
	if cfg.Generation.SchedulerEvents != nil {
		schedulerEvents := generateSchedulerEvents(cfg, log)
		scn.Events.SchedulerConfigs = schedulerEvents
	}

	// Generate cluster nodes if configured.
	if cfg.Generation.Cluster.Generate {
		cluster, err := generateCluster(cfg, log)
		if err != nil {
			return fmt.Errorf("failed to generate cluster: %w", err)
		}
		scn.Cluster = cluster
	}

	// Display scenario summary.
	displayScenarioSummary(scn, log)

	// Save or print scenario.
	if opts.DryRun {
		log.Info("Dry run mode - scenario not saved")
		scn.Describe()

		return nil
	}

	outputPath := cfg.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(cfg.OutputDir, fmt.Sprintf("%s.yaml", cfg.ScenarioName))
	}

	if err := scn.ToYaml(outputPath); err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"path": outputPath,
	}).Info("Scenario saved successfully")

	return nil
}

// generatePodEvents generates pod events based on configuration.
func generatePodEvents(cfg Config, log *logger.Logger) ([]*scenario.PodEvent, error) {
	events := make([]*scenario.PodEvent, 0, cfg.Generation.NumPodEvents)

	// Initialize random number generator.
	rng := rand.New(rand.NewSource(cfg.Seed))

	// Create distributions.
	arrivalDist := distribution.NewExponential(rng, cfg.Generation.ArrivalScale)
	durationDist := distribution.NewWeibull(rng, cfg.Generation.DurationScale, cfg.Generation.DurationShape)
	cpuDist := distribution.NewWeibull(rng, cfg.Generation.PodCpuShape, cfg.Generation.PodCpuScale)
	memDist := distribution.NewWeibull(rng, cfg.Generation.PodMemScale, cfg.Generation.PodMemShape)

	var currentTime float64

	for i := 0; i < cfg.Generation.NumPodEvents; i++ {
		// Calculate arrival time.
		interArrival := arrivalDist.Next()
		currentTime += interArrival

		// Calculate duration.
		serviceTime := durationDist.Next()

		// Generate resource requirements.
		cpuMillis := int(cpuDist.Next() * 1000 * cfg.Generation.PodCpuFactor)
		memMi := int(memDist.Next() * cfg.Generation.PodMemFactor)

		// Ensure minimum resources.
		if cpuMillis < 10 {
			cpuMillis = 10
		}

		if memMi < 16 {
			memMi = 16
		}

		cpuQty := resource.MustParse(fmt.Sprintf("%dm", cpuMillis))
		memQty := resource.MustParse(fmt.Sprintf("%dMi", memMi))

		// Create pod.
		pod := util.PodFactory.NewPod(
			util.PodWithMetadata(fmt.Sprintf("pod-%d", i), "default", nil, nil),
			util.PodWithContainer("server", "nginx", cpuQty.String(), memQty.String()),
		)

		// Calculate timing.
		at := time.Duration(currentTime*cfg.Generation.ArrivalScaleFactor) * time.Second
		duration := time.Duration(serviceTime*cfg.Generation.DurationScaleFactor) * time.Second

		// Ensure minimum duration.
		if duration <= 0 {
			duration = time.Duration(rand.Intn(50)+1) * time.Second
			log.WithFields(map[string]interface{}{
				"pod":      pod.Name,
				"original": serviceTime,
				"adjusted": duration,
			}).Warn("Adjusted negative duration")
		}

		event := scenario.NewPodEvent(pod.Name, at, duration, pod, nil, nil)
		events = append(events, event)
	}

	log.WithFields(map[string]interface{}{
		"count": len(events),
	}).Debug("Generated pod events")

	return events, nil
}

// generateSchedulerEvents generates scheduler configuration events.
func generateSchedulerEvents(cfg Config, log *logger.Logger) []*scenario.SchedulerEvent {
	events := make([]*scenario.SchedulerEvent, 0)

	if cfg.Generation.SchedulerEvents == nil {
		return events
	}

	for _, se := range cfg.Generation.SchedulerEvents {
		event := &scenario.SchedulerEvent{
			Name:         se.Name,
			ExecuteAfter: scenario.EventDuration(se.After),
			Weights:      se.Weights,
		}
		events = append(events, event)
	}

	log.WithFields(map[string]interface{}{
		"count": len(events),
	}).Debug("Generated scheduler events")

	return events
}

// generateCluster generates cluster nodes based on configuration.
func generateCluster(cfg Config, log *logger.Logger) (*scenario.Cluster, error) {
	cluster := scenario.NewCluster()
	
	// Calculate allocatable resources (95% of capacity for CPU, 87.5% for memory)
	allocatableCpu := int64(float64(cfg.Generation.Cluster.CpuPerNode) * 0.95)
	allocatableMemory := int64(float64(cfg.Generation.Cluster.MemoryPerNode) * 0.875)
	
	for i := 0; i < cfg.Generation.Cluster.NumNodes; i++ {
		nodeName := fmt.Sprintf("node-%d", i+1)
		
		// Select zone cyclically
		zone := cfg.Generation.Cluster.Zones[i%len(cfg.Generation.Cluster.Zones)]
		
		// Create node labels
		labels := map[string]string{
			"kubernetes.io/hostname":              nodeName,
			"node.kubernetes.io/instance-type":    "standard",
			"topology.kubernetes.io/zone":         zone,
			"type":                               "kwok",
		}
		
		// Build node using NodeBuilder
		nodeBuilder := builder.NewNodeBuilder(nodeName).
			WithCapacity(cfg.Generation.Cluster.CpuPerNode, cfg.Generation.Cluster.MemoryPerNode, cfg.Generation.Cluster.PodsPerNode).
			WithAllocatable(allocatableCpu, allocatableMemory, cfg.Generation.Cluster.PodsPerNode).
			WithLabels(labels)
		
		node, err := nodeBuilder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build node %s: %w", nodeName, err)
		}
		
		cluster.Nodes = append(cluster.Nodes, node)
		
		log.WithFields(map[string]interface{}{
			"node":   nodeName,
			"zone":   zone,
			"cpu":    cfg.Generation.Cluster.CpuPerNode,
			"memory": cfg.Generation.Cluster.MemoryPerNode,
			"pods":   cfg.Generation.Cluster.PodsPerNode,
		}).Debug("Generated cluster node")
	}
	
	log.WithFields(map[string]interface{}{
		"nodes": len(cluster.Nodes),
		"total_cpu": cfg.Generation.Cluster.CpuPerNode * int64(cfg.Generation.Cluster.NumNodes),
		"total_memory": cfg.Generation.Cluster.MemoryPerNode * int64(cfg.Generation.Cluster.NumNodes),
	}).Info("Generated cluster")
	
	return cluster, nil
}

// displayScenarioSummary displays a summary of the generated scenario.
func displayScenarioSummary(scn *scenario.Scenario, log *logger.Logger) {
	fmt.Printf("\nScenario Summary:\n")
	fmt.Printf("================\n")
	fmt.Printf("Name: %s\n", scn.Metadata.Name)
	fmt.Printf("Created: %s\n", scn.Metadata.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Pod Events: %d\n", len(scn.Events.Pods))
	fmt.Printf("Scheduler Events: %d\n", len(scn.Events.SchedulerConfigs))
	
	// Display cluster summary
	if scn.Cluster != nil && len(scn.Cluster.Nodes) > 0 {
		fmt.Printf("Cluster Nodes: %d\n", len(scn.Cluster.Nodes))
		
		// Calculate total cluster resources
		var totalCPU, totalMemory, totalPods int64
		zones := make(map[string]int)
		
		for _, node := range scn.Cluster.Nodes {
			if cpu, ok := node.Status.Capacity[v1.ResourceCPU]; ok {
				totalCPU += cpu.Value()
			}
			if mem, ok := node.Status.Capacity[v1.ResourceMemory]; ok {
				totalMemory += mem.Value()
			}
			if pods, ok := node.Status.Capacity[v1.ResourcePods]; ok {
				totalPods += pods.Value()
			}
			
			if zone, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
				zones[zone]++
			}
		}
		
		fmt.Printf("Total Cluster Capacity:\n")
		fmt.Printf("  CPU: %.0f cores\n", float64(totalCPU))
		fmt.Printf("  Memory: %.2f GB\n", float64(totalMemory)/(1024*1024*1024))
		fmt.Printf("  Pods: %d\n", totalPods)
		
		if len(zones) > 0 {
			fmt.Printf("Availability Zones:\n")
			for zone, count := range zones {
				fmt.Printf("  %s: %d nodes\n", zone, count)
			}
		}
	}

	if len(scn.Events.Pods) > 0 {
		// Calculate statistics.
		var totalCPU, totalMem int64

		var minDuration, maxDuration time.Duration

		var totalDuration time.Duration

		minDuration = time.Hour * 24 * 365

		for _, event := range scn.Events.Pods {
			// Sum resources.
			for _, container := range event.Pod.Spec.Containers {
				if cpu, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
					totalCPU += cpu.MilliValue()
				}

				if mem, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
					totalMem += mem.Value()
				}
			}

			// Track durations.
			duration := event.ExecuteForDuration()
			totalDuration += duration

			if duration < minDuration {
				minDuration = duration
			}

			if duration > maxDuration {
				maxDuration = duration
			}
		}

		avgDuration := totalDuration / time.Duration(len(scn.Events.Pods))

		fmt.Printf("\nResource Summary:\n")
		fmt.Printf("  Total CPU Requested: %.2f cores\n", float64(totalCPU)/1000)
		fmt.Printf("  Total Memory Requested: %.2f GB\n", float64(totalMem)/(1024*1024*1024))
		fmt.Printf("  Average CPU per Pod: %.0f millicores\n", float64(totalCPU)/float64(len(scn.Events.Pods)))
		fmt.Printf("  Average Memory per Pod: %.0f MB\n", float64(totalMem)/(1024*1024)/float64(len(scn.Events.Pods)))

		fmt.Printf("\nTiming Summary:\n")
		fmt.Printf("  Simulation Duration: %s\n", time.Duration(scn.Events.Pods[len(scn.Events.Pods)-1].ExecuteAfter))
		fmt.Printf("  Pod Duration Range: %s - %s\n", minDuration, maxDuration)
		fmt.Printf("  Average Pod Duration: %s\n", avgDuration)
	}

	fmt.Println()
}

// runValidate executes the validate command.
func runValidate(scenarioFile string, log *logger.Logger) error {
	log.WithFields(map[string]interface{}{
		"file": scenarioFile,
	}).Info("Validating scenario")

	// Load scenario.
	scn, err := scenario.LoadFromYaml(scenarioFile)
	if err != nil {
		return fmt.Errorf("failed to load scenario: %w", err)
	}

	// Validate metadata.
	if scn.Metadata.Name == "" {
		return fmt.Errorf("scenario name is required")
	}

	// Validate events.
	if len(scn.Events.Pods) == 0 && len(scn.Events.SchedulerConfigs) == 0 {
		return fmt.Errorf("scenario must contain at least one event")
	}

	// Validate pod events.
	for i, event := range scn.Events.Pods {
		if event.Name == "" {
			return fmt.Errorf("pod event %d: name is required", i)
		}

		if event.Pod == nil {
			return fmt.Errorf("pod event %s: pod definition is required", event.Name)
		}

		if event.Pod.Name == "" {
			return fmt.Errorf("pod event %s: pod name is required", event.Name)
		}

		if len(event.Pod.Spec.Containers) == 0 {
			return fmt.Errorf("pod event %s: at least one container is required", event.Name)
		}
	}

	// Validate scheduler events.
	for i, event := range scn.Events.SchedulerConfigs {
		if event.Name == "" {
			return fmt.Errorf("scheduler event %d: name is required", i)
		}

		if len(event.Weights) == 0 {
			return fmt.Errorf("scheduler event %s: weights are required", event.Name)
		}
	}

	log.WithFields(map[string]interface{}{
		"pod_events":       len(scn.Events.Pods),
		"scheduler_events": len(scn.Events.SchedulerConfigs),
	}).Info("Scenario is valid")

	// Display summary.
	displayScenarioSummary(scn, log)

	return nil
}

// runList executes the list command.
func runList(opts *Options, log *logger.Logger) error {
	log.WithFields(map[string]interface{}{
		"directory": opts.OutputDir,
	}).Debug("Listing scenarios")

	// Read directory.
	entries, err := os.ReadDir(opts.OutputDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No scenarios found in %s\n", opts.OutputDir)
			return nil
		}

		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter YAML files.
	scenarios := make([]os.DirEntry, 0)

	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
			scenarios = append(scenarios, entry)
		}
	}

	if len(scenarios) == 0 {
		fmt.Printf("No scenarios found in %s\n", opts.OutputDir)
		return nil
	}

	fmt.Printf("\nAvailable Scenarios:\n")
	fmt.Printf("===================\n")

	for _, entry := range scenarios {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Try to load scenario to get details.
		scenarioPath := filepath.Join(opts.OutputDir, entry.Name())

		scn, err := scenario.LoadFromYaml(scenarioPath)
		if err != nil {
			fmt.Printf("  %s (invalid: %v)\n", entry.Name(), err)
			continue
		}

		fmt.Printf("  %s\n", entry.Name())
		fmt.Printf("    Name: %s\n", scn.Metadata.Name)
		fmt.Printf("    Created: %s\n", scn.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Size: %d bytes\n", info.Size())
		fmt.Printf("    Pod Events: %d\n", len(scn.Events.Pods))
		fmt.Printf("    Scheduler Events: %d\n", len(scn.Events.SchedulerConfigs))
		fmt.Println()
	}

	return nil
}

// Cmd is deprecated, use NewCommand instead.
var Cmd = NewCommand(logger.Default())
