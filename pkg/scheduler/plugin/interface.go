package plugin

import (
	"context"

	kubescheduler "k8s.io/kube-scheduler/config/v1"
)

// Manager defines the interface for managing scheduler plugins.
type Manager interface {
	// GetPluginWeights returns current weights for all plugins.
	GetPluginWeights(ctx context.Context) (map[string]int32, error)

	// UpdatePluginWeight updates the weight of a specific plugin.
	UpdatePluginWeight(ctx context.Context, pluginName string, weight int32) error

	// UpdatePluginWeights updates multiple plugin weights atomically.
	UpdatePluginWeights(ctx context.Context, weights map[string]int32) error

	// GetConfiguration returns the current scheduler configuration.
	GetConfiguration(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error)

	// ValidatePluginName checks if a plugin name is valid.
	ValidatePluginName(pluginName string) error
}

// WeightUpdate represents a plugin weight update request.
type WeightUpdate struct {
	PluginName string
	Weight     int32
	Timestamp  int64
}

// Info contains information about a scheduler plugin.
type Info struct {
	Name        string
	Description string
	Weight      int32
	Enabled     bool
}
