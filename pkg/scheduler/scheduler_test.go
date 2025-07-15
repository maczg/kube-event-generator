package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerWithEviction(t *testing.T) {
	// Create a test logger
	testLogger := logger.Default()

	// Create a new scheduler
	scheduler := New(testLogger)

	// Start the scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = scheduler.Stop() }()

	// Create a base event with eviction time
	arrivalTime := 100 * time.Millisecond
	evictionTime := 500 * time.Millisecond
	event := NewBaseEvent(arrivalTime, evictionTime)

	// Schedule the event
	err = scheduler.Schedule(event)
	require.NoError(t, err)

	// Verify the event is scheduled
	events := scheduler.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, event.GetID(), events[0].GetID())
	assert.Equal(t, EventStatusPending, event.GetStatus())

	// Wait for the event to be executed
	time.Sleep(200 * time.Millisecond)

	// Check that the event was executed
	assert.Equal(t, EventStatusCompleted, event.GetStatus())

	// Wait a bit more to see if eviction event was scheduled
	time.Sleep(200 * time.Millisecond)

	// Check if additional events were scheduled (eviction events)
	events = scheduler.GetEvents()
	// There should be at least one more event (the eviction event)
	assert.GreaterOrEqual(t, len(events), 1)
}

func TestBaseEventEviction(t *testing.T) {
	// Test BaseEvent with eviction time
	arrivalTime := 100 * time.Millisecond
	evictionTime := 500 * time.Millisecond
	event := NewBaseEvent(arrivalTime, evictionTime)

	// Verify initial state
	assert.NotEmpty(t, event.GetID())
	assert.Equal(t, EventStatusPending, event.GetStatus())
	assert.Equal(t, arrivalTime, event.Arrival())
	assert.Greater(t, event.Eviction(), time.Duration(0))
	assert.Equal(t, evictionTime, event.Eviction())

	// Test setting eviction time
	newEvictionTime := 1 * time.Second
	event.SetEviction(newEvictionTime)
	assert.Equal(t, newEvictionTime, event.Eviction())
}

func TestBaseEventWithoutEviction(t *testing.T) {
	// Test BaseEvent without eviction time
	arrivalTime := 100 * time.Millisecond
	event := NewBaseEvent(arrivalTime, 0)

	// Verify initial state
	assert.NotEmpty(t, event.GetID())
	assert.Equal(t, EventStatusPending, event.GetStatus())
	assert.Equal(t, arrivalTime, event.Arrival())
	assert.Equal(t, time.Duration(0), event.Eviction())
}

func TestBaseEventExecution(t *testing.T) {
	// Create a test logger
	testLogger := logger.Default()

	// Create a scheduler to pass in context
	scheduler := New(testLogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = scheduler.Stop() }()

	// Create context with scheduler
	execCtx := context.WithValue(ctx, SchedulerContextKey, scheduler)

	// Test event execution with eviction
	arrivalTime := 100 * time.Millisecond
	evictionTime := 500 * time.Millisecond
	event := NewBaseEvent(arrivalTime, evictionTime)

	// Execute the event
	err = event.Execute(execCtx)
	require.NoError(t, err)

	// Verify the event status changed
	assert.Equal(t, EventStatusCompleted, event.GetStatus())

	// Wait a bit to allow eviction event to be scheduled
	time.Sleep(100 * time.Millisecond)

	// Check if eviction event was scheduled
	events := scheduler.GetEvents()
	assert.GreaterOrEqual(t, len(events), 1)
}

func TestEventOrdering(t *testing.T) {
	// Test that events are ordered correctly by arrival time
	event1 := NewBaseEvent(100*time.Millisecond, 0)
	event2 := NewBaseEvent(200*time.Millisecond, 0)
	event3 := NewBaseEvent(50*time.Millisecond, 0)

	// Test HappensBefore logic
	assert.True(t, event3.HappensBefore(event1))
	assert.True(t, event1.HappensBefore(event2))
	assert.False(t, event2.HappensBefore(event1))
}

func TestEventTimeout(t *testing.T) {
	// Test event execution timeout
	event := NewBaseEvent(100*time.Millisecond, 0)

	// Verify default timeout
	assert.Equal(t, 30*time.Second, event.GetExecuteTimeout())

	// Test setting custom timeout
	customTimeout := 5 * time.Second
	event.SetExecuteTimeout(customTimeout)
	assert.Equal(t, customTimeout, event.GetExecuteTimeout())
}
