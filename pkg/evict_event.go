package pkg

import (
	"context"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type EvictEvent struct {
	Event
}

func NewEvictEvent(p *corev1.Pod, at time.Time) *EvictEvent {
	event := NewEvent(&at, nil)
	event.PodSpec = p
	return &EvictEvent{
		Event: *event,
	}
}

func (e *EvictEvent) Handle(m *Manager) {
	logrus.Warnf("Evicting pod %s at %s", e.PodSpec.Name, time.Now().Format("15:04:05"))
	err := m.kubeClient.CoreV1().Pods(e.PodSpec.Namespace).Delete(
		context.TODO(),
		e.PodSpec.Name,
		metav1.DeleteOptions{},
	)
	if err != nil {
		logrus.Errorf("Error evicting pod %s: %v", e.PodSpec.Name, err)
	}
}

func (e *EvictEvent) OfType() string {
	return "Eviction"
}
