package scheduler

import (
	"context"
	"time"
)

// NodeEvent represents a node-related event (resource updates, etc.)
type NodeEvent struct {
	*BaseEvent
	NodeName  string                 `json:"node_name"`
	Action    string                 `json:"action"` // "update", "add", "remove"
	Resources map[string]interface{} `json:"resources,omitempty"`
}

// NewNodeEvent creates a new NodeEvent
func NewNodeEvent(nodeName, action string, arrivalTime time.Duration) *NodeEvent {
	return &NodeEvent{
		BaseEvent: NewBaseEvent(EventTypeNode, arrivalTime),
		NodeName:  nodeName,
		Action:    action,
		Resources: make(map[string]interface{}),
	}
}

// Execute implements the node-specific execution logic
func (e *NodeEvent) Execute(ctx context.Context) error {
	e.SetStatus(EventStatusExecuting)
	defer func() {
		if e.GetStatus() == EventStatusExecuting {
			e.SetStatus(EventStatusCompleted)
		}
	}()

	// Node-specific execution logic would go here
	// This would integrate with the Kubernetes client
	return nil
}

// EvictionFn implements node-specific eviction logic
func (e *NodeEvent) EvictionFn(ctx context.Context) error {
	// For node events, eviction might mean reverting resource changes
	// This would integrate with the Kubernetes client to revert node changes
	return nil
}
