package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	kube "github.com/maczg/kube-event-generator/pkg/kubeutils"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"time"
)

// KubeSchedulerEvent represents a scheduler configuration change event:
// at ArrivalTime the score plugin weights are set to Weights.
type KubeSchedulerEvent struct {
	*scheduler.BaseEvent
	// Name of the scheduler event
	Name string `yaml:"name" json:"name"`
	// ArrivalTime is the time, from the start of the simulation, at which the weights are applied
	ArrivalTime EventDuration `yaml:"arrivalTime" json:"arrivalTime"`
	// Weights maps a score plugin name to its new weight
	Weights map[string]int32 `yaml:"weights" json:"weights"`
	manager kube.SchedulerManager
}

// NewSchedulerEvent creates a new KubeSchedulerEvent
func NewSchedulerEvent(name string, arrivalTime time.Duration, weights map[string]int32, manager kube.SchedulerManager) *KubeSchedulerEvent {
	return &KubeSchedulerEvent{
		Name:        name,
		BaseEvent:   scheduler.NewBaseEvent(arrivalTime, 0, 30*time.Second),
		ArrivalTime: EventDuration(arrivalTime),
		Weights:     weights,
		manager:     manager,
	}
}

// UnmarshalJSON reads a scheduler event from a scenario file, where
// arrivalTime is a duration string such as "30s".
func (e *KubeSchedulerEvent) UnmarshalJSON(data []byte) error {
	var temp struct {
		Name        string           `json:"name"`
		ArrivalTime EventDuration    `json:"arrivalTime"`
		Weights     map[string]int32 `json:"weights"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if len(temp.Weights) == 0 {
		return fmt.Errorf("scheduler event %q has no weights", temp.Name)
	}

	e.Name = temp.Name
	e.ArrivalTime = temp.ArrivalTime
	e.Weights = temp.Weights
	e.BaseEvent = scheduler.NewBaseEvent(temp.ArrivalTime.Duration(), 0, 30*time.Second)

	return nil
}

func (e *KubeSchedulerEvent) SetManager(manager kube.SchedulerManager) {
	e.manager = manager
}

func (e *KubeSchedulerEvent) GetName() string {
	if e.Name != "" {
		return e.Name
	} else {
		return fmt.Sprintf("scheduler-event-%s", e.GetID())
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

	ctx, cancel := context.WithTimeout(ctx, e.GetExecuteTimeout())
	defer cancel()

	if err := e.manager.UpdatePluginWeights(ctx, e.Weights); err != nil {
		return err
	}
	logger.Default().Infof("scheduler event %s applied weights %v", e.GetName(), e.Weights)
	return nil
}
