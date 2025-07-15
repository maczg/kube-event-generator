package scheduler

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds performance and operational metrics for the scheduler
type Metrics struct {
	// Event counters
	EventsScheduled *AtomicCounter
	EventsExecuted  *AtomicCounter
	EventsCompleted *AtomicCounter
	EventsFailed    *AtomicCounter
	EventsCanceled  *AtomicCounter
	EventsEvicted   *AtomicCounter

	// Queue metrics
	QueueSize    *AtomicGauge
	MaxQueueSize *AtomicGauge

	// Performance metrics
	ExecutionDuration *Histogram

	// Timing metrics
	StartTime     time.Time
	LastEventTime time.Time
	lastEventMu   sync.RWMutex
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		EventsScheduled:   NewAtomicCounter(),
		EventsExecuted:    NewAtomicCounter(),
		EventsCompleted:   NewAtomicCounter(),
		EventsFailed:      NewAtomicCounter(),
		EventsCanceled:    NewAtomicCounter(),
		EventsEvicted:     NewAtomicCounter(),
		QueueSize:         NewAtomicGauge(),
		MaxQueueSize:      NewAtomicGauge(),
		ExecutionDuration: NewHistogram(),
		StartTime:         time.Now(),
	}
}

// UpdateLastEventTime updates the timestamp of the last processed event
func (m *Metrics) UpdateLastEventTime() {
	m.lastEventMu.Lock()
	defer m.lastEventMu.Unlock()
	m.LastEventTime = time.Now()
}

// GetLastEventTime returns the timestamp of the last processed event
func (m *Metrics) GetLastEventTime() time.Time {
	m.lastEventMu.RLock()
	defer m.lastEventMu.RUnlock()
	return m.LastEventTime
}

// GetUptime returns how long the scheduler has been running
func (m *Metrics) GetUptime() time.Duration {
	return time.Since(m.StartTime)
}

// GetEventRate returns the average events per second since start
func (m *Metrics) GetEventRate() float64 {
	uptime := m.GetUptime().Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(m.EventsExecuted.Value()) / uptime
}

// GetSuccessRate returns the success rate as a percentage
func (m *Metrics) GetSuccessRate() float64 {
	executed := m.EventsExecuted.Value()
	if executed == 0 {
		return 0
	}
	completed := m.EventsCompleted.Value()
	return (float64(completed) / float64(executed)) * 100
}

// Summary returns a summary of all metrics
func (m *Metrics) Summary() map[string]interface{} {
	return map[string]interface{}{
		"events_scheduled":     m.EventsScheduled.Value(),
		"events_executed":      m.EventsExecuted.Value(),
		"events_completed":     m.EventsCompleted.Value(),
		"events_failed":        m.EventsFailed.Value(),
		"events_canceled":      m.EventsCanceled.Value(),
		"events_evicted":       m.EventsEvicted.Value(),
		"queue_size":           m.QueueSize.Value(),
		"max_queue_size":       m.MaxQueueSize.Value(),
		"uptime_seconds":       m.GetUptime().Seconds(),
		"events_per_second":    m.GetEventRate(),
		"success_rate_percent": m.GetSuccessRate(),
		"last_event_time":      m.GetLastEventTime(),
		"execution_stats":      m.ExecutionDuration.Stats(),
	}
}

// AtomicCounter provides a thread-safe counter
type AtomicCounter struct {
	value int64
}

// NewAtomicCounter creates a new AtomicCounter
func NewAtomicCounter() *AtomicCounter {
	return &AtomicCounter{}
}

// Add atomically adds delta to the counter
func (c *AtomicCounter) Add(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

// Inc atomically increments the counter by 1
func (c *AtomicCounter) Inc() {
	c.Add(1)
}

// Value atomically loads the current value
func (c *AtomicCounter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// Reset atomically resets the counter to 0
func (c *AtomicCounter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// AtomicGauge provides a thread-safe gauge for values that can go up and down
type AtomicGauge struct {
	value int64
}

// NewAtomicGauge creates a new AtomicGauge
func NewAtomicGauge() *AtomicGauge {
	return &AtomicGauge{}
}

// Set atomically sets the gauge value
func (g *AtomicGauge) Set(value int64) {
	atomic.StoreInt64(&g.value, value)
}

// Store is an alias for Set
func (g *AtomicGauge) Store(value int64) {
	g.Set(value)
}

// Add atomically adds delta to the gauge
func (g *AtomicGauge) Add(delta int64) {
	atomic.AddInt64(&g.value, delta)
}

// Sub atomically subtracts delta from the gauge
func (g *AtomicGauge) Sub(delta int64) {
	atomic.AddInt64(&g.value, -delta)
}

// Value atomically loads the current value
func (g *AtomicGauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}

// Histogram provides basic histogram functionality for tracking distributions
type Histogram struct {
	mu      sync.RWMutex
	samples []float64
	count   int64
	sum     float64
	min     float64
	max     float64
}

// NewHistogram creates a new Histogram
func NewHistogram() *Histogram {
	return &Histogram{
		samples: make([]float64, 0, 1000), // Initial capacity
		min:     0,
		max:     0,
	}
}

// Observe adds a new observation to the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.count++
	h.sum += value

	// Update min/max
	if h.count == 1 {
		h.min = value
		h.max = value
	} else {
		if value < h.min {
			h.min = value
		}
		if value > h.max {
			h.max = value
		}
	}

	// Store sample (with basic capacity management)
	if len(h.samples) < cap(h.samples) {
		h.samples = append(h.samples, value)
	} else {
		// Replace oldest sample (simple ring buffer behavior)
		h.samples[h.count%int64(len(h.samples))] = value
	}
}

// Stats returns statistical information about the histogram
func (h *Histogram) Stats() map[string]float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]float64{
		"count": float64(h.count),
		"sum":   h.sum,
		"min":   h.min,
		"max":   h.max,
	}

	if h.count > 0 {
		stats["mean"] = h.sum / float64(h.count)
	}

	// Calculate percentiles if we have enough samples
	if len(h.samples) > 0 {
		// Make a copy for sorting
		samples := make([]float64, len(h.samples))
		copy(samples, h.samples)

		// Simple bubble sort for small datasets
		for i := 0; i < len(samples)-1; i++ {
			for j := 0; j < len(samples)-i-1; j++ {
				if samples[j] > samples[j+1] {
					samples[j], samples[j+1] = samples[j+1], samples[j]
				}
			}
		}

		// Calculate percentiles
		if len(samples) >= 2 {
			stats["p50"] = percentile(samples, 0.5)
			stats["p90"] = percentile(samples, 0.9)
			stats["p95"] = percentile(samples, 0.95)
			stats["p99"] = percentile(samples, 0.99)
		}
	}

	return stats
}

// percentile calculates the percentile value from sorted samples
func percentile(sortedSamples []float64, p float64) float64 {
	if len(sortedSamples) == 0 {
		return 0
	}
	if len(sortedSamples) == 1 {
		return sortedSamples[0]
	}

	index := p * float64(len(sortedSamples)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedSamples) {
		return sortedSamples[len(sortedSamples)-1]
	}

	weight := index - float64(lower)
	return sortedSamples[lower]*(1-weight) + sortedSamples[upper]*weight
}

// Reset clears all histogram data
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples = h.samples[:0]
	h.count = 0
	h.sum = 0
	h.min = 0
	h.max = 0
}
