package scheduler

import (
	"context"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"time"
)

// PodEvent represents a pod-related event (creation, deletion, etc.)
type PodEvent struct {
	*BaseEvent
	PodName   string                 `json:"pod_name"`
	Namespace string                 `json:"namespace"`
	Action    string                 `json:"action"` // "create", "delete", "update"
	PodSpec   map[string]interface{} `json:"pod_spec,omitempty"`
}

// NewPodEvent creates a new PodEvent
func NewPodEvent(podName, namespace, action string, arrivalTime time.Duration) *PodEvent {
	return &PodEvent{
		BaseEvent: NewBaseEvent(EventTypePod, arrivalTime),
		PodName:   podName,
		Namespace: namespace,
		Action:    action,
		PodSpec:   make(map[string]interface{}),
	}
}

// Execute implements the pod-specific execution logic
func (e *PodEvent) Execute(ctx context.Context) error {
	e.SetStatus(EventStatusExecuting)
	defer func() {
		if e.GetStatus() == EventStatusExecuting {
			e.SetStatus(EventStatusCompleted)
		}
	}()

	logger.Default().Infoln("Executing PodEvent")
	return nil
}

// EvictionFn implements pod-specific eviction logic
func (e *PodEvent) EvictionFn(ctx context.Context) error {
	// For pod events, eviction typically means deleting the pod
	// This would integrate with the Kubernetes client to delete the pod
	logger.Default().Infoln("Executing PodEvent Eviction")
	return nil
}
