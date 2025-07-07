package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubescheduler "k8s.io/kube-scheduler/config/v1"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/testing/mocks"
)

func TestManager_GetPluginWeights(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	weights, err := manager.GetPluginWeights(ctx)
	require.NoError(t, err)

	// Should return default weights from mock.
	assert.Len(t, weights, 2)
	assert.Equal(t, int32(1), weights["NodeResourcesFit"])
	assert.Equal(t, int32(1), weights["NodeResourcesBalancedAllocation"])
}

func TestManager_UpdatePluginWeight_Single(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Update single plugin weight.
	err := manager.UpdatePluginWeight(ctx, "NodeResourcesFit", 10)
	require.NoError(t, err)

	// Verify weight was updated.
	config := mockClient.Config
	require.NotNil(t, config)
	require.Len(t, config.Profiles, 1)

	var found bool

	for _, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
		if plugin.Name == "NodeResourcesFit" {
			require.NotNil(t, plugin.Weight)
			assert.Equal(t, int32(10), *plugin.Weight)

			found = true

			break
		}
	}

	assert.True(t, found, "plugin not found in configuration")
}

func TestManager_UpdatePluginWeight_InvalidWeight(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Try to update with negative weight.
	err := manager.UpdatePluginWeight(ctx, "NodeResourcesFit", -5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weight must be non-negative")
}

func TestManager_UpdatePluginWeight_EmptyName(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Try to update with empty plugin name.
	err := manager.UpdatePluginWeight(ctx, "", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin name cannot be empty")
}

func TestManager_UpdatePluginWeights_Multiple(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Update multiple plugin weights.
	weights := map[string]int32{
		"NodeResourcesFit":                10,
		"NodeResourcesBalancedAllocation": 20,
	}

	err := manager.UpdatePluginWeights(ctx, weights)
	require.NoError(t, err)

	// Verify weights were updated.
	config := mockClient.Config
	require.NotNil(t, config)
	require.Len(t, config.Profiles, 1)

	for _, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
		expectedWeight, exists := weights[plugin.Name]
		if exists {
			require.NotNil(t, plugin.Weight)
			assert.Equal(t, expectedWeight, *plugin.Weight)
		}
	}
}

func TestManager_UpdatePluginWeights_NoMatchingPlugins(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Update non-existent plugins.
	weights := map[string]int32{
		"NonExistentPlugin1": 10,
		"NonExistentPlugin2": 20,
	}

	// Should succeed but not update anything.
	err := manager.UpdatePluginWeights(ctx, weights)
	require.NoError(t, err)

	// Verify no weights were changed.
	config := mockClient.Config
	require.NotNil(t, config)
	require.Len(t, config.Profiles, 1)

	for _, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
		require.NotNil(t, plugin.Weight)
		assert.Equal(t, int32(1), *plugin.Weight)
	}
}

func TestManager_GetConfiguration(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	config, err := manager.GetConfiguration(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify it's the expected configuration.
	assert.Equal(t, "kubescheduler.config.k8s.io/v1", config.APIVersion)
	assert.Equal(t, "KubeSchedulerConfiguration", config.Kind)
}

func TestManager_GetPluginWeights_EmptyProfile(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	// Set up empty profile.
	schedulerName := "empty-scheduler"
	mockClient.Config = &kubescheduler.KubeSchedulerConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubescheduler.config.k8s.io/v1",
			Kind:       "KubeSchedulerConfiguration",
		},
		Profiles: []kubescheduler.KubeSchedulerProfile{
			{
				SchedulerName: &schedulerName,
				Plugins:       nil,
			},
		},
	}

	manager := NewManager(mockClient, logger.Default())

	weights, err := manager.GetPluginWeights(ctx)
	require.NoError(t, err)
	assert.Empty(t, weights)
}

func TestManager_ValidatePluginName(t *testing.T) {
	mockClient := mocks.NewMockSchedulerClient()
	manager := NewManager(mockClient, logger.Default())

	// Test known plugin.
	err := manager.ValidatePluginName("NodeResourcesFit")
	assert.NoError(t, err)

	// Test empty plugin name.
	err = manager.ValidatePluginName("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin name cannot be empty")

	// Test unknown plugin (should not error, just warn).
	err = manager.ValidatePluginName("CustomPlugin")
	assert.NoError(t, err)
}

func TestManager_UpdatePluginWeights_ClientError(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	// Configure mock to return error.
	mockClient.UpdateConfigFunc = func(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error {
		return assert.AnError
	}

	manager := NewManager(mockClient, logger.Default())

	err := manager.UpdatePluginWeight(ctx, "NodeResourcesFit", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update-config")
}
