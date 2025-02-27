package scenario

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/factory"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EventType enumerates the possible event types you might handle.
type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeEvict  EventType = "evict"
)

type PodResource struct {
	CPU    string `json:"cpu" yaml:"cpu"`
	Memory string `json:"memory" yaml:"memory"`
}
type PodSpec struct {
	// The pod spec you want to apply.
	Name      string      `json:"name" yaml:"name"`
	Namespace string      `json:"namespace" yaml:"namespace"`
	Image     string      `json:"image" yaml:"image"`
	Resources PodResource `json:"resources" yaml:"resources"`
}

// Event represents a single step in your scenario.
type Event struct {
	Type    EventType   `json:"type" yaml:"type"`
	PodSpec PodSpec     `json:"podSpec" yaml:"podSpec"`
	Pod     *corev1.Pod `json:"pod" yaml:"pod,omitempty"`
	// Optional: a time to wait before or after this event.
	RunAfter time.Duration `json:"delayAfter" yaml:"delayAfter"`
	Duration time.Duration `json:"duration" yaml:"duration"`
}

func (e *Event) GetPodFromSpec() *corev1.Pod {
	p := factory.NewPod(
		factory.WithMetadata(e.PodSpec.Name, e.PodSpec.Namespace),
		factory.WithContainer("server", e.PodSpec.Image, e.PodSpec.Resources.CPU, e.PodSpec.Resources.Memory),
	)
	e.Pod = p
	return p
}

func (e *Event) Execute(s *Scheduler) error {
	switch e.Type {
	case EventTypeCreate:
		return e.CreatePod(s)
	case EventTypeEvict:
		return e.EvictPod(s)
	}
	return nil
}

func (e *Event) CreatePod(s *Scheduler) error {
	logrus.Infof("creating pod %s", e.Pod.Name)
	p, err := s.KubeClient.CoreV1().Pods(e.Pod.Namespace).Create(
		context.TODO(),
		e.Pod,
		metav1.CreateOptions{},
	)
	if err != nil {
		return err
	}
	e.Pod = p
	go e.waitForPodRunning(s)
	return nil
}

func (e *Event) EvictPod(s *Scheduler) error {
	logrus.Warnf("evict pod %s with duration %s", e.Pod.Name, time.Now().Sub(e.Pod.CreationTimestamp.Time))
	err := s.KubeClient.CoreV1().Pods(e.Pod.Namespace).Delete(
		context.TODO(),
		e.Pod.Name,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}
	return nil
}

func (e *Event) waitForPodRunning(s *Scheduler) {
	logrus.Infof("waiting for pod %s to be running", e.Pod.Name)
	running := false

	fieldSelector := fmt.Sprintf("metadata.name=%s", e.Pod.Name)
	w, err := s.KubeClient.CoreV1().Pods(e.Pod.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	defer w.Stop()

	if err != nil {
		logrus.Errorf("Error watching pod %s: %v", e.Pod.Name, err)
		return
	}

	for event := range w.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			logrus.Errorf("Unexpected type in watch: %T", event.Object)
			continue
		}

		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			err = s.MetricCollector.AddRecord("pod_pending_durations",
				time.Now().Sub(p.CreationTimestamp.Time).Seconds(), nil)
			if err != nil {
				logrus.Errorf("Error adding record: %v", err)
			}

			if running {
				return
			}
			//s.Metrics.RecordPodPendingEnd(e.Pod.Name, time.Now())
			running = true
			logrus.Warnf("Pod %s is running at %s", p.Name, time.Now().Format("15:04:05"))

			runAfter := time.Since(s.startTime) + e.Duration

			evictEvent := &Event{
				Type:     EventTypeEvict,
				PodSpec:  e.PodSpec,
				Pod:      e.Pod,
				RunAfter: runAfter,
			}
			s.AddEvent(evictEvent)

			return
		}
	}
}
