package generate

import (
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"time"
)

var (
	outputFile       string
	outputDir        string
	numEvents        int
	interArrivalDist string
	durationDist     string = "exponential"
	cpuDist          string
	memDist          string
	seed             int64
	rng              = rand.New(rand.NewSource(seed))
	timeNow          = time.Now()
)

var Cmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate a simulation scenario",
	Long:    `Generate a simulation`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// Initialize random seed
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
		//rand.Seed(seed)

		scn := &scenario.Scenario{
			Name:   fmt.Sprintf("generated-%s", timeNow.Format("2006-01-02-15-04-05")),
			Events: []scenario.Event{},
		}

		nodes := utils.NodeFactory.NewNodeBatch("node", "4", "16", "110", 5)
		for _, node := range nodes {
			scn.Cluster.Nodes = append(scn.Cluster.Nodes, *node)
		}

		var currentTime time.Duration

		for i := 0; i < numEvents; i++ {
			nextArrival := generateArrival(interArrivalDist)
			currentTime += nextArrival
			duration := generateDuration(durationDist)
			cpuQty := generateResourceQuantity(cpuDist) // e.g. "100m"
			memQty := generateResourceQuantity(memDist) // e.g. "128Mi"

			pod := utils.PodFactory.NewPod(
				utils.WithName(fmt.Sprintf("pod-%d", i)),
				utils.WithNamespace("default"),
				utils.PodWithContainer("server", "server", cpuQty.String(), memQty.String()),
			)
			evt := scenario.NewEvent(pod.Name, currentTime, duration, *pod)
			scn.Events = append(scn.Events, *evt)
		}

		err := os.Mkdir(outputDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		outputFile = fmt.Sprintf("%s/%s", outputDir, outputFile)
		err = scn.Dump(outputFile)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.Flags().StringVarP(&outputDir, "output-dir", "d", ".", "Output directory")
	Cmd.Flags().StringVarP(&outputFile, "output", "o", fmt.Sprintf("scenario-%s.yaml", timeNow.Format("2006-01-02-15-04-05")), "Output file name")
	Cmd.Flags().IntVarP(&numEvents, "events", "e", 20, "Number of events (pods) to generate")
	Cmd.Flags().StringVarP(&interArrivalDist, "arrival-dist", "z", "exponential", "Distribution for arrival times (exponential, uniform, etc.)")
	//Cmd.Flags().StringVarP(&durationDist, "duration-dist", "j", "exponential", "Distribution for pod lifetimes (exponential, uniform, etc.)")
	Cmd.Flags().StringVarP(&cpuDist, "cpu-dist", "k", "uniform", "Distribution for CPU requests (uniform, normal, etc.)")
	Cmd.Flags().StringVarP(&memDist, "mem-dist", "m", "normal", "Distribution for Memory requests (uniform, normal, etc.)")
	Cmd.Flags().Int64VarP(&seed, "seed", "s", 0, "Random seed (0 means use current time)")
}
