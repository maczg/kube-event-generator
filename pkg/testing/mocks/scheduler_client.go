package mocks

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubescheduler "k8s.io/kube-scheduler/config/v1"
)

// MockSchedulerClient is a mock implementation of SchedulerClient.
type MockSchedulerClient struct {
	Config            *kubescheduler.KubeSchedulerConfiguration
	GetConfigFunc     func(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error)
	UpdateConfigFunc  func(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error
	GetConfigCalls    int
	UpdateConfigCalls int
	mu                sync.RWMutex
}

// NewMockSchedulerClient creates a new mock scheduler client.
func NewMockSchedulerClient() *MockSchedulerClient {
	schedulerName := "default-scheduler"

	return &MockSchedulerClient{
		Config: &kubescheduler.KubeSchedulerConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kubescheduler.config.k8s.io/v1",
				Kind:       "KubeSchedulerConfiguration",
			},
			Profiles: []kubescheduler.KubeSchedulerProfile{
				{
					SchedulerName: &schedulerName,
					Plugins: &kubescheduler.Plugins{
						MultiPoint: kubescheduler.PluginSet{
							Enabled: []kubescheduler.Plugin{
								{Name: "NodeResourcesFit", Weight: int32Ptr(1)},
								{Name: "NodeResourcesBalancedAllocation", Weight: int32Ptr(1)},
							},
						},
					},
				},
			},
		},
	}
}

// GetConfig returns the mock configuration.
func (m *MockSchedulerClient) GetConfig(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetConfigCalls++

	if m.GetConfigFunc != nil {
		return m.GetConfigFunc(ctx)
	}

	// Return a copy of the config.
	configCopy := *m.Config

	return &configCopy, nil
}

// UpdateConfig updates the mock configuration.
func (m *MockSchedulerClient) UpdateConfig(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateConfigCalls++

	if m.UpdateConfigFunc != nil {
		return m.UpdateConfigFunc(ctx, config)
	}

	m.Config = config

	return nil
}

// GetCallCounts returns the number of times each method was called.
func (m *MockSchedulerClient) GetCallCounts() (getConfig, updateConfig int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.GetConfigCalls, m.UpdateConfigCalls
}

// int32Ptr returns a pointer to an int32.
func int32Ptr(i int32) *int32 {
	return &i
}
