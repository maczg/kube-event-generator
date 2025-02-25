package pkg

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type ScheduleEvent struct {
	Event
}

func NewScheduleEvent(p *corev1.Pod, at time.Time, duration time.Duration) *ScheduleEvent {
	event := NewEvent(&at, &duration)
	event.PodSpec = p
	return &ScheduleEvent{
		Event: *event,
	}
}

func (e *ScheduleEvent) Handle(m *Manager) {
	p, err := m.kubeClient.CoreV1().Pods(e.PodSpec.Namespace).Create(
		context.TODO(),
		e.PodSpec,
		metav1.CreateOptions{},
	)
	if err != nil {
		logrus.Errorf("Error creating pod %s: %v", e.PodSpec.Name, err)
		return
	}
	e.PodSpec = p
	// Watch for the pod to become running, then schedule its eviction
	go e.watchForRunningState(m)
}

func (e *ScheduleEvent) OfType() string {
	return "Scheduling"
}

func (e *ScheduleEvent) watchForRunningState(m *Manager) {
	seen := false
	// watch for running state
	fieldSelector := fmt.Sprintf("metadata.name=%s", e.PodSpec.Name)
	w, err := m.kubeClient.CoreV1().Pods(e.PodSpec.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		logrus.Errorf("Error watching pod %s: %v", e.PodSpec.Name, err)
		return
	}
	defer w.Stop()

	for event := range w.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			logrus.Errorf("Unexpected type in watch: %T", event.Object)
			continue
		}

		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil && !seen {
			logrus.Infof("Pod %s is running at %s", p.Name, time.Now().Format("15:04:05"))
			if e.duration != nil {
				logrus.Infof("Sending eviction event for pod %s to syncCh at %s", p.Name, time.Now().Format("15:04:05"))
				ev := NewEvictEvent(p, time.Now().Add(*e.duration))
				m.syncCh <- ev
			}
			seen = true
			return
		}
	}
}
