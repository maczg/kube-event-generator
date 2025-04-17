package scenario

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/distribution"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/rand"
	"os"
	"time"
)

var configFile string

func init() {
	Cmd.AddCommand(GenCmd)
	GenCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file with generation parameneters")
}

var Cmd = &cobra.Command{
	Use:     "scenario",
	Aliases: []string{"scn"},
	Short:   "Scenario commands",
	Long:    `Scenario commands`,
}

var GenCmd = &cobra.Command{
	Use:     "generate",
	Short:   "Generate a scenario",
	Long:    `Generate a scenario`,
	Aliases: []string{"gen"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig(configFile)
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("failed to create output directory: %v", err)
			}
		}

		scn := scenario.NewScenario(scenario.WithName(cfg.ScenarioName))
		logrus.Infoln("generating new scenario")
		events := generatePodEvents(*cfg)
		for _, event := range events {
			scn.Events.Pods = append(scn.Events.Pods, event)
		}
		scn.Describe()
		return scn.ToYaml(cfg.OutputPath)
	},
}

func generatePodEvents(cfg Config) (events []*scenario.PodEvent) {
	events = make([]*scenario.PodEvent, 0)
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
		pod := util.PodFactory.NewPod(
			util.PodWithMetadata(fmt.Sprintf("pod-%d", i), "default", nil, nil),
			util.PodWithContainer("server", "nginx", cpuQty.String(), memQty.String()),
		)

		at := time.Duration(currentTime*cfg.Generation.ArrivalScaleFactor) * (time.Second)
		duration := time.Duration(serviceTime*cfg.Generation.DurationScaleFactor) * (time.Second)
		if duration <= 0 {
			duration = time.Duration(rand.Intn(50)+1) * time.Second
			logrus.Warnf("Duration is less than or equal to 0. Setting to %s ", duration.String())
			duration = time.Duration(rand.Intn(50)+1) * time.Second
		}

		event := scenario.NewPodEvent(pod.Name, at, duration, pod, nil, nil)
		events = append(events, event)
	}
	return events
}
