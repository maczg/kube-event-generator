package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents health check status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Component represents a system component's health.
type Component struct {
	LastCheck time.Time `json:"last_check"`
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	Message   string    `json:"message,omitempty"`
}

// Check represents a health check function.
type Check func(ctx context.Context) *Component

// Checker manages health checks.
type Checker struct {
	checks     map[string]Check
	components map[string]*Component
	mu         sync.RWMutex
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checks:     make(map[string]Check),
		components: make(map[string]*Component),
	}
}

// RegisterCheck registers a health check.
func (c *Checker) RegisterCheck(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks[name] = check
}

// RunChecks executes all registered health checks.
func (c *Checker) RunChecks(ctx context.Context) {
	c.mu.Lock()

	checks := make(map[string]Check)
	for k, v := range c.checks {
		checks[k] = v
	}
	c.mu.Unlock()

	results := make(map[string]*Component)

	var wg sync.WaitGroup

	var resultMu sync.Mutex

	for name, check := range checks {
		wg.Add(1)

		go func(n string, ch Check) {
			defer wg.Done()

			// Run check with timeout.
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			component := ch(checkCtx)
			component.Name = n
			component.LastCheck = time.Now()

			resultMu.Lock()
			results[n] = component
			resultMu.Unlock()
		}(name, check)
	}

	wg.Wait()

	c.mu.Lock()
	c.components = results
	c.mu.Unlock()
}

// GetStatus returns the overall health status.
func (c *Checker) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.components) == 0 {
		return StatusUnhealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, component := range c.components {
		switch component.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}

	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// GetComponents returns all component health statuses.
func (c *Checker) GetComponents() map[string]*Component {
	c.mu.RLock()
	defer c.mu.RUnlock()

	components := make(map[string]*Component)
	for k, v := range c.components {
		components[k] = v
	}

	return components
}

// Handler returns an HTTP handler for health checks.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c.RunChecks(ctx)

		response := struct {
			Timestamp  time.Time             `json:"timestamp"`
			Components map[string]*Component `json:"components"`
			Status     Status                `json:"status"`
		}{
			Status:     c.GetStatus(),
			Components: c.GetComponents(),
			Timestamp:  time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")

		if response.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// Common health checks.

// KubernetesCheck creates a health check for Kubernetes connectivity.
func KubernetesCheck(checkFunc func(context.Context) error) Check {
	return func(ctx context.Context) *Component {
		err := checkFunc(ctx)
		if err != nil {
			return &Component{
				Status:  StatusUnhealthy,
				Message: err.Error(),
			}
		}

		return &Component{
			Status: StatusHealthy,
		}
	}
}

// SchedulerCheck creates a health check for the scheduler.
func SchedulerCheck(isRunning func() bool) Check {
	return func(ctx context.Context) *Component {
		if isRunning() {
			return &Component{
				Status: StatusHealthy,
			}
		}

		return &Component{
			Status:  StatusDegraded,
			Message: "scheduler not running",
		}
	}
}
