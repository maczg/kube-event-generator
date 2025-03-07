package generate

import (
	"fmt"
	"github.com/maczg/kube-event-generator/cmd/simulation/generate/dist"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"time"
)

const MilliCoresPerCore = 1000

// randomPodCpu converts a random float from a distribution into a K8s resource.Quantity string.
func randomPodCpu(d dist.Distribution, maxNodeCpu resource.Quantity) resource.Quantity {
	log := logger.NewLogger(logger.LevelInfo, "generate")
	val := d.Next()
	millicores := int(val * MilliCoresPerCore)
	res := resource.MustParse(fmt.Sprintf("%dm", millicores))
	if res.Cmp(maxNodeCpu) > 0 {
		log.Warn("Pod CPU request exceeds node capacity. Setting to node capacity.")
		return maxNodeCpu
	}
	return res
}

// randomPodMemory converts a random float from a distribution into a K8s resource.Quantity string.
func randomPodMemory(d dist.Distribution, scaleFactory float64, maxNodeMemory resource.Quantity) resource.Quantity {
	log := logger.NewLogger(logger.LevelInfo, "generate")
	val := d.Next()
	mebi := val * scaleFactory
	res := resource.MustParse(fmt.Sprintf("%dMi", int(mebi)))
	if res.Cmp(maxNodeMemory) > 0 {
		log.Warn("Pod memory request exceeds node capacity. Setting to node capacity.")
		return maxNodeMemory
	}
	return res
}

// generateEvents generates a list of events based on the given parameters.
func generateEvents(params Params) []scenario.Event {
	// arrival lambda is the rate of events per second = 1
	// duration shape is the shape parameter of the Weibull distribution = 1.5
	// duration k is the scale parameter of the Weibull distribution = 2.0

	r := params.Rng
	maxNodeCpu := GetMaxNodeCpu(params.Scenario.Cluster.Nodes)
	maxNodeMemory := GetMaxNodeMem(params.Scenario.Cluster.Nodes)

	// Create Exponential distribution for inter-arrival times (rate λ=1.0 => mean=1).
	// Adjust λ for your arrival rate.
	arrivalDist := dist.NewExponential(r, params.ArrivalLambda)
	// Create Weibull distribution for durations with shape=1.5, scale=2.0 (example params).
	durationDist := dist.NewWeibull(r, params.DurationShape, params.DurationK)
	cpuDist := dist.NewWeibull(r, params.PodCpuScaleK, params.PodCpuScaleLambda)
	memDist := dist.NewWeibull(r, params.PodMemShapeK, params.PodMemScaleLambda)

	var events []scenario.Event
	var currentTime float64
	for i := 0; i < params.NumEvents; i++ {
		// Increment by an exponential inter-arrival time
		interArrival := arrivalDist.Next()
		currentTime += interArrival

		// Duration from Weibull
		serviceTime := durationDist.Next()

		cpuQty := randomPodCpu(cpuDist, maxNodeCpu)                            // e.g. "100m"
		memQty := randomPodMemory(memDist, params.PodMemFactor, maxNodeMemory) // e.g. "128Mi"

		pod := utils.PodFactory.NewPod(
			utils.WithName(fmt.Sprintf("pod-%d", i)),
			utils.WithNamespace("default"),
			utils.PodWithContainer("server", "nginx", cpuQty.String(), memQty.String()),
		)

		at := time.Duration(currentTime*params.ArrivalScaleFactor) * (time.Second)
		duration := time.Duration(serviceTime*params.DurationScaleFactor) * (time.Second)
		e := scenario.NewEvent(pod.Name, at, duration, *pod)
		events = append(events, *e)
	}
	return events
}

// GetMaxNodeCpu returns the maximum CPU capacity of all nodes in the cluster.
func GetMaxNodeCpu(nodes []v1.Node) resource.Quantity {
	var maxCpu resource.Quantity
	for _, node := range nodes {
		cpu := node.Status.Capacity[v1.ResourceCPU]
		if maxCpu.Cmp(cpu) < 0 {
			maxCpu = cpu
		}
	}
	return maxCpu
}

// GetMaxNodeMem returns the maximum memory capacity of all nodes in the cluster.
func GetMaxNodeMem(nodes []v1.Node) resource.Quantity {
	var maxMem resource.Quantity
	for _, node := range nodes {
		mem := node.Status.Capacity[v1.ResourceMemory]
		if maxMem.Cmp(mem) < 0 {
			maxMem = mem
		}
	}
	return maxMem
}
