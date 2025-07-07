package scenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	kubescheduler "k8s.io/kube-scheduler/config/v1"
)

// SchedulerClient interface for scheduler configuration management.
type SchedulerClient interface {
	GetConfig(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error)
	UpdateConfig(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error
}

// HTTPSchedulerClient implements SchedulerClient using HTTP.
type HTTPSchedulerClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPSchedulerClient creates a new HTTP-based scheduler client.
func NewHTTPSchedulerClient(baseURL string, timeout time.Duration) *HTTPSchedulerClient {
	return &HTTPSchedulerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetConfig retrieves the current scheduler configuration.
func (c *HTTPSchedulerClient) GetConfig(ctx context.Context) (*kubescheduler.KubeSchedulerConfiguration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
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

// UpdateConfig updates the scheduler configuration.
func (c *HTTPSchedulerClient) UpdateConfig(ctx context.Context, config *kubescheduler.KubeSchedulerConfiguration) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return nil
}
