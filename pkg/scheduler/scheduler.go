package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
)

// Scheduler interface defines the contract for the event scheduler
type Scheduler interface {
	// Start starts the scheduler
	Start(ctx context.Context) error
	// Stop gracefully stops the scheduler
	Stop() error
	// Schedule adds an event to the scheduling queue
	Schedule(event SchedulableEvent) error
	// GetEvents returns all events currently in the queue
	GetEvents() []SchedulableEvent
	// StartedAt returns the time when the scheduler was started
	StartedAt() time.Time
}

// scheduler is the main implementation of the Scheduler interface
type scheduler struct {
	logger    *logger.Logger
	queue     *Queue[SchedulableEvent]
	startTime time.Time
	running   bool
	mu        sync.RWMutex

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// Eviction timers for events with duration
	evictionTimers map[string]*time.Timer
	evictionMu     sync.RWMutex
}

// New creates a new scheduler
func New(log *logger.Logger) Scheduler {
	if log == nil {
		log = logger.Default()
	}

	return &scheduler{
		logger:         log,
		queue:          NewQueue[SchedulableEvent](),
		done:           make(chan struct{}),
		evictionTimers: make(map[string]*time.Timer),
	}
}

// Start starts the scheduler
func (s *scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrSchedulerAlreadyStarted
	}

	s.logger.Info("starting scheduler")
	s.startTime = time.Now()
	s.running = true

	// Create cancellable context
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start the main scheduler loop
	go s.schedulerLoop()

	s.logger.Info("scheduler started")
	return nil
}

// Stop gracefully stops the scheduler
func (s *scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrSchedulerNotStarted
	}

	s.logger.Info("stopping scheduler")
	s.running = false

	// Cancel context to signal stop
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for scheduler loop to finish
	<-s.done

	// Cancel all eviction timers
	s.cancelAllEvictionTimers()

	s.logger.Info("scheduler stopped")
	return nil
}

// Schedule adds an event to the queue
func (s *scheduler) Schedule(event SchedulableEvent) error {
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return ErrSchedulerNotStarted
	}
	s.mu.RUnlock()

	if event == nil {
		return ErrInvalidEvent
	}

	if err := s.queue.PushEvent(event); err != nil {
		s.logger.Errorf("failed to schedule event %s: %v", event.GetID(), err)
		return err
	}

	s.logger.Debugf("event scheduled: %s (type: %s, arrival: %v)",
		event.GetID(), event.GetType(), event.Arrival())

	return nil
}

// GetEvents returns all events currently in the queue
func (s *scheduler) GetEvents() []SchedulableEvent {
	return s.queue.GetEvents()
}

// StartedAt returns the time when the scheduler was started
func (s *scheduler) StartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

// schedulerLoop is the main event processing loop
func (s *scheduler) schedulerLoop() {
	defer close(s.done)

	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processReadyEvents()
		}
	}
}

// processReadyEvents checks for and processes events that are ready to execute
func (s *scheduler) processReadyEvents() {
	now := time.Since(s.startTime)

	for {
		event, err := s.queue.Peek()
		if err != nil {
			// Queue is empty
			break
		}

		if event.Arrival() > now {
			// Next event is not ready yet
			break
		}

		// Remove event from queue and execute it
		event, err = s.queue.PopEvent()
		if err != nil {
			break
		}

		s.executeEvent(event)
	}
}

// executeEvent executes a single event
func (s *scheduler) executeEvent(event SchedulableEvent) {
	s.logger.Debugf("executing event: %s", event.GetID())

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Execute the event
	if err := event.Execute(ctx); err != nil {
		event.SetStatus(EventStatusFailed)
		s.logger.Errorf("event execution failed: %s - %v", event.GetID(), err)
	} else {
		event.SetStatus(EventStatusCompleted)
		s.logger.Debugf("event executed successfully: %s", event.GetID())
	}

	// Schedule eviction if the event has a duration
	if event.Eviction() != nil {
		s.scheduleEviction(event)
	}
}

// scheduleEviction schedules an eviction for an event after its duration
func (s *scheduler) scheduleEviction(event SchedulableEvent) {
	evictionTime := *event.Eviction()

	timer := time.AfterFunc(evictionTime, func() {
		s.handleEviction(event)
	})

	s.evictionMu.Lock()
	s.evictionTimers[event.GetID()] = timer
	s.evictionMu.Unlock()

	s.logger.Debugf("eviction scheduled for event %s in %v", event.GetID(), evictionTime)
}

// handleEviction handles the eviction of an event
func (s *scheduler) handleEviction(event SchedulableEvent) {
	s.logger.Infof("evicting event: %s", event.GetID())

	// Remove timer from map
	s.evictionMu.Lock()
	delete(s.evictionTimers, event.GetID())
	s.evictionMu.Unlock()

	// Create eviction context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Use the event's custom eviction function
	if err := event.EvictionFn(ctx); err != nil {
		s.logger.Errorf("event eviction failed: %s - %v", event.GetID(), err)
	} else {
		s.logger.Infof("event evicted successfully: %s", event.GetID())
	}
}

// cancelAllEvictionTimers cancels all pending eviction timers
func (s *scheduler) cancelAllEvictionTimers() {
	s.evictionMu.Lock()
	defer s.evictionMu.Unlock()

	for eventID, timer := range s.evictionTimers {
		timer.Stop()
		delete(s.evictionTimers, eventID)
	}
}
