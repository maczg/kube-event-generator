package scenario

import (
	"context"
	"fmt"
	"time"

	kubescheduler "k8s.io/kube-scheduler/config/v1"

	"github.com/maczg/kube-event-generator/pkg/errors"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
)

// SchedulerEvent represents a scheduler configuration change event.
type SchedulerEvent struct {
	client       SchedulerClient
	Weights      map[string]int32 `yaml:"weights" json:"weights"`
	Name         string           `yaml:"name" json:"name"`
	ExecuteAfter EventDuration    `yaml:"after" json:"after"`
}

// NewSchedulerEvent creates a new scheduler event.
func NewSchedulerEvent(name string, after time.Duration, weights map[string]int32, client SchedulerClient) *SchedulerEvent {
	return &SchedulerEvent{
		Name:         name,
		ExecuteAfter: EventDuration(after),
		Weights:      weights,
		client:       client,
	}
}

// SetClient sets the scheduler client.
func (e *SchedulerEvent) SetClient(client SchedulerClient) {
	e.client = client
}

// ID returns the event ID.
func (e *SchedulerEvent) ID() string {
	return e.Name
}

// ExecuteAfterDuration returns the duration to wait before execution.
func (e *SchedulerEvent) ExecuteAfterDuration() time.Duration {
	return time.Duration(e.ExecuteAfter)
}

// ExecuteForDuration returns 0 as scheduler events are instant.
func (e *SchedulerEvent) ExecuteForDuration() time.Duration {
	return 0
}

// ComparePriority compares this event with another for scheduling priority.
func (e *SchedulerEvent) ComparePriority(other scheduler.Schedulable) bool {
	return e.ExecuteAfterDuration() < other.ExecuteAfterDuration()
}

// Execute applies the scheduler configuration changes.
func (e *SchedulerEvent) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"event_type": "scheduler_config",
		"event":      e.Name,
		"weights":    e.Weights,
	})

	log.Info("executing scheduler event")

	if e.client == nil {
		return errors.WrapSchedulerError("execute", e.Name, fmt.Errorf("scheduler client not configured"))
	}

	// Get current configuration.
	currentConfig, err := e.client.GetConfig(ctx)
	if err != nil {
		return errors.WrapSchedulerError("get-config", e.Name, err)
	}

	// Apply weight changes.
	matched := e.applyWeights(currentConfig)
	if len(matched) == 0 {
		log.Warn("no matching plugins found in scheduler config")
	} else {
		log.WithFields(map[string]interface{}{
			"plugins": matched,
		}).Debug("updated plugin weights")
	}

	// Update configuration.
	if err := e.client.UpdateConfig(ctx, currentConfig); err != nil {
		return errors.WrapSchedulerError("update-config", e.Name, err)
	}

	log.Info("scheduler event completed successfully")

	return nil
}

// applyWeights applies the weight changes to the configuration.
func (e *SchedulerEvent) applyWeights(config *kubescheduler.KubeSchedulerConfiguration) []string {
	if len(config.Profiles) == 0 {
		return nil
	}

	matched := []string{}
	profile := &config.Profiles[0]

	if profile.Plugins == nil {
		return nil
	}

	if profile.Plugins.MultiPoint.Enabled == nil {
		return nil
	}

	for name, weight := range e.Weights {
		for i, plugin := range profile.Plugins.MultiPoint.Enabled {
			if plugin.Name == name {
				matched = append(matched, name)
				weightCopy := weight
				profile.Plugins.MultiPoint.Enabled[i].Weight = &weightCopy

				break
			}
		}
	}

	return matched
}
