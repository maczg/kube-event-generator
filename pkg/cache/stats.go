package cache

import (
	"encoding/csv"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	defaultTimeFormat = "2006-01-02T15:04:05.000"
)

const (
	PodPendingDurationKey     = "pod_pending_durations"
	PodRunningDurationKey     = "pod_running_durations"
	PodQueueLengthKey         = "pod_queue_length"
	AllocationHistoryKey      = "allocation_history"
	AllocationRatioHistoryKey = "allocation_ratio_history"
)

type PendingQAction int

const (
	AddPodToPendingQ PendingQAction = iota
	RemovePodFromPendingQ
)

type Stats struct {
	// PendingQ is a map of pod to the pod object that are in the pending queue.
	PendingQ map[Key]*v1.Pod
	// PendingQHistory is a history of the length of the pending queue.
	PendingQHistory []Record[int]
	// PendingDurations is a map of pod to the time the pod spent in the pending queue.
	PendingDurations map[Key]time.Duration
	// ExecutionDuration is a map of pod to the time the pod spent running.
	RunningDurations map[Key]time.Duration

	//AllocationHistory is a history of the allocated resource on the cluster nodes.
	AllocationHistory      map[Key][]Record[v1.ResourceList]
	AllocationRatioHistory map[Key][]Record[map[v1.ResourceName]float64]
}

func NewStats() *Stats {
	return &Stats{
		PendingQ:               make(map[Key]*v1.Pod),
		PendingDurations:       make(map[Key]time.Duration),
		RunningDurations:       make(map[Key]time.Duration),
		PendingQHistory:        make([]Record[int], 0),
		AllocationHistory:      make(map[Key][]Record[v1.ResourceList]),
		AllocationRatioHistory: make(map[Key][]Record[map[v1.ResourceName]float64]),
	}
}

func (ps *Stats) UpdateHistory(nodeStore NodeStore) {
	key := NewKey(nodeStore.Node)
	if _, ok := ps.AllocationHistory[key]; !ok {
		ps.AllocationHistory[key] = make([]Record[v1.ResourceList], 0)
	}
	allocated := nodeStore.GetAllocated()
	ps.AllocationHistory[key] = append(ps.AllocationHistory[key], Record[v1.ResourceList]{
		At:    time.Now(),
		Value: allocated,
	})

	allocatedRatio := nodeStore.GetAllocatedRatio()

	if _, ok := ps.AllocationRatioHistory[key]; !ok {
		ps.AllocationRatioHistory[key] = make([]Record[map[v1.ResourceName]float64], 0)
	}

	ratioMap := make(map[v1.ResourceName]float64)
	for k, v := range allocatedRatio {
		ratioMap[k] = v
	}
	ps.AllocationRatioHistory[key] = append(ps.AllocationRatioHistory[key], Record[map[v1.ResourceName]float64]{
		At:    time.Now(),
		Value: ratioMap,
	})
}

func (ps *Stats) UpdatePendingQ(pod *v1.Pod, action PendingQAction) {
	key := NewKey(pod)
	switch action {
	case AddPodToPendingQ:
		ps.PendingQ[key] = pod
	case RemovePodFromPendingQ:
		ps.PendingDurations[key] = time.Since(pod.CreationTimestamp.Time)
		delete(ps.PendingQ, key)
	}
	ps.PendingQHistory = append(ps.PendingQHistory, Record[int]{
		Value: len(ps.PendingQ),
		At:    time.Now(),
	})
}

func (ps *Stats) GetPodQueueHistory() []Record[int] {
	cp := make([]Record[int], len(ps.PendingQHistory))
	copy(cp, ps.PendingQHistory)
	return cp
}

func (ps *Stats) ExportCSV(dir string) error {
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

	nodeAllocationHistoryFiles := make(map[string]*os.File)
	nodeAllocationRatioHistoryFiles := make(map[string]*os.File)
	for nodeKey := range ps.AllocationHistory {
		f, _ := os.Create(fmt.Sprintf("%s/%s_%s.csv", dir, nodeKey.Name, AllocationHistoryKey))
		nodeAllocationHistoryFiles[nodeKey.Name] = f
	}

	for nodeKey := range ps.AllocationRatioHistory {
		f, _ := os.Create(fmt.Sprintf("%s/%s_%s.csv", dir, nodeKey.Name, AllocationRatioHistoryKey))
		nodeAllocationRatioHistoryFiles[nodeKey.Name] = f
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
	writer.Write(header)
	for k, pendingTime := range ps.PendingDurations {
		writer.Write([]string{k.GetUID(), k.GetName(), fmt.Sprintf("%d", pendingTime.Milliseconds())})
	}
	writer.Flush()

	// Export the running times
	header = []string{"pod_uid", "pod_name", "running_time_milliseconds"}
	writer = csv.NewWriter(fileRunningDurations)
	writer.Write(header)
	for k, runningTime := range ps.RunningDurations {
		writer.Write([]string{k.GetUID(), k.GetName(), fmt.Sprintf("%d", runningTime.Milliseconds())})
	}
	writer.Flush()

	// Export the queue length history
	header = []string{"timestamp", "length"}
	writer = csv.NewWriter(fileQueueLength)
	writer.Write(header)
	podQHistory := ps.GetPodQueueHistory()
	sort.Slice(podQHistory, func(i, j int) bool {
		return podQHistory[i].At.Before(podQHistory[j].At)
	})
	for _, value := range podQHistory {
		writer.Write([]string{value.At.Format(defaultTimeFormat), strconv.Itoa(value.Value)})
	}
	writer.Flush()

	// Export the allocation history
	for k, hist := range ps.AllocationHistory {
		file := nodeAllocationHistoryFiles[k.Name]
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
	for nodeKey, hist := range ps.AllocationRatioHistory {
		file := nodeAllocationRatioHistoryFiles[nodeKey.Name]
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
	return nil
}
