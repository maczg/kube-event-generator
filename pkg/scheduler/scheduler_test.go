package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/maczg/kube-event-generator/pkg/errors"
	testhelpers "github.com/maczg/kube-event-generator/pkg/testing"
)

// mockSchedulable implements Schedulable interface for testing.
type mockSchedulable struct {
	executeFunc  func(context.Context) error
	id           string
	after        time.Duration
	duration     time.Duration
	executeCalls int
	mu           sync.Mutex
}

func (m *mockSchedulable) ID() string {
	return m.id
}

func (m *mockSchedulable) ExecuteAfterDuration() time.Duration {
	return m.after
}

func (m *mockSchedulable) ExecuteForDuration() time.Duration {
	return m.duration
}

func (m *mockSchedulable) Execute(ctx context.Context) error {
	m.mu.Lock()
	m.executeCalls++
	m.mu.Unlock()

	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}

	return nil
}

func (m *mockSchedulable) ComparePriority(other Schedulable) bool {
	return m.ExecuteAfterDuration() < other.ExecuteAfterDuration()
}

func (m *mockSchedulable) getExecuteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.executeCalls
}

func TestScheduler_Schedule(t *testing.T) {
	s := New()

	events := []Schedulable{
		&mockSchedulable{id: "event1", after: 100 * time.Millisecond},
		&mockSchedulable{id: "event2", after: 50 * time.Millisecond},
		&mockSchedulable{id: "event3", after: 150 * time.Millisecond},
	}

	for _, e := range events {
		s.Schedule(e)
	}

	scheduledEvents := s.GetEvents()
	assert.Len(t, scheduledEvents, 3)

	// Verify events are in priority order (shortest duration first).
	assert.Equal(t, "event2", scheduledEvents[0].ID())
	assert.Equal(t, "event1", scheduledEvents[1].ID())
	assert.Equal(t, "event3", scheduledEvents[2].ID())
}

func TestScheduler_StartStop(t *testing.T) {
	ctx, cancel := testhelpers.TestContext(t, 5*time.Second)
	defer cancel()

	s := New()

	// Test starting scheduler.
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Give scheduler time to start.
	time.Sleep(50 * time.Millisecond)

	// Test double start.
	err := s.Start(ctx)
	assert.ErrorIs(t, err, errors.ErrSchedulerAlreadyRunning)

	// Test stop.
	err = s.Stop()
	assert.NoError(t, err)

	// Wait for Start to return.
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("scheduler did not stop in time")
	}

	// Test double stop.
	err = s.Stop()
	assert.ErrorIs(t, err, errors.ErrSchedulerNotRunning)
}

func TestScheduler_ExecuteEvents(t *testing.T) {
	ctx, cancel := testhelpers.TestContext(t, 5*time.Second)
	defer cancel()

	s := New()

	executed := make(chan string, 3)
	events := []*mockSchedulable{
		{
			id:    "event1",
			after: 50 * time.Millisecond,
			executeFunc: func(ctx context.Context) error {
				executed <- "event1"
				return nil
			},
		},
		{
			id:    "event2",
			after: 100 * time.Millisecond,
			executeFunc: func(ctx context.Context) error {
				executed <- "event2"
				return nil
			},
		},
		{
			id:    "event3",
			after: 150 * time.Millisecond,
			executeFunc: func(ctx context.Context) error {
				executed <- "event3"
				return nil
			},
		},
	}

	// Schedule events.
	for _, e := range events {
		s.Schedule(e)
	}

	// Start scheduler.
	go func() {
		s.Start(ctx)
	}()

	// Collect executed events.
	var executedOrder []string

	for i := 0; i < 3; i++ {
		select {
		case eventID := <-executed:
			executedOrder = append(executedOrder, eventID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for events to execute")
		}
	}

	// Verify execution order.
	assert.Equal(t, []string{"event1", "event2", "event3"}, executedOrder)

	// Stop scheduler.
	err := s.Stop()
	assert.NoError(t, err)
}

func TestScheduler_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	s := New()

	// Add an event that won't execute before cancellation.
	event := &mockSchedulable{
		id:    "event1",
		after: 1 * time.Second,
		executeFunc: func(ctx context.Context) error {
			t.Fatal("event should not execute")
			return nil
		},
	}
	s.Schedule(event)

	// Start scheduler.
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Give scheduler time to start.
	time.Sleep(50 * time.Millisecond)

	// Cancel context.
	cancel()

	// Verify scheduler stops.
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}

	// Verify event was not executed.
	assert.Equal(t, 0, event.getExecuteCalls())
}

func TestScheduler_ErrorHandling(t *testing.T) {
	ctx, cancel := testhelpers.TestContext(t, 5*time.Second)
	defer cancel()

	s := New()

	errorOccurred := make(chan bool, 1)

	// Add event that returns error.
	event := &mockSchedulable{
		id:    "error-event",
		after: 50 * time.Millisecond,
		executeFunc: func(ctx context.Context) error {
			errorOccurred <- true
			return assert.AnError
		},
	}
	s.Schedule(event)

	// Start scheduler.
	go func() {
		s.Start(ctx)
	}()

	// Wait for error to occur.
	select {
	case <-errorOccurred:
		// Error was handled, scheduler should continue.
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for error event")
	}

	// Add another event to verify scheduler continues.
	successEvent := &mockSchedulable{
		id:    "success-event",
		after: 100 * time.Millisecond,
		executeFunc: func(ctx context.Context) error {
			return nil
		},
	}
	s.Schedule(successEvent)

	// Give time for second event to execute.
	time.Sleep(200 * time.Millisecond)

	// Verify second event executed.
	assert.Equal(t, 1, successEvent.getExecuteCalls())

	// Stop scheduler.
	err := s.Stop()
	assert.NoError(t, err)
}

func TestScheduler_ConcurrentScheduling(t *testing.T) {
	ctx, cancel := testhelpers.TestContext(t, 5*time.Second)
	defer cancel()

	s := New()

	// Start scheduler.
	go func() {
		s.Start(ctx)
	}()

	// Give scheduler time to start.
	time.Sleep(50 * time.Millisecond)

	// Schedule events concurrently.
	var wg sync.WaitGroup

	executed := make(chan string, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			event := &mockSchedulable{
				id:    string(rune('a' + id)),
				after: time.Duration(id*10) * time.Millisecond,
				executeFunc: func(ctx context.Context) error {
					executed <- string(rune('a' + id))
					return nil
				},
			}
			s.Schedule(event)
		}(i)
	}

	wg.Wait()

	// Collect all executed events.
	var executedEvents []string

	for i := 0; i < 10; i++ {
		select {
		case event := <-executed:
			executedEvents = append(executedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for events, got %d of 10", len(executedEvents))
		}
	}

	// Verify all events executed.
	assert.Len(t, executedEvents, 10)

	// Stop scheduler.
	err := s.Stop()
	assert.NoError(t, err)
}

func TestScheduler_StartedAt(t *testing.T) {
	ctx, cancel := testhelpers.TestContext(t, 1*time.Second)
	defer cancel()

	s := New()

	// StartedAt should be zero before start.
	assert.True(t, s.StartedAt().IsZero())

	beforeStart := time.Now()

	// Start scheduler.
	go func() {
		s.Start(ctx)
	}()

	// Give scheduler time to start.
	time.Sleep(50 * time.Millisecond)

	startedAt := s.StartedAt()
	assert.False(t, startedAt.IsZero())
	assert.True(t, startedAt.After(beforeStart) || startedAt.Equal(beforeStart))
	assert.True(t, startedAt.Before(time.Now()))

	// Stop scheduler.
	err := s.Stop()
	assert.NoError(t, err)
}
