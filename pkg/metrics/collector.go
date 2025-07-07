package metrics

import (
	"context"
	"time"
)

// EventType represents the type of event being recorded.
type EventType string

const (
	EventTypePodCreated   EventType = "pod_created"
	EventTypePodScheduled EventType = "pod_scheduled"
	EventTypePodRunning   EventType = "pod_running"
	EventTypePodDeleted   EventType = "pod_deleted"
	EventTypePodFailed    EventType = "pod_failed"
)

// Event represents a single metric event.
type Event struct {
	Timestamp time.Time
	Details   map[string]interface{}
	Type      EventType
	Name      string
	Namespace string
	Node      string
}

// NodeMetrics represents node resource metrics.
type NodeMetrics struct {
	Timestamp       time.Time
	NodeName        string
	CPUAllocated    int64
	CPUCapacity     int64
	MemoryAllocated int64
	MemoryCapacity  int64
	PodsAllocated   int
	PodsCapacity    int
}

// PodMetrics represents pod lifecycle metrics.
type PodMetrics struct {
	Name            string
	Namespace       string
	CreatedAt       time.Time
	ScheduledAt     *time.Time
	RunningAt       *time.Time
	DeletedAt       *time.Time
	PendingDuration *time.Duration
	RunningDuration *time.Duration
	Node            string
	CPURequested    int64
	MemoryRequested int64
}

// Collector interface for metrics collection.
type Collector interface {
	// RecordEvent records a single event.
	RecordEvent(ctx context.Context, event Event) error

	// RecordNodeMetrics records node resource metrics.
	RecordNodeMetrics(ctx context.Context, metrics NodeMetrics) error

	// RecordPodMetrics records pod lifecycle metrics.
	RecordPodMetrics(ctx context.Context, metrics PodMetrics) error

	// GetEvents returns all recorded events.
	GetEvents() []Event

	// GetNodeMetrics returns metrics for a specific node.
	GetNodeMetrics(nodeName string) []NodeMetrics

	// GetPodMetrics returns metrics for a specific pod.
	GetPodMetrics(namespace, name string) *PodMetrics

	// ExportMetrics exports all metrics to the specified format.
	ExportMetrics(ctx context.Context, format, outputPath string) error

	// Reset clears all collected metrics.
	Reset()
}

// QueueMetrics represents scheduler queue metrics.
type QueueMetrics struct {
	Timestamp   time.Time
	PendingPods []string
	QueueLength int
}

// SimulationSummary represents overall simulation metrics.
type SimulationSummary struct {
	StartTime          time.Time
	EndTime            time.Time
	NodeUtilization    map[string]float64
	Duration           time.Duration
	TotalPods          int
	SuccessfulPods     int
	FailedPods         int
	AveragePendingTime time.Duration
	AverageRunningTime time.Duration
	MaxQueueLength     int
}
