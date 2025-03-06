package generate

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/spf13/cobra"
	"math"
	"math/rand"
	"time"
)

var (
	outputDefault = func() string {
		return fmt.Sprintf("scenario-%s.yaml", time.Now().Format("15-04-05"))
	}()
	eventCount int
	GeometricP float64
	Lambda     float64
	CpuMin     int
	CpuMax     int
	MemMin     int
	MemMax     int
	RandSeed   int64
)

var Cmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate a simulation scenario",
	Long:    `Generate a simulation`,
	Run: func(cmd *cobra.Command, args []string) {
		simName, _ := cmd.Flags().GetString("name")
		nodes := utils.NodeFactory.NewNodeBatch("node", "4", "16", "110", 5)
		scn := &scenario.Scenario{
			Name: simName,
		}
		for _, node := range nodes {
			scn.Cluster.Nodes = append(scn.Cluster.Nodes, *node)
		}

		var currentTime time.Duration
		rng := rand.New(rand.NewSource(RandSeed))

		// Generate all Pod Events
		for i := 0; i < eventCount; i++ {
			interArrivalSeconds := exponentialRand(Lambda, rng)
			currentTime += time.Duration(float64(time.Second) * interArrivalSeconds)
			durationSeconds := geometricRand(GeometricP, rng)
			duration := time.Duration(durationSeconds) * time.Second

			cpuReq := fmt.Sprintf("%dm", randomBetween(int64(CpuMin), int64(CpuMax), rng))
			memReq := fmt.Sprintf("%dMi", randomBetween(int64(MemMin), int64(MemMax), rng))
			pod := utils.PodFactory.NewPod(
				utils.WithName(fmt.Sprintf("pod-%d", i)),
				utils.WithName(fmt.Sprintf("pod-%d", i)),
				utils.PodWithContainer("server", "server", cpuReq, memReq),
			)
			evt := scenario.NewEvent(pod.Name, currentTime, duration, *pod)
			scn.Events = append(scn.Events, *evt)
		}
		err := scn.Dump("scenario.yaml")
		if err != nil {
			fmt.Printf("failed to dump scenario: %v", err)
		}
	},
}

func init() {
	Cmd.Flags().IntVarP(&eventCount, "count", "c", 10, "Number of events to generate")
	Cmd.Flags().StringP("output", "o", outputDefault, "Output file")
	Cmd.Flags().Float64VarP(&GeometricP, "geometric-p", "g", 0.5, "Geometric distribution parameter")
	Cmd.Flags().Float64VarP(&Lambda, "lambda", "l", 0.5, "Poisson distribution parameter")
	Cmd.Flags().IntVarP(&CpuMin, "cpu-min", "m", 100, "Minimum CPU value")
	Cmd.Flags().IntVarP(&CpuMax, "cpu-max", "M", 1000, "Maximum CPU value")
	Cmd.Flags().IntVarP(&MemMin, "mem-min", "e", 256, "Minimum memory value")
	Cmd.Flags().IntVarP(&MemMax, "mem-max", "E", 1024, "Maximum memory value")
	Cmd.Flags().Int64VarP(&RandSeed, "seed", "s", 42, "Random seed")
}

// exponentialRand samples an exponential random variable with rate λ = lambda.
func exponentialRand(lambda float64, rng *rand.Rand) float64 {
	// If X ~ Exp(lambda), then X = -log(U) / lambda, where U ~ Uniform(0,1).
	u := rng.Float64()
	return -math.Log(u) / lambda
}

// geometricRand returns an integer sample from a geometric distribution with parameter p.
// This is "the number of trials up to *and including* the first success."
// If you prefer "the number of failures before the first success," adjust accordingly.
func geometricRand(p float64, rng *rand.Rand) int64 {
	if p <= 0.0 || p >= 1.0 {
		// fallback: default to 1 second if p is out of range
		return 1
	}
	u := rng.Float64()
	return int64(math.Ceil(math.Log(u) / math.Log(1.0-p)))
}

// randomBetween returns a uniformly distributed integer in [min, max].
func randomBetween(min, max int64, rng *rand.Rand) int64 {
	if max < min {
		// guard
		return min
	}
	if max == min {
		return min
	}
	return rng.Int63n(max-min+1) + min
}
