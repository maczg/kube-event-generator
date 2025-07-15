package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"io"
	kubescheduler "k8s.io/kube-scheduler/config/v1"
	"net/http"
)

// Default scheduler plugins as per Kubernetes default configuration
const (
	SchedulingGates                 = "SchedulingGates"
	PrioritySort                    = "PrioritySort"
	NodeUnschedulable               = "NodeUnschedulable"
	NodeName                        = "NodeName"
	TaintToleration                 = "TaintToleration"
	NodeAffinity                    = "NodeAffinity"
	NodePorts                       = "NodePorts"
	NodeResourcesFit                = "NodeResourcesFit"
	VolumeRestrictions              = "VolumeRestrictions"
	EBSLimits                       = "EBSLimits"
	GCEPDLimits                     = "GCEPDLimits"
	NodeVolumeLimits                = "NodeVolumeLimits"
	AzureDiskLimits                 = "AzureDiskLimits"
	VolumeBinding                   = "VolumeBinding"
	VolumeZone                      = "VolumeZone"
	PodTopologySpread               = "PodTopologySpread"
	InterPodAffinity                = "InterPodAffinity"
	DefaultPreemption               = "DefaultPreemption"
	NodeResourcesBalancedAllocation = "NodeResourcesBalancedAllocation"
	ImageLocality                   = "ImageLocality"
	DefaultBinder                   = "DefaultBinder"
)

// SchedulerManager defines the interface for managing scheduler plugins.
type SchedulerManager interface {
	// GetPluginWeights returns current weights for all plugins.
	GetPluginWeights(ctx context.Context) (map[string]int32, error)

	// UpdatePluginWeight updates the weight of a specific plugin.
	UpdatePluginWeight(ctx context.Context, pluginName string, weight int32) error

	// UpdatePluginWeights updates multiple plugin weights atomically.
	UpdatePluginWeights(ctx context.Context, weights map[string]int32) error

	// GetConfiguration returns the current scheduler configuration.
	GetConfiguration(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error)

	// UpdateConfiguration updates the scheduler configuration.
	UpdateConfiguration(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error

	// ValidatePluginName checks if a plugin name is valid.
	ValidatePluginName(pluginName string) error
}

type HTTPKubeSchedulerManager struct {
	httpClient   *http.Client
	baseURL      string
	knownPlugins map[string]bool
}

// NewHTTPKubeSchedulerManager creates a new HTTPKubeSchedulerManager instance.
func NewHTTPKubeSchedulerManager(baseURL string) *HTTPKubeSchedulerManager {
	return &HTTPKubeSchedulerManager{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		knownPlugins: map[string]bool{
			SchedulingGates:                 true,
			PrioritySort:                    true,
			NodeUnschedulable:               true,
			NodeName:                        true,
			TaintToleration:                 true,
			NodeAffinity:                    true,
			NodePorts:                       true,
			NodeResourcesFit:                true,
			VolumeRestrictions:              true,
			EBSLimits:                       true,
			GCEPDLimits:                     true,
			NodeVolumeLimits:                true,
			AzureDiskLimits:                 true,
			VolumeBinding:                   true,
			VolumeZone:                      true,
			PodTopologySpread:               true,
			InterPodAffinity:                true,
			DefaultPreemption:               true,
			NodeResourcesBalancedAllocation: true,
			ImageLocality:                   true,
			DefaultBinder:                   true,
		},
	}
}

func (h *HTTPKubeSchedulerManager) GetPluginWeights(ctx context.Context) (map[string]int32, error) {
	config, err := h.GetConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler configuration: %w", err)
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

func (h *HTTPKubeSchedulerManager) UpdatePluginWeight(ctx context.Context, pluginName string, weight int32) error {
	return h.UpdatePluginWeights(ctx, map[string]int32{pluginName: weight})
}

func (h *HTTPKubeSchedulerManager) UpdatePluginWeights(ctx context.Context, weights map[string]int32) error {
	config, err := h.GetConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scheduler configuration: %w", err)
	}

	// Validate all plugins and weights first
	for pluginName, weight := range weights {
		if err := h.ValidatePluginName(pluginName); err != nil {
			return fmt.Errorf("invalid plugin name %s: %w", pluginName, err)
		}
		if weight < 1 {
			return fmt.Errorf("weight for plugin %s must be greater than 1", pluginName)
		}
	}

	// Update all plugin weights in the configuration
	updated := false
	for pluginName, weight := range weights {
		pluginFound := false
		for i, profile := range config.Profiles {
			if profile.Plugins == nil || profile.Plugins.MultiPoint.Enabled == nil {
				continue
			}

			for j, plugin := range profile.Plugins.MultiPoint.Enabled {
				if plugin.Name == pluginName {
					if plugin.Weight == nil {
						plugin.Weight = new(int32)
					}
					*plugin.Weight = weight
					profile.Plugins.MultiPoint.Enabled[j] = plugin
					config.Profiles[i] = profile
					pluginFound = true
					updated = true
					break
				}
			}
			if pluginFound {
				break
			}
		}
		if !pluginFound {
			return fmt.Errorf("plugin %s not found in scheduler configuration", pluginName)
		}
	}

	if updated {
		return h.UpdateConfiguration(ctx, config)
	}
	return nil
}

func (h *HTTPKubeSchedulerManager) GetConfiguration(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var config kubescheduler.KubeSchedulerConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &config, nil
}

func (h *HTTPKubeSchedulerManager) UpdateConfiguration(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error {
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL, bytes.NewBuffer(configData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return nil
}

func (h *HTTPKubeSchedulerManager) ValidatePluginName(pluginName string) error {
	if pluginName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if !h.knownPlugins[pluginName] {
		logger.Default().Warnf("unknown plugin name %s", pluginName)
	}

	return nil
}

// AddKnownPlugin adds a plugin to the known plugins list.
func (h *HTTPKubeSchedulerManager) AddKnownPlugin(pluginName string) {
	h.knownPlugins[pluginName] = true
}

func (h *HTTPKubeSchedulerManager) RemoveKnownPlugin(pluginName string) {
	delete(h.knownPlugins, pluginName)
}

// ResetToDefaults resets all plugin weights to default values as per Kubernetes default configuration.
func (h *HTTPKubeSchedulerManager) ResetToDefaults(ctx context.Context) error {
	weights := map[string]int32{
		// Plugins with specific weights in default config
		TaintToleration:                 3,
		NodeAffinity:                    2,
		NodeResourcesFit:                1,
		PodTopologySpread:               2,
		InterPodAffinity:                2,
		NodeResourcesBalancedAllocation: 1,
		ImageLocality:                   1,
		// All other plugins default to weight 1 (implicit in scheduler config)
		SchedulingGates:    1,
		PrioritySort:       1,
		NodeUnschedulable:  1,
		NodeName:           1,
		NodePorts:          1,
		VolumeRestrictions: 1,
		EBSLimits:          1,
		GCEPDLimits:        1,
		NodeVolumeLimits:   1,
		AzureDiskLimits:    1,
		VolumeBinding:      1,
		VolumeZone:         1,
		DefaultPreemption:  1,
		DefaultBinder:      1,
	}
	return h.UpdatePluginWeights(ctx, weights)
}
