package scheduler

import (
	"context"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type CreatePodEvent struct {
	PodEvent
}

func NewCreatePodEvent(pod *corev1.Pod, at time.Duration, for_ time.Duration) Event {
	c := &CreatePodEvent{
		PodEvent: NewPodEvent(*pod, at, for_),
	}
	return c
}

func (c *CreatePodEvent) Execute(ctx context.Context) error {
	if scheduler, ok := ctx.Value(SchedulerCtxKey).(*Scheduler); ok {
		err := scheduler.KubeManager.CreatePod(c.pod)
		if err != nil {
			logrus.Errorf("error creating pod %s: %v", c.pod.Name, err)
		}
		logrus.Infof("pod %s created", c.pod.Name)
	}
	go c.waitForPodCreation(ctx)
	return nil
}

func (c *CreatePodEvent) waitForPodCreation(ctx context.Context) {
	if scheduler, ok := ctx.Value(SchedulerCtxKey).(*Scheduler); ok {
		err := scheduler.KubeManager.WaitForPodReady(c.pod.Namespace, c.pod.Name)
		if err != nil {
			logrus.Errorf("error waiting for pod %s to be ready: %v", c.pod.Name, err)
			return
		}
		logrus.Infof("pod %s is ready", c.pod.Name)
		if c.IsEvictable() {
			at := time.Since(scheduler.StartTime()) + c.Duration()
			ev := NewEvictPodEvent(c, at)
			scheduler.queue.Push(ev)
		}
	}
}
