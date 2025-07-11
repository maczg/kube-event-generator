package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	EventTypePod       EventType = "pod"
	EventTypeScheduler EventType = "scheduler"
	EventTypeNode      EventType = "node"
	EventTypeCustom    EventType = "custom"
)

// EventStatus represents the current status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusExecuting EventStatus = "executing"
	EventStatusCompleted EventStatus = "completed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusCanceled  EventStatus = "canceled"
)

// SchedulableEvent interface defines the contract for all schedulable events
type SchedulableEvent interface {
	GetID() string
	GetType() EventType
	GetStatus() EventStatus
	SetStatus(status EventStatus)
	Arrival() time.Duration
	Eviction() *time.Duration
	Execute(ctx context.Context) error
	EvictionFn(ctx context.Context) error
	HappensBefore(other SchedulableEvent) bool
}

// BaseEvent provides a basic implementation of SchedulableEvent
type BaseEvent struct {
	ID          string         `json:"id"`
	Type        EventType      `json:"type"`
	Status      EventStatus    `json:"status"`
	ArrivalTime time.Duration  `json:"arrival_time"`
	EvictTime   *time.Duration `json:"evict_time,omitempty"`
	Priority    int            `json:"priority"`
	CreatedAt   time.Time      `json:"created_at"`
	mu          sync.RWMutex   // Protects Status field
}

// NewBaseEvent creates a new BaseEvent with default values
func NewBaseEvent(eventType EventType, arrivalTime time.Duration) *BaseEvent {
	return &BaseEvent{
		ID:          uuid.New().String(),
		Type:        eventType,
		Status:      EventStatusPending,
		ArrivalTime: arrivalTime,
		Priority:    0,
		CreatedAt:   time.Now(),
	}
}

// GetID returns the event ID
func (e *BaseEvent) GetID() string {
	return e.ID
}

// GetType returns the event type
func (e *BaseEvent) GetType() EventType {
	return e.Type
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
func (e *BaseEvent) Eviction() *time.Duration {
	return e.EvictTime
}

// SetEviction sets the eviction time duration
func (e *BaseEvent) SetEviction(evictTime time.Duration) {
	e.EvictTime = &evictTime
}

// Execute provides a default implementation - should be overridden by specific event types
func (e *BaseEvent) Execute(ctx context.Context) error {
	e.SetStatus(EventStatusExecuting)
	defer e.SetStatus(EventStatusCompleted)
	return fmt.Errorf("not implemented")

}

// EvictionFn provides a default eviction implementation - should be overridden by specific event types
func (e *BaseEvent) EvictionFn(ctx context.Context) error {
	return fmt.Errorf("not implemented")
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
	return fmt.Sprintf("Event{ID: %s, Type: %s, Status: %s, Arrival: %v}",
		e.ID, e.Type, e.Status, e.ArrivalTime)
}
