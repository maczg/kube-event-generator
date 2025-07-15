package cache

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	defaultTimeFormat         = "2006-01-02T15:04:05.000"
	PodPendingDurationKey     = "pod_pending_durations"
	PodRunningDurationKey     = "pod_running_durations"
	PodQueueLengthKey         = "pod_queue_length"
	AllocationHistoryKey      = "allocation_history"
	AllocationRatioHistoryKey = "allocation_ratio_history"
	FreeHistoryKey            = "free_resource_history"
)

// PendingQAction represents the action to be performed on the pending queue.
type PendingQAction int

const (
	AddPodToPendingQ PendingQAction = iota
	RemovePodFromPendingQ
)

// PodEvent represents an event related to a pod in the cluster.
type PodEvent struct {
	PodName   string
	NodeName  string
	Phase     string
	EventType string
	CPUReq    string
	MemReq    string
}

func (p *PodEvent) String() string {
	return fmt.Sprintf("[PodEvent] podName: %s, nodeName: %s, phase: %s, eventType: %s, cpuReq: %s, memReq: %s}",
		p.PodName, p.NodeName, p.Phase, p.EventType, p.CPUReq, p.MemReq)
}

// NewPodEvent creates a new PodEvent from a v1.Pod object and an event type.
func NewPodEvent(pod *v1.Pod, eventType string) PodEvent {
	cpu := pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]
	mem := pod.Spec.Containers[0].Resources.Requests[v1.ResourceMemory]

	return PodEvent{
		PodName:   pod.Name,
		NodeName:  pod.Spec.NodeName,
		Phase:     string(pod.Status.Phase),
		EventType: eventType,
		CPUReq:    cpu.String(),
		MemReq:    mem.String(),
	}
}

type Stats struct {
	// PendingQ is a map of pod to the pod object that are in the pending queue.
	PendingQ map[Key]*v1.Pod
	// PendingQHistory is a history of the length of the pending queue.
	PendingQHistory []Record[int]
	// PendingDurations is a map of pod to the time the pod spent in the pending queue.
	PendingDurations map[Key]time.Duration
	// ExecutionDuration is a map of pod to the time the pod spent running.
	RunningDurations map[Key]time.Duration

	// AllocationHistory is a history of the allocated resource on the cluster nodes.
	AllocationHistory      map[Key][]Record[v1.ResourceList]
	AllocationRatioHistory map[Key][]Record[map[v1.ResourceName]float64]
	// ResourceFreeHistory is a history of the free resource on the cluster nodes.
	ResourceFreeHistory map[Key][]Record[v1.ResourceList]
	PodEventHistory     []Record[PodEvent]
}

// NewStats creates a new Stats object with initialized maps and slices.
func NewStats() *Stats {
	return &Stats{
		PendingQ:               make(map[Key]*v1.Pod),
		PendingDurations:       make(map[Key]time.Duration),
		RunningDurations:       make(map[Key]time.Duration),
		PendingQHistory:        make([]Record[int], 0),
		AllocationHistory:      make(map[Key][]Record[v1.ResourceList]),
		AllocationRatioHistory: make(map[Key][]Record[map[v1.ResourceName]float64]),
		ResourceFreeHistory:    make(map[Key][]Record[v1.ResourceList]),
		PodEventHistory:        make([]Record[PodEvent], 0),
	}
}

// UpdatePodEvent updates the running duration of a pod.
func (s *Stats) UpdatePodEvent(podEvent PodEvent) {
	s.PodEventHistory = append(s.PodEventHistory, Record[PodEvent]{
		At:    time.Now(),
		Value: podEvent,
	})
}

func (s *Stats) UpdateHistory(nodeStore NodeStore) {
	key := NewKey(nodeStore.Node)
	if _, ok := s.AllocationHistory[key]; !ok {
		s.AllocationHistory[key] = make([]Record[v1.ResourceList], 0)
	}

	allocated := nodeStore.GetAllocated()
	s.AllocationHistory[key] = append(s.AllocationHistory[key], Record[v1.ResourceList]{
		At:    time.Now(),
		Value: allocated,
	})
	allocatedRatio := nodeStore.GetAllocatedRatio()

	if _, ok := s.AllocationRatioHistory[key]; !ok {
		s.AllocationRatioHistory[key] = make([]Record[map[v1.ResourceName]float64], 0)
	}

	ratioMap := make(map[v1.ResourceName]float64)
	for k, v := range allocatedRatio {
		ratioMap[k] = v
	}

	s.AllocationRatioHistory[key] = append(s.AllocationRatioHistory[key], Record[map[v1.ResourceName]float64]{
		At:    time.Now(),
		Value: ratioMap,
	})

	if _, ok := s.ResourceFreeHistory[key]; !ok {
		s.ResourceFreeHistory[key] = make([]Record[v1.ResourceList], 0)
	}

	free := nodeStore.GetFree()
	s.ResourceFreeHistory[key] = append(s.ResourceFreeHistory[key], Record[v1.ResourceList]{
		At:    time.Now(),
		Value: free,
	})
}

func (s *Stats) UpdatePendingQ(pod *v1.Pod, action PendingQAction) {
	key := NewKey(pod)

	switch action {
	case AddPodToPendingQ:
		s.PendingQ[key] = pod
	case RemovePodFromPendingQ:
		s.PendingDurations[key] = time.Since(pod.CreationTimestamp.Time)
		delete(s.PendingQ, key)
	}

	s.PendingQHistory = append(s.PendingQHistory, Record[int]{
		Value: len(s.PendingQ),
		At:    time.Now(),
	})
}

func (s *Stats) GetPodQueueHistory() []Record[int] {
	cp := make([]Record[int], len(s.PendingQHistory))
	copy(cp, s.PendingQHistory)

	return cp
}

// ExportCSV exports the stats to a CSV file into the given directory.
// If the directory does not exist, it will be created.
func (s *Stats) ExportCSV(dir string) error {
	if dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	} else {
		dir = "."
	}

	podPendingDurations := fmt.Sprintf("%s/%s.csv", dir, PodPendingDurationKey)
	podRunningDurations := fmt.Sprintf("%s/%s.csv", dir, PodRunningDurationKey)
	podQueueLength := fmt.Sprintf("%s/%s.csv", dir, PodQueueLengthKey)

	nodeAllocationFiles := make(map[string]*os.File)
	nodeAllocationRatioFiles := make(map[string]*os.File)
	nodeFreeFiles := make(map[string]*os.File)

	for nodeKey := range s.AllocationHistory {
		f, _ := os.Create(fmt.Sprintf("%s/%s_%s.csv", dir, nodeKey.Name, AllocationHistoryKey))
		nodeAllocationFiles[nodeKey.Name] = f
	}

	for nodeKey := range s.AllocationRatioHistory {
		f, _ := os.Create(fmt.Sprintf("%s/%s_%s.csv", dir, nodeKey.Name, AllocationRatioHistoryKey))
		nodeAllocationRatioFiles[nodeKey.Name] = f
	}

	for nodeKey := range s.ResourceFreeHistory {
		f, _ := os.Create(fmt.Sprintf("%s/%s_%s.csv", dir, nodeKey.Name, FreeHistoryKey))
		nodeFreeFiles[nodeKey.Name] = f
	}

	filePendingDurations, err := os.Create(podPendingDurations)
	if err != nil {
		return err
	}

	fileRunningDurations, err := os.Create(podRunningDurations)
	if err != nil {
		return err
	}

	fileQueueLength, err := os.Create(podQueueLength)
	if err != nil {
		return err
	}

	defer filePendingDurations.Close()
	defer fileRunningDurations.Close()
	defer fileQueueLength.Close()

	// Export the pending times
	header := []string{"pod_uid", "pod_name", "pending_time_milliseconds"}
	writer := csv.NewWriter(filePendingDurations)
	_ = writer.Write(header)

	for k, pendingTime := range s.PendingDurations {
		_ = writer.Write([]string{k.GetUID(), k.GetName(), fmt.Sprintf("%d", pendingTime.Milliseconds())})
	}

	writer.Flush()

	// Export the running times
	header = []string{"pod_uid", "pod_name", "running_time_milliseconds"}
	writer = csv.NewWriter(fileRunningDurations)
	_ = writer.Write(header)

	for k, runningTime := range s.RunningDurations {
		_ = writer.Write([]string{k.GetUID(), k.GetName(), fmt.Sprintf("%d", runningTime.Milliseconds())})
	}

	writer.Flush()

	// Export the queue length history
	header = []string{"timestamp", "length"}
	writer = csv.NewWriter(fileQueueLength)
	_ = writer.Write(header)

	podQHistory := s.GetPodQueueHistory()
	sort.Slice(podQHistory, func(i, j int) bool {
		return podQHistory[i].At.Before(podQHistory[j].At)
	})

	for _, value := range podQHistory {
		_ = writer.Write([]string{value.At.Format(defaultTimeFormat), strconv.Itoa(value.Value)})
	}

	writer.Flush()

	// Export the allocation history
	for k, hist := range s.AllocationHistory {
		file := nodeAllocationFiles[k.Name]
		writer = csv.NewWriter(file)
		resourceTypes := make(map[v1.ResourceName]struct{})

		for _, record := range hist {
			for resourceType := range record.Value {
				resourceTypes[resourceType] = struct{}{}
			}
		}

		header = []string{"timestamp"}
		for resourceType := range resourceTypes {
			header = append(header, string(resourceType))
		}

		if err = writer.Write(header); err != nil {
			return err
		}

		for _, record := range hist {
			row := make([]string, len(header))
			row[0] = record.At.Format(defaultTimeFormat)

			for i, resourceType := range header[1:] {
				if quantity, ok := record.Value[v1.ResourceName(resourceType)]; ok {
					row[i+1] = strconv.FormatInt(quantity.MilliValue(), 10)
				} else {
					row[i+1] = "0"
				}
			}

			if err = writer.Write(row); err != nil {
				return err
			}
		}

		writer.Flush()
	}

	// Export the allocation ratio history
	for nodeKey, hist := range s.AllocationRatioHistory {
		file := nodeAllocationRatioFiles[nodeKey.Name]
		writer = csv.NewWriter(file)
		resourceTypes := make(map[v1.ResourceName]struct{})

		for _, record := range hist {
			for resourceType := range record.Value {
				resourceTypes[resourceType] = struct{}{}
			}
		}

		header = []string{"timestamp"}
		for resourceType := range resourceTypes {
			header = append(header, string(resourceType))
		}

		if err = writer.Write(header); err != nil {
			return err
		}

		for _, record := range hist {
			row := make([]string, len(header))
			row[0] = record.At.Format(defaultTimeFormat)

			for i, resourceType := range header[1:] {
				if ratio, ok := record.Value[v1.ResourceName(resourceType)]; ok {
					row[i+1] = fmt.Sprintf("%.2f", ratio)
				} else {
					row[i+1] = "0"
				}
			}

			if err = writer.Write(row); err != nil {
				return err
			}
		}

		writer.Flush()
	}

	// Export the allocation ratio history
	for nodeKey, hist := range s.ResourceFreeHistory {
		file := nodeFreeFiles[nodeKey.Name]
		writer = csv.NewWriter(file)
		resourceTypes := make(map[v1.ResourceName]struct{})

		for _, record := range hist {
			for resourceType := range record.Value {
				resourceTypes[resourceType] = struct{}{}
			}
		}

		header = []string{"timestamp"}
		for resourceType := range resourceTypes {
			header = append(header, string(resourceType))
		}

		if err = writer.Write(header); err != nil {
			return err
		}

		for _, record := range hist {
			row := make([]string, len(header))
			row[0] = record.At.Format(defaultTimeFormat)

			for i, resourceType := range header[1:] {
				if quantity, ok := record.Value[v1.ResourceName(resourceType)]; ok {
					row[i+1] = strconv.FormatInt(quantity.MilliValue(), 10)
				} else {
					row[i+1] = "0"
				}
			}

			if err = writer.Write(row); err != nil {
				return err
			}
		}

		writer.Flush()
	}

	// event history
	fileEventHistory, err := os.Create(fmt.Sprintf("%s/event_history.csv", dir))
	if err != nil {
		return err
	}

	defer fileEventHistory.Close()
	writer = csv.NewWriter(fileEventHistory)
	header = []string{"timestamp", "pod_name", "node_name", "phase", "event_type", "cpu_req", "mem_req"}

	if err = writer.Write(header); err != nil {
		return err
	}

	for _, record := range s.PodEventHistory {
		row := []string{
			record.At.Format(defaultTimeFormat),
			record.Value.PodName,
			record.Value.NodeName,
			record.Value.Phase,
			record.Value.EventType,
			record.Value.CPUReq,
			record.Value.MemReq,
		}
		if err = writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()

	return nil
}
