package scheduler

import (
	"context"
	"time"
)

// KubeSchedulerEvent represents a scheduler configuration change event
type KubeSchedulerEvent struct {
	*BaseEvent
	ConfigChanges map[string]interface{} `json:"config_changes"`
	PluginWeights map[string]int32       `json:"plugin_weights,omitempty"`
}

// NewSchedulerEvent creates a new KubeSchedulerEvent
func NewSchedulerEvent(arrivalTime time.Duration) *KubeSchedulerEvent {
	return &KubeSchedulerEvent{
		BaseEvent:     NewBaseEvent(EventTypeScheduler, arrivalTime),
		ConfigChanges: make(map[string]interface{}),
		PluginWeights: make(map[string]int32),
	}
}

// Execute implements the scheduler-specific execution logic
func (e *KubeSchedulerEvent) Execute(ctx context.Context) error {
	e.SetStatus(EventStatusExecuting)
	defer func() {
		if e.GetStatus() == EventStatusExecuting {
			e.SetStatus(EventStatusCompleted)
		}
	}()

	// Scheduler-specific execution logic would go here
	// This would integrate with the scheduler manager
	return nil
}

// EvictionFn implements scheduler-specific eviction logic
func (e *KubeSchedulerEvent) EvictionFn(ctx context.Context) error {
	// For scheduler events, eviction might mean reverting configuration changes
	// This would integrate with the scheduler manager to revert changes
	return nil
}
