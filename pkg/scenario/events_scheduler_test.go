package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kubescheduler "k8s.io/kube-scheduler/config/v1"

	"github.com/maczg/kube-event-generator/pkg/testing/mocks"
)

func TestSchedulerEvent_Basic(t *testing.T) {
	mockClient := mocks.NewMockSchedulerClient()

	event := NewSchedulerEvent(
		"test-event",
		100*time.Millisecond,
		map[string]int32{
			"NodeResourcesFit":                20,
			"NodeResourcesBalancedAllocation": 5,
		},
		mockClient,
	)

	// Test basic properties
	assert.Equal(t, "test-event", event.ID())
	assert.Equal(t, 100*time.Millisecond, event.ExecuteAfterDuration())
	assert.Equal(t, time.Duration(0), event.ExecuteForDuration())
}

func TestSchedulerEvent_Execute_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	event := NewSchedulerEvent(
		"update-weights",
		0,
		map[string]int32{
			"NodeResourcesFit":                10,
			"NodeResourcesBalancedAllocation": 20,
		},
		mockClient,
	)

	// Execute event
	err := event.Execute(ctx)
	require.NoError(t, err)

	// Verify client was called
	getCount, updateCount := mockClient.GetCallCounts()
	assert.Equal(t, 1, getCount)
	assert.Equal(t, 1, updateCount)

	// Verify weights were updated
	config := mockClient.Config
	require.NotNil(t, config)
	require.Len(t, config.Profiles, 1)
	require.NotNil(t, config.Profiles[0].Plugins)
	require.NotNil(t, config.Profiles[0].Plugins.MultiPoint)

	plugins := config.Profiles[0].Plugins.MultiPoint.Enabled
	require.Len(t, plugins, 2)

	// Check updated weights
	for _, plugin := range plugins {
		switch plugin.Name {
		case "NodeResourcesFit":
			require.NotNil(t, plugin.Weight)
			assert.Equal(t, int32(10), *plugin.Weight)
		case "NodeResourcesBalancedAllocation":
			require.NotNil(t, plugin.Weight)
			assert.Equal(t, int32(20), *plugin.Weight)
		default:
			t.Errorf("unexpected plugin: %s", plugin.Name)
		}
	}
}

func TestSchedulerEvent_Execute_NoMatchingPlugins(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	event := NewSchedulerEvent(
		"update-nonexistent",
		0,
		map[string]int32{
			"NonExistentPlugin": 10,
		},
		mockClient,
	)

	// Execute should succeed even if no plugins match
	err := event.Execute(ctx)
	require.NoError(t, err)

	// Verify client was called
	getCount, updateCount := mockClient.GetCallCounts()
	assert.Equal(t, 1, getCount)
	assert.Equal(t, 1, updateCount)
}

func TestSchedulerEvent_Execute_NoClient(t *testing.T) {
	ctx := context.Background()

	event := &SchedulerEvent{
		Name: "no-client",
		Weights: map[string]int32{
			"NodeResourcesFit": 10,
		},
	}

	// Execute should fail when client is nil
	err := event.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler client not configured")
}

func TestSchedulerEvent_Execute_GetConfigError(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	// Configure mock to return error
	mockClient.GetConfigFunc = func(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error) {
		return nil, assert.AnError
	}

	event := NewSchedulerEvent(
		"error-event",
		0,
		map[string]int32{"NodeResourcesFit": 10},
		mockClient,
	)

	// Execute should fail
	err := event.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get-config")
}

func TestSchedulerEvent_Execute_UpdateConfigError(t *testing.T) {
	ctx := context.Background()
	mockClient := mocks.NewMockSchedulerClient()

	// Configure mock to return error on update
	mockClient.UpdateConfigFunc = func(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error {
		return assert.AnError
	}

	event := NewSchedulerEvent(
		"update-error",
		0,
		map[string]int32{"NodeResourcesFit": 10},
		mockClient,
	)

	// Execute should fail
	err := event.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update-config")
}

func TestSchedulerEvent_ComparePriority(t *testing.T) {
	event1 := &SchedulerEvent{
		Name:         "event1",
		ExecuteAfter: EventDuration(100 * time.Millisecond),
	}

	event2 := &SchedulerEvent{
		Name:         "event2",
		ExecuteAfter: EventDuration(200 * time.Millisecond),
	}

	// event1 should have higher priority (executes sooner)
	assert.True(t, event1.ComparePriority(event2))
	assert.False(t, event2.ComparePriority(event1))
}

func TestSchedulerEvent_ApplyWeights_EmptyProfiles(t *testing.T) {
	event := &SchedulerEvent{
		Weights: map[string]int32{"NodeResourcesFit": 10},
	}

	config := &kubescheduler.KubeSchedulerConfiguration{
		Profiles: []kubescheduler.KubeSchedulerProfile{},
	}

	matched := event.applyWeights(config)
	assert.Empty(t, matched)
}

func TestSchedulerEvent_ApplyWeights_NilPlugins(t *testing.T) {
	schedulerName := "test-scheduler"
	event := &SchedulerEvent{
		Weights: map[string]int32{"NodeResourcesFit": 10},
	}

	config := &kubescheduler.KubeSchedulerConfiguration{
		Profiles: []kubescheduler.KubeSchedulerProfile{
			{
				SchedulerName: &schedulerName,
				Plugins:       nil,
			},
		},
	}

	matched := event.applyWeights(config)
	assert.Empty(t, matched)
}

func TestSchedulerEvent_SetClient(t *testing.T) {
	event := &SchedulerEvent{
		Name: "test-event",
	}

	mockClient := mocks.NewMockSchedulerClient()
	event.SetClient(mockClient)

	// Verify client is set
	assert.NotNil(t, event.client)
}

func TestHTTPSchedulerClient_GetConfig(t *testing.T) {
	// This is an integration test example - would need actual HTTP server
	t.Skip("Skipping integration test")
}

func TestHTTPSchedulerClient_UpdateConfig(t *testing.T) {
	// This is an integration test example - would need actual HTTP server
	t.Skip("Skipping integration test")
}
