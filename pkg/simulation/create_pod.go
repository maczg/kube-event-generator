package simulation

import (
	"context"
	"errors"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type CreatePodEvent struct {
	event
}

func NewSimEvent(e scenario.Event, mgr *utils.KubernetesManager, scheduler scheduler.Scheduler) *CreatePodEvent {
	return &CreatePodEvent{
		event: event{
			Event:     e,
			mgr:       mgr,
			scheduler: scheduler,
		},
	}
}

func (s *CreatePodEvent) Run(ctx context.Context) error {
	c := s.mgr.Clientset()
	p, err := s.mgr.CreatePod(ctx, *c, &s.Pod)
	if err != nil {
		return err
	}
	if s.For() > 0 {
		w, err := c.CoreV1().Pods(p.Namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + p.Name,
		})
		if err != nil {
			return err
		}
		for {
			select {
			case <-ctx.Done():
				return errors.New("cannot wait pod finish, context cancelled")
			case e := <-w.ResultChan():
				if pod, ok := e.Object.(*corev1.Pod); ok {
					if pod.Status.Phase == corev1.PodRunning {
						newEvEvent := NewEvictionEvent(s.Event, s.mgr)
						newEvEvent.Pod = *pod
						at := time.Since(s.scheduler.StartedAt()) + s.For()
						newEvEvent.From = at
						newEvEvent.Duration = 0
						newEvEvent.Name = "eviction-" + s.Name
						s.scheduler.Schedule(newEvEvent)
						return nil
					}
				}
			}
		}
	}
	return nil
}
