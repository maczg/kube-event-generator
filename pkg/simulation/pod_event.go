package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
	eventscheduler "github.com/maczg/kube-event-generator/pkg/scheduler"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
	// Name is the name of the pod event
	Name string `yaml:"name" json:"name"`
	// ArrivalTime is the time when the event arrives in the scheduler
	ArrivalTime EventDuration `yaml:"arrivalTime" json:"arrivalTime"`
	// EvictTime is the time when the pod should be evicted after creation
	EvictTime EventDuration `yaml:"evictTime" json:"evictTime"`
	// Timeout is the maximum time allowed for the event to execute
	Timeout EventDuration `yaml:"timeout" json:"timeout"` // Timeout for event execution
	// PodSpec is the specification of the pod to be created or deleted
	PodSpec *v1.Pod `yaml:"podSpec" json:"podSpec"`
	// EventType indicates the type of pod event (create or delete)
	EventType PodEventType `json:"eventType"`
	// Clientset is the Kubernetes clientset used to interact with the cluster
	clientset *kubernetes.Clientset
}

// NewCreatePodEvent creates a new pod creation event
func NewCreatePodEvent(arrivalTime, evictionTime time.Duration, spec *v1.Pod) *PodEvent {
	return &PodEvent{
		BaseEvent:   eventscheduler.NewBaseEvent(arrivalTime, evictionTime, 30*time.Second),
		Name:        spec.Name,
		ArrivalTime: EventDuration(arrivalTime),
		EvictTime:   EventDuration(evictionTime),
		PodSpec:     spec,
		EventType:   PodEventTypeCreate,
		Timeout:     EventDuration(30 * time.Second),
	}
}

// NewDeletePodEvent creates a new pod deletion event
func NewDeletePodEvent(arrivalTime time.Duration, spec *v1.Pod) *PodEvent {
	return &PodEvent{
		BaseEvent: eventscheduler.NewBaseEvent(arrivalTime, 0, 30*time.Second),
		Name:      spec.Name,
		PodSpec:   spec,
		EventType: PodEventTypeDelete,
		Timeout:   EventDuration(30 * time.Second),
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

	if e.clientset == nil {
		return errors.New("clientset is nil")
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

// createAndWatch creates a pod and, when the pod has an eviction time, waits
// for it to run and schedules its eviction.
func (e *PodEvent) createAndWatch(ctx context.Context) error {
	clientset := e.clientset

	createCtx, cancel := context.WithTimeout(ctx, e.GetExecuteTimeout())
	defer cancel()

	created, err := clientset.CoreV1().Pods(e.PodSpec.Namespace).Create(createCtx, e.PodSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	logger.Default().Infof("event %s with pod %s created successfully", e.GetID(), e.PodSpec.Name)

	// If no eviction time is set, we're done
	if e.EvictTime <= 0 {
		return nil
	}

	// Watch for pod to become running, then schedule eviction
	return e.watchAndScheduleEviction(ctx, clientset, created.ResourceVersion)
}

func (e *PodEvent) SetClientset(clientset *kubernetes.Clientset) {
	e.clientset = clientset
}

// watchAndScheduleEviction waits until the pod is running and schedules its
// eviction EvictTime later, so that the run time starts at placement and the
// time spent pending is not counted.
//
// The watch starts from the resource version returned by the create call:
// a pod on a simulated node can be bound and running a few milliseconds after
// creation, and a watch that starts later would miss the transition. The wait
// has no deadline of its own, since a pod may stay pending for a long time in
// a saturated cluster; it ends when the pod runs, is deleted, or ctx is done.
func (e *PodEvent) watchAndScheduleEviction(ctx context.Context, clientset *kubernetes.Clientset, resourceVersion string) error {
	pods := clientset.CoreV1().Pods(e.PodSpec.Namespace)

	for {
		watcher, err := pods.Watch(ctx, metav1.ListOptions{
			FieldSelector:   "metadata.name=" + e.PodSpec.Name,
			ResourceVersion: resourceVersion,
		})
		if err != nil {
			return fmt.Errorf("failed to watch pod %s: %w", e.PodSpec.Name, err)
		}

		running, err := waitForRunning(ctx, watcher, e.PodSpec.Name)
		watcher.Stop()

		if err != nil {
			return err
		}

		if running {
			return e.scheduleEviction(ctx)
		}

		// The watch ended without a verdict (closed by the server or expired
		// resource version): read the current state and resume from there.
		pod, err := pods.Get(ctx, e.PodSpec.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod %s: %w", e.PodSpec.Name, err)
		}

		if pod.Status.Phase == v1.PodRunning {
			return e.scheduleEviction(ctx)
		}

		resourceVersion = pod.ResourceVersion
	}
}

// waitForRunning consumes watch events until the pod is running (true), the
// watch ends without a verdict (false), or the pod is deleted or ctx is done
// (error).
func waitForRunning(ctx context.Context, watcher watch.Interface, name string) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("context done while waiting for pod %s to start: %w", name, ctx.Err())
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return false, nil
			}

			switch event.Type {
			case watch.Error:
				return false, nil
			case watch.Deleted:
				return false, fmt.Errorf("pod %s deleted before running", name)
			}

			if pod, ok := event.Object.(*v1.Pod); ok && pod.Status.Phase == v1.PodRunning {
				logger.Default().Infof("pod %s is running on %s, scheduling eviction", pod.Name, pod.Spec.NodeName)
				return true, nil
			}
		}
	}
}

// scheduleEviction schedules the deletion of the running pod EvictTime from now.
func (e *PodEvent) scheduleEviction(ctx context.Context) error {
	scheduler, ok := ctx.Value(eventscheduler.SchedulerContextKey).(eventscheduler.Scheduler)
	if !ok {
		return errors.New("scheduler not found in context")
	}

	evictionTime := time.Since(scheduler.StartedAt()) + e.EvictTime.Duration()
	evictEvent := NewDeletePodEvent(evictionTime, e.PodSpec)
	evictEvent.SetClientset(e.clientset)

	if err := scheduler.Schedule(evictEvent); err != nil {
		return err
	}

	logger.Default().Debugf("scheduled eviction for pod %s at %s", e.PodSpec.Name, evictionTime)
	return nil
}

// Evict deletes the pod from the cluster
func (e *PodEvent) Evict(ctx context.Context) error {
	clientset := e.clientset

	deleteCtx, cancel := context.WithTimeout(ctx, e.GetExecuteTimeout())
	defer cancel()

	if err := clientset.CoreV1().Pods(e.PodSpec.Namespace).Delete(deleteCtx, e.PodSpec.Name, metav1.DeleteOptions{}); err != nil {
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

	if temp.EventType == "" {
		temp.EventType = PodEventTypeCreate
	}

	e.Name = temp.Name
	e.ArrivalTime = temp.ArrivalTime
	e.EvictTime = temp.EvictTime
	e.PodSpec = temp.PodSpec
	e.EventType = temp.EventType
	e.BaseEvent = eventscheduler.NewBaseEvent(temp.ArrivalTime.Duration(), temp.EvictTime.Duration(), 30*time.Second)

	return nil
}
