package metrics

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// MemoryCollector implements Collector interface using in-memory storage.
type MemoryCollector struct {
	nodeMetrics  map[string][]NodeMetrics
	podMetrics   map[string]*PodMetrics
	events       []Event
	queueMetrics []QueueMetrics
	mu           sync.RWMutex
}

// NewMemoryCollector creates a new in-memory metrics collector.
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		events:       make([]Event, 0),
		nodeMetrics:  make(map[string][]NodeMetrics),
		podMetrics:   make(map[string]*PodMetrics),
		queueMetrics: make([]QueueMetrics, 0),
	}
}

// RecordEvent records a single event.
func (c *MemoryCollector) RecordEvent(ctx context.Context, event Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, event)

	return nil
}

// RecordNodeMetrics records node resource metrics.
func (c *MemoryCollector) RecordNodeMetrics(ctx context.Context, metrics NodeMetrics) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodeMetrics[metrics.NodeName] == nil {
		c.nodeMetrics[metrics.NodeName] = make([]NodeMetrics, 0)
	}

	c.nodeMetrics[metrics.NodeName] = append(c.nodeMetrics[metrics.NodeName], metrics)

	return nil
}

// RecordPodMetrics records pod lifecycle metrics.
func (c *MemoryCollector) RecordPodMetrics(ctx context.Context, metrics PodMetrics) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s/%s", metrics.Namespace, metrics.Name)

	if existing, ok := c.podMetrics[key]; ok {
		// Update existing metrics.
		if metrics.ScheduledAt != nil {
			existing.ScheduledAt = metrics.ScheduledAt
		}

		if metrics.RunningAt != nil {
			existing.RunningAt = metrics.RunningAt
		}

		if metrics.DeletedAt != nil {
			existing.DeletedAt = metrics.DeletedAt
		}

		if metrics.PendingDuration != nil {
			existing.PendingDuration = metrics.PendingDuration
		}

		if metrics.RunningDuration != nil {
			existing.RunningDuration = metrics.RunningDuration
		}

		if metrics.Node != "" {
			existing.Node = metrics.Node
		}
	} else {
		c.podMetrics[key] = &metrics
	}

	return nil
}

// GetEvents returns all recorded events.
func (c *MemoryCollector) GetEvents() []Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	events := make([]Event, len(c.events))
	copy(events, c.events)

	return events
}

// GetNodeMetrics returns metrics for a specific node.
func (c *MemoryCollector) GetNodeMetrics(nodeName string) []NodeMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := c.nodeMetrics[nodeName]
	if metrics == nil {
		return []NodeMetrics{}
	}

	result := make([]NodeMetrics, len(metrics))
	copy(result, metrics)

	return result
}

// GetPodMetrics returns metrics for a specific pod.
func (c *MemoryCollector) GetPodMetrics(namespace, name string) *PodMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	if metrics, ok := c.podMetrics[key]; ok {
		// Return a copy.
		copy := *metrics
		return &copy
	}

	return nil
}

// RecordQueueMetrics records scheduler queue metrics.
func (c *MemoryCollector) RecordQueueMetrics(ctx context.Context, metrics QueueMetrics) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queueMetrics = append(c.queueMetrics, metrics)

	return nil
}

// ExportMetrics exports all metrics to the specified format.
func (c *MemoryCollector) ExportMetrics(ctx context.Context, format, outputPath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create output directory.
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch format {
	case "csv":
		return c.exportCSV(outputPath)
	case "json":
		return c.exportJSON(outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportCSV exports metrics in CSV format.
func (c *MemoryCollector) exportCSV(outputPath string) error {
	// Export events.
	if err := c.exportEventsCSV(filepath.Join(outputPath, "event_history.csv")); err != nil {
		return err
	}

	// Export pod metrics.
	if err := c.exportPodMetricsCSV(outputPath); err != nil {
		return err
	}

	// Export node metrics.
	if err := c.exportNodeMetricsCSV(outputPath); err != nil {
		return err
	}

	// Export queue metrics.
	if err := c.exportQueueMetricsCSV(filepath.Join(outputPath, "pod_queue_length.csv")); err != nil {
		return err
	}

	return nil
}

// exportEventsCSV exports events to CSV.
func (c *MemoryCollector) exportEventsCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create events file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header.
	if err := writer.Write([]string{"timestamp", "type", "name", "namespace", "node", "details"}); err != nil {
		return err
	}

	// Sort events by timestamp.
	sort.Slice(c.events, func(i, j int) bool {
		return c.events[i].Timestamp.Before(c.events[j].Timestamp)
	})

	// Write events.
	for _, event := range c.events {
		details, _ := json.Marshal(event.Details)

		record := []string{
			event.Timestamp.Format(time.RFC3339Nano),
			string(event.Type),
			event.Name,
			event.Namespace,
			event.Node,
			string(details),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportPodMetricsCSV exports pod metrics to CSV files.
func (c *MemoryCollector) exportPodMetricsCSV(outputPath string) error {
	// Pending durations.
	pendingFile, err := os.Create(filepath.Join(outputPath, "pod_pending_durations.csv"))
	if err != nil {
		return fmt.Errorf("failed to create pending durations file: %w", err)
	}
	defer pendingFile.Close()

	pendingWriter := csv.NewWriter(pendingFile)
	defer pendingWriter.Flush()

	if err := pendingWriter.Write([]string{"pod", "namespace", "pending_duration_ms"}); err != nil {
		return err
	}

	// Running durations.
	runningFile, err := os.Create(filepath.Join(outputPath, "pod_running_durations.csv"))
	if err != nil {
		return fmt.Errorf("failed to create running durations file: %w", err)
	}
	defer runningFile.Close()

	runningWriter := csv.NewWriter(runningFile)
	defer runningWriter.Flush()

	if err := runningWriter.Write([]string{"pod", "namespace", "running_duration_ms"}); err != nil {
		return err
	}

	// Write pod metrics.
	for _, metrics := range c.podMetrics {
		if metrics.PendingDuration != nil {
			record := []string{
				metrics.Name,
				metrics.Namespace,
				fmt.Sprintf("%d", metrics.PendingDuration.Milliseconds()),
			}
			if err := pendingWriter.Write(record); err != nil {
				return err
			}
		}

		if metrics.RunningDuration != nil {
			record := []string{
				metrics.Name,
				metrics.Namespace,
				fmt.Sprintf("%d", metrics.RunningDuration.Milliseconds()),
			}
			if err := runningWriter.Write(record); err != nil {
				return err
			}
		}
	}

	return nil
}

// exportNodeMetricsCSV exports node metrics to CSV files.
func (c *MemoryCollector) exportNodeMetricsCSV(outputPath string) error {
	for nodeName, metrics := range c.nodeMetrics {
		// Allocation history.
		allocFile, err := os.Create(filepath.Join(outputPath, fmt.Sprintf("node-%s_allocation_history.csv", nodeName)))
		if err != nil {
			return fmt.Errorf("failed to create allocation history file: %w", err)
		}
		defer allocFile.Close()

		allocWriter := csv.NewWriter(allocFile)
		defer allocWriter.Flush()

		if err := allocWriter.Write([]string{"timestamp", "cpu_allocated", "memory_allocated", "pods_allocated"}); err != nil {
			return err
		}

		// Sort by timestamp.
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Timestamp.Before(metrics[j].Timestamp)
		})

		for _, m := range metrics {
			record := []string{
				m.Timestamp.Format(time.RFC3339Nano),
				fmt.Sprintf("%d", m.CPUAllocated),
				fmt.Sprintf("%d", m.MemoryAllocated),
				fmt.Sprintf("%d", m.PodsAllocated),
			}
			if err := allocWriter.Write(record); err != nil {
				return err
			}
		}
	}

	return nil
}

// exportQueueMetricsCSV exports queue metrics to CSV.
func (c *MemoryCollector) exportQueueMetricsCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create queue metrics file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"timestamp", "queue_length"}); err != nil {
		return err
	}

	// Sort by timestamp.
	sort.Slice(c.queueMetrics, func(i, j int) bool {
		return c.queueMetrics[i].Timestamp.Before(c.queueMetrics[j].Timestamp)
	})

	for _, m := range c.queueMetrics {
		record := []string{
			m.Timestamp.Format(time.RFC3339Nano),
			fmt.Sprintf("%d", m.QueueLength),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportJSON exports all metrics in JSON format.
func (c *MemoryCollector) exportJSON(outputPath string) error {
	data := map[string]interface{}{
		"events":       c.events,
		"nodeMetrics":  c.nodeMetrics,
		"podMetrics":   c.podMetrics,
		"queueMetrics": c.queueMetrics,
	}

	file, err := os.Create(filepath.Join(outputPath, "metrics.json"))
	if err != nil {
		return fmt.Errorf("failed to create metrics file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode metrics: %w", err)
	}

	return nil
}

// Reset clears all collected metrics.
func (c *MemoryCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = make([]Event, 0)
	c.nodeMetrics = make(map[string][]NodeMetrics)
	c.podMetrics = make(map[string]*PodMetrics)
	c.queueMetrics = make([]QueueMetrics, 0)
}
