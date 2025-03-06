package simulation

import (
	"context"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
)

type EvictPodEvent struct {
	event
}

func (e *EvictPodEvent) Run(ctx context.Context) error {
	c := e.mgr.Clientset()
	err := e.mgr.DeletePod(ctx, *c, &e.Pod)
	if err != nil {
		return err
	}
	return nil
}

func NewEvictionEvent(e scenario.Event, mgr *utils.KubernetesManager) *EvictPodEvent {
	return &EvictPodEvent{
		event: event{
			Event: e,
			mgr:   mgr,
		},
	}
}
