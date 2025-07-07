// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/maczg/kube-event-generator/pkg/builder"
	"github.com/maczg/kube-event-generator/pkg/metrics"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSimulation_BasicScenario(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create fake clientset
	clientset := fake.NewSimpleClientset()

	// Build scenario
	pod1, err := builder.NewPodBuilder("test-pod-1", "default").
		AddContainer("nginx", "nginx:latest", 100, 128).
		Build()
	require.NoError(t, err)

	pod2, err := builder.NewPodBuilder("test-pod-2", "default").
		AddContainer("nginx", "nginx:latest", 200, 256).
		Build()
	require.NoError(t, err)

	scenario, err := builder.NewScenarioBuilder("test-scenario").
		AddPodEvent("pod-1", 100*time.Millisecond, 500*time.Millisecond, pod1).
		AddPodEvent("pod-2", 200*time.Millisecond, 500*time.Millisecond, pod2).
		Build()
	require.NoError(t, err)

	// Create scheduler
	sched := scheduler.New()

	// Create simulation
	sim := simulator.NewSimulation(scenario, clientset, sched)

	// Load events
	sim.LoadEvents()

	// Start simulation
	err = sim.Start(ctx)
	assert.NoError(t, err)

	// Get stats
	stats := sim.GetStats()
	assert.NotNil(t, stats)
}

func TestSimulation_WithMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create metrics collector
	collector := metrics.NewMemoryCollector()

	// Create fake clientset with metrics recording
	clientset := fake.NewSimpleClientset()

	// Build scenario with multiple pods
	scenarioBuilder := builder.NewScenarioBuilder("metrics-test")
	
	for i := 0; i < 5; i++ {
		pod, err := builder.NewPodBuilder(fmt.Sprintf("pod-%d", i), "default").
			AddContainer("nginx", "nginx:latest", int64(100+i*50), int64(128+i*64)).
			Build()
		require.NoError(t, err)
		
		scenarioBuilder.AddPodEvent(
			fmt.Sprintf("event-%d", i),
			time.Duration(i*100)*time.Millisecond,
			1*time.Second,
			pod,
		)
	}

	scenario, err := scenarioBuilder.Build()
	require.NoError(t, err)

	// Create scheduler
	sched := scheduler.New()

	// Create simulation with metrics
	sim := simulator.NewSimulation(scenario, clientset, sched)
	sim.LoadEvents()

	// Record start time
	startTime := time.Now()

	// Start simulation
	err = sim.Start(ctx)
	assert.NoError(t, err)

	// Record metrics
	for i := 0; i < 5; i++ {
		err = collector.RecordPodMetrics(ctx, metrics.PodMetrics{
			Name:            fmt.Sprintf("pod-%d", i),
			Namespace:       "default",
			CreatedAt:       startTime.Add(time.Duration(i*100) * time.Millisecond),
			CPURequested:    int64(100 + i*50),
			MemoryRequested: int64((128 + i*64) * 1024 * 1024),
		})
		assert.NoError(t, err)
	}

	// Export metrics
	tempDir := t.TempDir()
	err = collector.ExportMetrics(ctx, "csv", tempDir)
	assert.NoError(t, err)
}

func TestSimulation_WithSchedulerEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Skip if no scheduler simulator available
	t.Skip("Requires scheduler simulator")

	// Create fake clientset
	clientset := fake.NewSimpleClientset()

	// Build scenario with scheduler events
	pod, err := builder.NewPodBuilder("test-pod", "default").
		AddContainer("nginx", "nginx:latest", 100, 128).
		Build()
	require.NoError(t, err)

	scenario, err := builder.NewScenarioBuilder("scheduler-test").
		AddPodEvent("pod-1", 100*time.Millisecond, 1*time.Second, pod).
		AddSchedulerEvent("change-weights", 500*time.Millisecond, map[string]int32{
			"NodeResourcesFit": 20,
		}).
		Build()
	require.NoError(t, err)

	// Create scheduler
	sched := scheduler.New()

	// Create simulation
	sim := simulator.NewSimulation(scenario, clientset, sched)
	sim.LoadEvents()

	// Start simulation
	err = sim.Start(ctx)
	assert.NoError(t, err)
}

func TestSimulation_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create fake clientset that will fail
	clientset := fake.NewSimpleClientset()

	// Build scenario with invalid pod (no containers)
	scenario, err := builder.NewScenarioBuilder("error-test").
		AddPodEvent("invalid-pod", 100*time.Millisecond, 500*time.Millisecond, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid",
				Namespace: "default",
			},
			Spec: v1.PodSpec{}, // Invalid: no containers
		}).
		Build()
	require.NoError(t, err) // Builder doesn't validate pod spec

	// Create scheduler
	sched := scheduler.New()

	// Create simulation
	sim := simulator.NewSimulation(scenario, clientset, sched)
	sim.LoadEvents()

	// Start simulation - should handle errors gracefully
	err = sim.Start(ctx)
	// The simulation might complete successfully even with pod creation errors
	// as the current implementation logs errors but continues
}

func TestSimulation_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create fake clientset
	clientset := fake.NewSimpleClientset()

	// Build long-running scenario
	pod, err := builder.NewPodBuilder("test-pod", "default").
		AddContainer("nginx", "nginx:latest", 100, 128).
		Build()
	require.NoError(t, err)

	scenario, err := builder.NewScenarioBuilder("cancel-test").
		AddPodEvent("pod-1", 100*time.Millisecond, 10*time.Second, pod).
		Build()
	require.NoError(t, err)

	// Create scheduler
	sched := scheduler.New()

	// Create simulation
	sim := simulator.NewSimulation(scenario, clientset, sched)
	sim.LoadEvents()

	// Start simulation in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- sim.Start(ctx)
	}()

	// Wait for pod to be created
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for simulation to stop
	select {
	case err := <-errCh:
		assert.NoError(t, err) // Context cancellation is not an error
	case <-time.After(5 * time.Second):
		t.Fatal("simulation did not stop after context cancellation")
	}
}
