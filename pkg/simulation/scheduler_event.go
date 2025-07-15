package simulation

import (
	"context"
	"errors"
	kube "github.com/maczg/kube-event-generator/pkg/kubernetes"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"time"
)

// KubeSchedulerEvent represents a scheduler configuration change event
type KubeSchedulerEvent struct {
	*scheduler.BaseEvent
	Weights map[string]int32 `yaml:"weights" json:"weights"`
	manager kube.SchedulerManager
}

// NewSchedulerEvent creates a new KubeSchedulerEvent
func NewSchedulerEvent(arrivalTime time.Duration, weights map[string]int32, manager kube.SchedulerManager) *KubeSchedulerEvent {
	return &KubeSchedulerEvent{
		BaseEvent: scheduler.NewBaseEvent(arrivalTime, 30),
		Weights:   weights,
		manager:   manager,
	}
}

// Execute implements the scheduler-specific execution logic
func (e *KubeSchedulerEvent) Execute(ctx context.Context) error {
	e.SetStatus(scheduler.EventStatusExecuting)
	defer func() {
		if e.GetStatus() == scheduler.EventStatusExecuting {
			e.SetStatus(scheduler.EventStatusCompleted)
		}
	}()
	if e.manager == nil {
		return errors.New("scheduler manager is nil")
	}
	err := e.manager.UpdatePluginWeights(ctx, e.Weights)
	if err != nil {
		return err
	}
	logger.Default().Infoln("scheduler event executed successfully")
	return nil
}
