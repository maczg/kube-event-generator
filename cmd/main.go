package main

import (
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	scenarioFile := "scenario.yaml"

	scenarioData, err := os.ReadFile(scenarioFile)
	if err != nil {
		logrus.Fatalf("Failed to read scenario file: %v", err)
	}
	sc, err := scenario.Load(scenarioData)
	if err != nil {
		logrus.Fatalf("Failed to load scenario: %v", err)
	}
	logrus.Infof("Loaded scenario with %d events", len(sc.Events))

	mc := metric.NewCollector(
		metric.WithResultDir("results"),
		metric.WithMetric("pending_queue_length"),
		metric.WithMetric("pending_duration"),
	)

	sdl := scenario.NewScheduler(
		scenario.WithKubeClient(),
		scenario.WithMetricCollector(mc),
		scenario.WithDeadline(40),
	)

	var events []*scenario.Event
	for _, e := range sc.Events {
		e.GetPodFromSpec()
		events = append(events, &e)
	}
	sdl.Enqueue(events)
	sdl.Run()
}
