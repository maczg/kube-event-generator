package scheduler

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type EvictPodEvent struct {
	PodEvent
	parent string
}

func NewEvictPodEvent(e *CreatePodEvent, at time.Duration) Event {
	ev := &EvictPodEvent{
		PodEvent: PodEvent{
			ID:  uuid.New().String(),
			pod: e.pod,
			at:  &at,
		},
		parent: e.ID,
	}
	return ev
}

func (ev *EvictPodEvent) Execute(ctx context.Context) error {
	if scheduler, ok := ctx.Value(SchedulerCtxKey).(*Scheduler); ok {
		err := scheduler.KubeManager.DeletePod(ev.pod.Namespace, ev.pod.Name)
		if err != nil {
			logrus.Errorf("error evicting pod %s: %v", ev.pod.Name, err)
		}
		logrus.Infof("pod %s evicted", ev.pod.Name)
	}
	return nil
}
