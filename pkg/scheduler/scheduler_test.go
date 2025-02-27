package scheduler

import (
	"github.com/maczg/kube-event-generator/pkg/factory"
	"testing"
	"time"
)

func TestScheduler_Run(t *testing.T) {
	km, err := NewKubeManager()
	if err != nil {
		t.Fatalf("error creating kube manager: %v", err)
	}

	s := NewScheduler(
		WithDeadline(30),
		WithKubeManager(km),
	)

	p1 := factory.NewPod(
		factory.WithMetadata("pod-1", "default"),
		factory.WithContainer("nginx", "nginx:latest", "100m", "100Mi"),
	)

	p2 := factory.NewPod(
		factory.WithMetadata("pod-2", "default"),
		factory.WithContainer("nginx", "nginx:latest", "100m", "100Mi"),
	)

	e1 := NewCreatePodEvent(p1, 5*time.Second, 10*time.Second)
	e2 := NewCreatePodEvent(p2, 10*time.Second, 10*time.Second)

	s.AddEvent(e1)
	s.AddEvent(e2)

	s.Run()
}
