package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"k8s.io/client-go/kubernetes"
	"time"

	kube "github.com/maczg/kube-event-generator/pkg/kubernetes"
	"github.com/maczg/kube-event-generator/pkg/logger"
	eventscheduler "github.com/maczg/kube-event-generator/pkg/scheduler"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodEventType represents the type of pod event
type PodEventType string

const (
	// PodEventTypeCreate represents a pod creation event
	PodEventTypeCreate PodEventType = "create"
	// PodEventTypeDelete represents a pod deletion event
	PodEventTypeDelete PodEventType = "delete"
)

// PodEvent represents a pod-related event (creation, deletion, etc.)
type PodEvent struct {
	*eventscheduler.BaseEvent
	// ArrivalTime is the time when the event arrives in the scheduler
	ArrivalTime EventDuration `yaml:"arrivalTime" json:"arrivalTime"`
	// EvictTime is the time when the pod should be evicted after creation
	EvictTime EventDuration `yaml:"evictTime" json:"evictTime"`

	// PodSpec is the specification of the pod to be created or deleted
	PodSpec *v1.Pod `yaml:"pod" json:"podSpec"`
	// EventType indicates the type of pod event (create or delete)
	EventType PodEventType `json:"eventType"`
}

// NewCreatePodEvent creates a new pod creation event
func NewCreatePodEvent(arrivalTime, evictionTime time.Duration, spec *v1.Pod) *PodEvent {
	return &PodEvent{
		BaseEvent: eventscheduler.NewBaseEvent(arrivalTime, evictionTime),
		PodSpec:   spec,
		EventType: PodEventTypeCreate,
	}
}

// NewDeletePodEvent creates a new pod deletion event
func NewDeletePodEvent(arrivalTime time.Duration, spec *v1.Pod) *PodEvent {
	return &PodEvent{
		BaseEvent: eventscheduler.NewBaseEvent(arrivalTime, 0),
		PodSpec:   spec,
		EventType: PodEventTypeDelete,
	}
}

// Execute implements the pod-specific execution logic
func (e *PodEvent) Execute(ctx context.Context) error {
	e.SetStatus(eventscheduler.EventStatusExecuting)
	defer func() {
		if e.GetStatus() == eventscheduler.EventStatusExecuting {
			e.SetStatus(eventscheduler.EventStatusCompleted)
		}
	}()

	if e.PodSpec == nil {
		return errors.New("pod spec is nil")
	}

	switch e.EventType {
	case PodEventTypeCreate:
		if err := e.createAndWatch(ctx); err != nil {
			logger.Default().Errorf("failed to create and watch pod %s: %v", e.PodSpec.Name, err)
			return err
		}
	case PodEventTypeDelete:
		if err := e.Evict(ctx); err != nil {
			logger.Default().Errorf("failed to evict pod %s: %v", e.PodSpec.Name, err)
			return err
		}
	default:
		return errors.New("unknown pod event type")
	}

	return nil
}

// createAndWatch creates a pod and watches for its running state to schedule eviction if needed
func (e *PodEvent) createAndWatch(ctx context.Context) error {
	clientset, err := kube.GetClientset()
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(e.PodSpec.Namespace).Create(ctx, e.PodSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	logger.Default().Infof("event %s with pod %s created successfully", e.GetID(), e.PodSpec.Name)

	// If no eviction time is set, we're done
	if e.EvictTime <= 0 {
		return nil
	}

	// Watch for pod to become running, then schedule eviction
	return e.watchAndScheduleEviction(ctx, clientset)
}

// watchAndScheduleEviction watches for the pod to become running and schedules its eviction
func (e *PodEvent) watchAndScheduleEviction(ctx context.Context, clientset *kubernetes.Clientset) error {
	watcher, err := clientset.CoreV1().Pods(e.PodSpec.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + e.PodSpec.Name,
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("context canceled while waiting for pod to start")
		case event := <-watcher.ResultChan():
			if pod, ok := event.Object.(*v1.Pod); ok && pod.Status.Phase == v1.PodRunning {
				return e.scheduleEviction(ctx, pod)
			}
		}
	}
}

// scheduleEviction schedules the eviction of a running pod
func (e *PodEvent) scheduleEviction(ctx context.Context, pod *v1.Pod) error {
	scheduler, ok := ctx.Value(eventscheduler.SchedulerContextKey).(eventscheduler.Scheduler)
	if !ok {
		return errors.New("scheduler not found in context")
	}

	evictionTime := time.Since(scheduler.StartedAt()) + e.EvictTime.Duration()
	evictEvent := NewDeletePodEvent(evictionTime, e.PodSpec)

	if err := scheduler.Schedule(evictEvent); err != nil {
		return err
	}

	logger.Default().Debugf("scheduled eviction for pod %s at %s", pod.Name, evictionTime)
	return nil
}

// Evict deletes the pod from the cluster
func (e *PodEvent) Evict(ctx context.Context) error {
	clientset, err := kube.GetClientset()
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().Pods(e.PodSpec.Namespace).Delete(ctx, e.PodSpec.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	logger.Default().Infof("pod %s deleted successfully", e.PodSpec.Name)
	return nil
}

// UnmarshalJSON implements custom JSON unmarshalling for PodEvent.
// It converts the EventDuration fields from JSON numbers to time.Duration.
func (e *PodEvent) UnmarshalJSON(data []byte) error {
	type Alias PodEvent
	var temp Alias

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	*e = PodEvent{
		BaseEvent: eventscheduler.NewBaseEvent(
			temp.ArrivalTime.Duration(),
			temp.EvictTime.Duration(),
		),
		PodSpec:   temp.PodSpec,
		EventType: temp.EventType,
	}

	return nil
}
