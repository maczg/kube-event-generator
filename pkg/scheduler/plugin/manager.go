// Package plugin provides functionality to manage scheduler plugins.
package plugin

import (
	"context"
	"fmt"
	"sync"

	kubescheduler "k8s.io/kube-scheduler/config/v1"

	"github.com/maczg/kube-event-generator/pkg/errors"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
)

// manager implements the Manager interface.
type manager struct {
	client       scenario.SchedulerClient
	logger       *logger.Logger
	knownPlugins map[string]bool
	mu           sync.RWMutex
}

// NewManager creates a new plugin manager.
func NewManager(client scenario.SchedulerClient, log *logger.Logger) Manager {
	if log == nil {
		log = logger.Default()
	}

	return &manager{
		client: client,
		logger: log,
		knownPlugins: map[string]bool{
			// Default Kubernetes scheduler plugins.
			"NodeResourcesFit":                true,
			"NodeResourcesBalancedAllocation": true,
			"ImageLocality":                   true,
			"TaintToleration":                 true,
			"NodeAffinity":                    true,
			"NodePorts":                       true,
			"NodeName":                        true,
			"PodTopologySpread":               true,
			"InterPodAffinity":                true,
			"VolumeBinding":                   true,
			"VolumeRestrictions":              true,
			"VolumeZone":                      true,
			"NodeVolumeLimits":                true,
			"EBSLimits":                       true,
			"GCEPDLimits":                     true,
			"AzureDiskLimits":                 true,
			"ServiceAffinity":                 true,
			"DefaultPreemption":               true,
			"PrioritySort":                    true,
			"DefaultBinder":                   true,
		},
	}
}

// GetPluginWeights returns current weights for all plugins.
func (m *manager) GetPluginWeights(ctx context.Context) (map[string]int32, error) {
	config, err := m.client.GetConfig(ctx)
	if err != nil {
		return nil, errors.WrapSchedulerError("get-config", "", err)
	}

	weights := make(map[string]int32)

	for _, profile := range config.Profiles {
		if profile.Plugins == nil || profile.Plugins.MultiPoint.Enabled == nil {
			continue
		}

		for _, plugin := range profile.Plugins.MultiPoint.Enabled {
			if plugin.Weight != nil {
				weights[plugin.Name] = *plugin.Weight
			} else {
				weights[plugin.Name] = 1
			}
		}
	}

	return weights, nil
}

// UpdatePluginWeight updates the weight of a specific plugin.
func (m *manager) UpdatePluginWeight(ctx context.Context, pluginName string, weight int32) error {
	if err := m.ValidatePluginName(pluginName); err != nil {
		return err
	}

	if weight < 0 {
		return errors.NewValidationError("weight", weight, "weight must be non-negative")
	}

	return m.UpdatePluginWeights(ctx, map[string]int32{pluginName: weight})
}

// UpdatePluginWeights updates multiple plugin weights atomically.
func (m *manager) UpdatePluginWeights(ctx context.Context, weights map[string]int32) error {
	// Validate all plugin names and weights first.
	for name, weight := range weights {
		if err := m.ValidatePluginName(name); err != nil {
			return err
		}

		if weight < 0 {
			return errors.NewValidationError("weight", weight, fmt.Sprintf("weight for plugin %s must be non-negative", name))
		}
	}

	// Get current configuration.
	config, err := m.client.GetConfig(ctx)
	if err != nil {
		return errors.WrapSchedulerError("get-config", "", err)
	}

	// Apply updates.
	updatedCount := 0

	for i := range config.Profiles {
		profile := &config.Profiles[i]
		if profile.Plugins == nil || profile.Plugins.MultiPoint.Enabled == nil {
			continue
		}

		for j := range profile.Plugins.MultiPoint.Enabled {
			plugin := &profile.Plugins.MultiPoint.Enabled[j]
			if newWeight, exists := weights[plugin.Name]; exists {
				plugin.Weight = &newWeight
				updatedCount++

				m.logger.WithFields(map[string]interface{}{
					"plugin":  plugin.Name,
					"weight":  newWeight,
					"profile": profile.SchedulerName,
				}).Debug("updated plugin weight")
			}
		}
	}

	if updatedCount == 0 {
		m.logger.Warn("no matching plugins found in scheduler configuration")
	}

	// Update configuration.
	if err := m.client.UpdateConfig(ctx, config); err != nil {
		return errors.WrapSchedulerError("update-config", "", err)
	}

	m.logger.WithFields(map[string]interface{}{
		"plugins_updated": updatedCount,
		"total_weights":   len(weights),
	}).Info("plugin weights updated successfully")

	return nil
}

// GetConfiguration returns the current scheduler configuration.
func (m *manager) GetConfiguration(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error) {
	config, err := m.client.GetConfig(ctx)
	if err != nil {
		return nil, errors.WrapSchedulerError("get-config", "", err)
	}

	return config, nil
}

// ValidatePluginName checks if a plugin name is valid.
func (m *manager) ValidatePluginName(pluginName string) error {
	if pluginName == "" {
		return errors.NewValidationError("pluginName", pluginName, "plugin name cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.knownPlugins[pluginName] {
		// Log warning but don't fail - custom plugins might exist.
		m.logger.WithFields(map[string]interface{}{
			"plugin": pluginName,
		}).Warn("unknown plugin name - proceeding anyway")
	}

	return nil
}

// AddKnownPlugin adds a plugin to the known plugins list.
func (m *manager) AddKnownPlugin(pluginName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.knownPlugins[pluginName] = true
}
