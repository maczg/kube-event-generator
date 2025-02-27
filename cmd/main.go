package main

import (
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/sirupsen/logrus"
)

func main() {
	scenarioFile := "scenario.yaml"
	sc, err := scenario.LoadFromPath(scenarioFile)
	if err != nil {
		logrus.Fatalf("Failed to load scenario: %v", err)
	}

	mc := metric.NewCollector(
		metric.WithResultDir("results"),
	)
	mc.WithMetric(scenario.PendingPodQueueMetric)
	mc.WithMetric(scenario.PendingPodDurationMetric)

	sdl := scenario.NewScheduler(
		scenario.WithKubeClient(),
		scenario.WithMetricCollector(mc),
		scenario.WithDeadline(100),
	)

	var events []*scenario.Event
	for _, e := range sc.Events {
		e.GetPodFromSpec()
		events = append(events, &e)
	}
	sdl.Enqueue(events)
	sdl.Run()
}
