package scheduler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"sync"
	"time"
)

// BaseEvent provides a basic implementation of SchedulableEvent
type BaseEvent struct {
	ID             string        `json:"id"`
	Status         EventStatus   `json:"status"`
	ArrivalTime    time.Duration `json:"arrivalTime"`
	EvictTime      time.Duration `json:"evictTime"`       // Zero value means no eviction
	ExecuteTimeout time.Duration `json:"execute_timeout"` // Timeout for event execution
	CreatedAt      time.Time     `json:"created_at"`
	mu             sync.RWMutex  // Protects Status field
}

// NewBaseEvent creates a new BaseEvent with default values
func NewBaseEvent(arrivalTime time.Duration, evictionTime time.Duration) *BaseEvent {
	return &BaseEvent{
		ID:             uuid.New().String(),
		Status:         EventStatusPending,
		ArrivalTime:    arrivalTime,
		EvictTime:      evictionTime,     // Zero value means no eviction
		ExecuteTimeout: 30 * time.Second, // Default execution timeout
		CreatedAt:      time.Now(),
	}
}

// GetID returns the event ID
func (e *BaseEvent) GetID() string {
	return e.ID
}

// GetStatus returns the current event status
func (e *BaseEvent) GetStatus() EventStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Status
}

// SetStatus sets the event status
func (e *BaseEvent) SetStatus(status EventStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Status = status
}

// Arrival returns the arrival time duration
func (e *BaseEvent) Arrival() time.Duration {
	return e.ArrivalTime
}

// Eviction returns the eviction time duration
func (e *BaseEvent) Eviction() time.Duration {
	return e.EvictTime
}

// SetEviction sets the eviction time duration
func (e *BaseEvent) SetEviction(evictTime time.Duration) {
	e.EvictTime = evictTime
}

// Execute provides a default implementation - should be overridden by specific event types
func (e *BaseEvent) Execute(ctx context.Context) error {
	e.SetStatus(EventStatusExecuting)
	defer e.SetStatus(EventStatusCompleted)

	log := logger.Default()
	log.Infof("Executing event %s", e.ID)

	// Handle eviction scheduling if eviction time is set
	if e.EvictTime > 0 {
		log.Infof("Event %s has eviction time set to %v", e.ID, e.EvictTime)

		if scheduler, ok := ctx.Value(SchedulerContextKey).(Scheduler); ok {
			log.Infof("Scheduling eviction for event %s", e.ID)
			evictionEvent := NewBaseEvent(e.EvictTime, 0)

			if err := scheduler.Schedule(evictionEvent); err != nil {
				log.Errorf("Failed to schedule eviction event for %s: %v", e.ID, err)
				return err
			}
		} else {
			log.Warnf("No scheduler found in context for event %s eviction", e.ID)
		}
	}

	return nil
}

// HappensBefore determines if this event should be executed before another event
func (e *BaseEvent) HappensBefore(other SchedulableEvent) bool {
	// Compare by arrival time (earlier events first)
	if e.ArrivalTime != other.Arrival() {
		return e.ArrivalTime < other.Arrival()
	}
	// If arrival times are equal, compare by ID for deterministic ordering
	return e.ID < other.GetID()
}

// String returns a string representation of the event
func (e *BaseEvent) String() string {
	return fmt.Sprintf("Event{ID: %s, Status: %s, Arrival: %v, ExecuteFor: %v}",
		e.ID, e.Status, e.ArrivalTime, e.EvictTime)
}

// GetExecuteTimeout returns the execution timeout for the event
func (e *BaseEvent) GetExecuteTimeout() time.Duration {
	// Default timeout can be overridden by specific event types
	return e.ExecuteTimeout
}

// SetExecuteTimeout sets the execution timeout for the event
func (e *BaseEvent) SetExecuteTimeout(timeout time.Duration) {
	e.ExecuteTimeout = timeout
}
