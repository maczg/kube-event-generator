package pkg

import (
	"testing"
	"time"
)

func TestManager_Run(t *testing.T) {
	// Create a new Manager
	m, _ := NewManager(
		WithEndAfter(60),
		WithKubeClient(),
	)
	//time.AfterFunc(2*time.Second, func() {
	//	at := time.Now().Add(3 * time.Second)
	//	event := NewEvent(&at, nil)
	//	m.EnqueueEvent(event)
	//})

	time.AfterFunc(5*time.Second, func() {
		at := time.Now().Add(5 * time.Second)
		duration := 10 * time.Second
		pod := NewPod(WitMetadata("test-pod", "default"),
			WithContainer("test-container", "nginx"),
			WithResource("100m", "100Mi"))
		event := NewScheduleEvent(pod, at, duration)
		m.EnqueueEvent(event)
	})

	// Start the Manager
	m.Run()

}
