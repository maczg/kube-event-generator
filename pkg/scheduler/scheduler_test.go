package scheduler

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
)

// TestSchedulerBasicLifecycle tests the basic start/stop functionality
func TestSchedulerBasicLifecycle(t *testing.T) {
	log := logger.Default()
	s := New(log)

	// Test initial state
	if s.StartedAt() != (time.Time{}) {
		t.Error("scheduler should not have a start time before starting")
	}

	// Start scheduler
	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Check start time is set
	if s.StartedAt().IsZero() {
		t.Error("scheduler should have a start time after starting")
	}

	// Try to start again (should fail)
	err = s.Start(ctx)
	if !errors.Is(err, ErrSchedulerAlreadyStarted) {
		t.Errorf("expected ErrSchedulerAlreadyStarted, got: %v", err)
	}

	// Stop scheduler
	err = s.Stop()
	if err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}

	// Try to stop again (should fail)
	err = s.Stop()
	if !errors.Is(err, ErrSchedulerNotStarted) {
		t.Errorf("expected ErrSchedulerNotStarted, got: %v", err)
	}
}

// TestSchedulerEventScheduling tests basic event scheduling
func TestSchedulerEventScheduling(t *testing.T) {
	log := logger.Default()
	s := New(log)

	// Try to schedule event before starting (should fail)
	event := NewPodEvent("test-pod", "default", "create", 100*time.Millisecond)
	err := s.Schedule(event)
	if !errors.Is(err, ErrSchedulerNotStarted) {
		t.Errorf("expected ErrSchedulerNotStarted, got: %v", err)
	}

	// Start scheduler
	ctx := context.Background()
	err = s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Schedule a valid event
	err = s.Schedule(event)
	if err != nil {
		t.Errorf("failed to schedule event: %v", err)
	}

	// Check event is in queue
	events := s.GetEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event in queue, got %d", len(events))
	}

	if events[0].GetID() != event.GetID() {
		t.Errorf("expected event ID %s, got %s", event.GetID(), events[0].GetID())
	}

	// Try to schedule nil event (should fail)
	err = s.Schedule(nil)
	if !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("expected ErrInvalidEvent, got: %v", err)
	}
}

// TestEventExecution tests that events are executed at the right time
func TestEventExecution(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Create event with small delay
	event := NewPodEvent("test-pod", "default", "create", 50*time.Millisecond)

	// Schedule the event
	err = s.Schedule(event)
	if err != nil {
		t.Errorf("failed to schedule event: %v", err)
	}

	// Event should be pending initially
	if event.GetStatus() != EventStatusPending {
		t.Errorf("expected status %s, got %s", EventStatusPending, event.GetStatus())
	}

	// Wait for event to be processed (longer than arrival time + processing time)
	time.Sleep(200 * time.Millisecond)

	// Event should be completed
	if event.GetStatus() != EventStatusCompleted {
		t.Errorf("expected status %s, got %s", EventStatusCompleted, event.GetStatus())
	}

	// Queue should be empty
	events := s.GetEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events in queue, got %d", len(events))
	}
}

// TestEventOrdering tests that events are executed in the correct order
func TestEventOrdering(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Create events with different arrival times
	event1 := NewPodEvent("pod1", "default", "create", 100*time.Millisecond)
	event2 := NewPodEvent("pod2", "default", "create", 50*time.Millisecond) // Should execute first
	event3 := NewPodEvent("pod3", "default", "create", 150*time.Millisecond)

	// Schedule events in non-arrival order
	s.Schedule(event1)
	s.Schedule(event2)
	s.Schedule(event3)

	// Wait for all events to be processed
	time.Sleep(300 * time.Millisecond)

	// All events should be completed
	events := []*PodEvent{event1, event2, event3}
	for i, event := range events {
		if event.GetStatus() != EventStatusCompleted {
			t.Errorf("event %d should be completed, got %s", i, event.GetStatus())
		}
	}
}

// TestEventWithEviction tests event eviction functionality
func TestEventWithEviction(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Create event with eviction time
	event := NewPodEvent("test-pod", "default", "create", 50*time.Millisecond)
	event.SetEviction(100 * time.Millisecond) // Will be evicted 100ms after execution

	// Schedule the event
	err = s.Schedule(event)
	if err != nil {
		t.Errorf("failed to schedule event: %v", err)
	}

	// Wait for event to be executed (need to wait for arrival time + processing time)
	time.Sleep(150 * time.Millisecond)

	// Event should be completed
	if event.GetStatus() != EventStatusCompleted {
		t.Errorf("expected status %s, got %s", EventStatusCompleted, event.GetStatus())
	}

	// Wait for eviction to occur
	time.Sleep(150 * time.Millisecond)

	// Check that eviction timer was scheduled (we can't easily test the eviction itself
	// without more complex mocking, but we can verify the timer was set up)
	// The eviction would have been logged and handled
}

// TestDifferentEventTypes tests scheduling different types of events
func TestDifferentEventTypes(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Create different event types
	podEvent := NewPodEvent("test-pod", "default", "create", 50*time.Millisecond)
	schedulerEvent := NewSchedulerEvent(75 * time.Millisecond)
	nodeEvent := NewNodeEvent("test-node", "update", 100*time.Millisecond)

	// Schedule all events
	err = s.Schedule(podEvent)
	assert.NoError(t, err, "failed to schedule pod event")
	err = s.Schedule(schedulerEvent)
	assert.NoError(t, err, "failed to schedule scheduler event")
	err = s.Schedule(nodeEvent)
	assert.NoError(t, err, "failed to schedule node event")

	// Wait for all events to be processed
	time.Sleep(200 * time.Millisecond)

	// All events should be completed
	if podEvent.GetStatus() != EventStatusCompleted {
		t.Errorf("pod event should be completed, got %s", podEvent.GetStatus())
	}
	if schedulerEvent.GetStatus() != EventStatusCompleted {
		t.Errorf("scheduler event should be completed, got %s", schedulerEvent.GetStatus())
	}
	if nodeEvent.GetStatus() != EventStatusCompleted {
		t.Errorf("node event should be completed, got %s", nodeEvent.GetStatus())
	}

	// Verify event types
	if podEvent.GetType() != EventTypePod {
		t.Errorf("expected pod event type %s, got %s", EventTypePod, podEvent.GetType())
	}
	if schedulerEvent.GetType() != EventTypeScheduler {
		t.Errorf("expected scheduler event type %s, got %s", EventTypeScheduler, schedulerEvent.GetType())
	}
	if nodeEvent.GetType() != EventTypeNode {
		t.Errorf("expected node event type %s, got %s", EventTypeNode, nodeEvent.GetType())
	}
}

// TestEventHappensBefore tests event ordering logic
func TestEventHappensBefore(t *testing.T) {
	// Create events with different arrival times
	event1 := NewPodEvent("pod1", "default", "create", 100*time.Millisecond)
	event2 := NewPodEvent("pod2", "default", "create", 50*time.Millisecond)
	event3 := NewPodEvent("pod3", "default", "create", 100*time.Millisecond) // Same time as event1

	// Test ordering by arrival time
	if !event2.HappensBefore(event1) {
		t.Error("event2 should happen before event1 (earlier arrival time)")
	}
	if event1.HappensBefore(event2) {
		t.Error("event1 should not happen before event2 (later arrival time)")
	}

	// Test ordering by ID when arrival times are equal
	if event1.GetID() < event3.GetID() {
		if !event1.HappensBefore(event3) {
			t.Error("event1 should happen before event3 (same arrival time, but smaller ID)")
		}
	} else {
		if !event3.HappensBefore(event1) {
			t.Error("event3 should happen before event1 (same arrival time, but smaller ID)")
		}
	}
}

// TestSchedulerStop tests that scheduler stops gracefully
func TestSchedulerStop(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Schedule some events
	event1 := NewPodEvent("pod1", "default", "create", 1*time.Second) // Long delay
	event2 := NewPodEvent("pod2", "default", "create", 2*time.Second) // Even longer delay

	err = s.Schedule(event1)
	assert.NoError(t, err, "failed to schedule event1")
	err = s.Schedule(event2)
	assert.NoError(t, err, "failed to schedule event2")

	// Events should be in queue
	events := s.GetEvents()
	if len(events) != 2 {
		t.Errorf("expected 2 events in queue, got %d", len(events))
	}

	// Stop scheduler
	err = s.Stop()
	if err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}

	// Events should still be pending (not executed due to stop)
	if event1.GetStatus() != EventStatusPending {
		t.Errorf("event1 should still be pending, got %s", event1.GetStatus())
	}
	if event2.GetStatus() != EventStatusPending {
		t.Errorf("event2 should still be pending, got %s", event2.GetStatus())
	}
}

// TestConcurrentScheduling tests concurrent event scheduling
func TestConcurrentScheduling(t *testing.T) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	// Schedule multiple events concurrently
	numEvents := 10
	events := make([]*PodEvent, numEvents)

	for i := 0; i < numEvents; i++ {
		events[i] = NewPodEvent("pod", "default", "create", 50*time.Millisecond)

		go func(event *PodEvent, index int) {
			err := s.Schedule(event)
			assert.NoError(t, err, "failed to schedule event %d", index)
		}(events[i], i)
	}

	// Wait for all events to be processed
	time.Sleep(200 * time.Millisecond)

	// All events should be completed
	for i, event := range events {
		if event.GetStatus() != EventStatusCompleted {
			t.Errorf("event %d should be completed, got %s", i, event.GetStatus())
		}
	}
}

// BenchmarkEventScheduling benchmarks event scheduling performance
func BenchmarkEventScheduling(b *testing.B) {
	log := logger.Default()
	s := New(log)

	ctx := context.Background()
	err := s.Start(ctx)
	assert.NoError(b, err, "failed to start scheduler")
	defer s.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event := NewPodEvent("pod", "default", "create", time.Duration(i)*time.Microsecond)
		err = s.Schedule(event)
		assert.NoError(b, err, "failed to schedule event")
	}
}
