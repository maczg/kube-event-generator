package scenario

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/distribution"
	"github.com/maczg/kube-event-generator/pkg/kubeutils"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/simulation"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/rand"
	"os"
	"strings"
	"time"
)

// newGenerateScenarioCommand creates the scenario generation sub-command.
func newGenerateScenarioCommand(log *logger.Logger) *cobra.Command {
	var outputDir string
	var scenarioName string
	var configFile string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a new scenario",
		Long:  "Create a new scenario based on specified parameters and save it to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(configFile)
			fullPath := outputDir + "/" + strings.ToLower(scenarioName) + ".yaml"

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create output directory: %v", err)
				}
			}
			scn := simulation.NewScenario(simulation.ScenarioWithName(scenarioName))
			events := generatePodEvents(log, *cfg)
			for _, event := range events {
				scn.Events.Pods = append(scn.Events.Pods, *event)
			}
			return scn.ToYaml(fullPath)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file for scenario generation")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "scenarios", "Directory to save the generated scenario")
	cmd.Flags().StringVarP(&scenarioName, "name", "n", defaultScenarioName(), "Name of the generated scenario")

	return cmd
}

func defaultScenarioName() string {
	// Generate a default scenario name based on the current timestamp
	return fmt.Sprintf("scenario-%s", time.Now().Format("20060102150405"))
}

func generatePodEvents(logger *logger.Logger, cfg Config) (events []*simulation.PodEvent) {
	events = make([]*simulation.PodEvent, 0)
	// Generate pod events based on the configuration
	rng := rand.New(rand.NewSource(cfg.Seed))
	arrivalDist := distribution.NewExponential(rng, cfg.Generation.ArrivalScale)
	// Create Weibull distribution for durations with shape=1.5, scale=2.0 (example params).
	durationDist := distribution.NewWeibull(rng, cfg.Generation.DurationScale, cfg.Generation.DurationShape)
	cpuDist := distribution.NewWeibull(rng, cfg.Generation.PodCpuShape, cfg.Generation.PodCpuScale)
	memDist := distribution.NewWeibull(rng, cfg.Generation.PodMemScale, cfg.Generation.PodMemShape)
	var currentTime float64
	for i := 0; i < cfg.Generation.NumPodEvents; i++ {
		// Increment by an exponential inter-arrival time
		interArrival := arrivalDist.Next()
		currentTime += interArrival

		// Duration from Weibull
		serviceTime := durationDist.Next()

		cpuQty := resource.MustParse(fmt.Sprintf("%dm", int(cpuDist.Next()*1000)))                         // e.g. "100m"
		memQty := resource.MustParse(fmt.Sprintf("%dMi", int(memDist.Next()*cfg.Generation.PodMemFactor))) // e.g. "128Mi"

		pod := kubeutils.ObjectFactory.NewPod(
			kubeutils.PodWithMetadata(fmt.Sprintf("pod-%d", i), "default", nil, nil),
			kubeutils.PodWithContainer("server", "nginx", cpuQty.String(), memQty.String()),
		)
		at := time.Duration(currentTime*cfg.Generation.ArrivalScaleFactor) * (time.Second)
		duration := time.Duration(serviceTime*cfg.Generation.DurationScaleFactor) * (time.Second)
		if duration <= 0 {
			duration = time.Duration(rand.Intn(50)+1) * time.Second
			logger.Warnf("duration is less than or equal to 0. Setting to %s ", duration.String())
		}
		event := simulation.NewCreatePodEvent(at, duration, pod)
		events = append(events, event)
	}
	return events
}
