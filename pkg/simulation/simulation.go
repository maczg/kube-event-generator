package simulation

import (
	"context"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"sync"
	"time"
)

const (
	simulatorName    = "simulator"
	defaultResultDir = "results"
)

type Simulation struct {
	ID        string
	logger    logger.Logger
	Scenario  *scenario.Scenario
	Scheduler scheduler.Scheduler
	kubeMgr   *utils.KubernetesManager
	errCh     chan error
	podMap    []string
	resultDir string
	stopCtx   context.Context
	stopFn    context.CancelCauseFunc
	mu        sync.Mutex
}

func NewWithOpts(opts ...Option) *Simulation {
	s := &Simulation{
		ID:        "",
		logger:    logger.NewLogger(logger.LevelInfo, simulatorName),
		Scenario:  nil,
		Scheduler: scheduler.New(),
		kubeMgr:   nil,
		errCh:     make(chan error),
		podMap:    make([]string, 0),
		resultDir: defaultResultDir,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.ID == "" {
		scnName := strings.ReplaceAll(strings.ToLower(s.Scenario.Name), " ", "-")
		s.ID = fmt.Sprintf("%s-%s", scnName, time.Now().Format("15-04-05"))
	}

	s.initializeEvents()
	return s

}

func New(scn *scenario.Scenario, manager *utils.KubernetesManager) *Simulation {
	ctx, fn := context.WithCancelCause(context.Background())
	scnName := strings.ReplaceAll(strings.ToLower(scn.Name), " ", "-")
	id := fmt.Sprintf("%s-%s", scnName, time.Now().Format("15-04-05"))
	s := &Simulation{
		ID:        id,
		logger:    logger.NewLogger(logger.LevelInfo, simulatorName),
		Scenario:  scn,
		Scheduler: scheduler.New(),
		kubeMgr:   manager,
		errCh:     make(chan error),
		podMap:    make([]string, 0),
		resultDir: defaultResultDir,
		stopCtx:   ctx,
		stopFn:    fn,
	}
	s.initializeEvents()
	return s
}

func (s *Simulation) initializeEvents() {
	for _, e := range s.Scenario.Events {
		if e.Duration != 0 {
			s.podMap = append(s.podMap, e.Pod.Name)
		}
		ev := NewSimEvent(e, s.kubeMgr, s.Scheduler)
		s.Scheduler.Schedule(ev)
	}
}

func (s *Simulation) initMetricValues() {
	for _, n := range s.Scenario.Cluster.Nodes {
		NodeResourceMetric.Set(n.Status.Capacity.Cpu().AsApproximateFloat64(), metric.WithLabels(n.Name, "cpu"))
		NodeResourceMetric.Set(n.Status.Capacity.Memory().AsApproximateFloat64(), metric.WithLabels(n.Name, "memory"))
		pendingPodQueueMetric.Add(0, metric.WithLabels("pending"))
	}

}

func (s *Simulation) startScheduler() {
	err := s.Scheduler.Start()
	if err != nil {
		s.errCh <- err
	}
}

func (s *Simulation) Start() error {
	var host string
	if s.kubeMgr.RestCfg() != nil {
		host = s.kubeMgr.RestCfg().Host
	} else {
		host = "unknown"
	}
	s.logger.Info("kube host: %s", host)

	s.logger.Info("resetting cluster")
	if err := s.resetCluster(); err != nil {
		return err
	}

	s.initMetricValues()
	s.logger.Info("starting simulation %s", s.ID)

	go s.startScheduler()
	go s.watchState()
	go s.checkAllPodsScheduled()
	return s.waitToFinish()

}

func (s *Simulation) waitToFinish() error {
	for {
		select {
		case <-s.stopCtx.Done():

			s.exportMetrics()

			err := s.stopCtx.Err()
			s.logger.Info("simulation %s stopping. Cause: %s ", s.ID, s.stopCtx.Err())
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrAllPodFinished) {
				return nil
			}
			return err
		case err := <-s.errCh:
			s.logger.Error("error in simulation %s: %v", s.ID, err)
			s.stopFn(err)
			return err
		}
	}
}

func (s *Simulation) exportMetrics() {
	s.summarizeMetrics()

	resultDir := fmt.Sprintf("%s/%s", s.resultDir, s.ID)

	if err := NodeResourceMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := eventTimelineMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := podPendingDurationMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := pendingPodQueueMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
}

func (s *Simulation) Stop() {
	s.stopFn(nil)
}

// checkAllPodsScheduled checks if the inner pods in queue are all finished.
// this ensures that all pod with duration have done
func (s *Simulation) checkAllPodsScheduled() {
	for {
		select {
		case <-s.stopCtx.Done():
			return
		default:
			s.mu.Lock()
			if len(s.podMap) == 0 {
				s.mu.Unlock()
				s.stopFn(ErrAllPodFinished)
				return
			}
			s.mu.Unlock()
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Simulation) watchState() {
	w, err := s.kubeMgr.Clientset().CoreV1().Pods("").Watch(context.Background(), metav1.ListOptions{})
	defer w.Stop()
	if err != nil {
		s.logger.Error("error watching pods: %v", err)
		s.stopFn(err)
		return
	}

	runningPod := make(map[string]bool)

	for {
		select {
		case <-s.stopCtx.Done():
			s.logger.Info("watcher %s stopping", s.ID)
			return

		case e := <-w.ResultChan():
			s.handlePodEvent(e, runningPod)
		}
	}
}

func (s *Simulation) resetCluster() error {
	err := s.kubeMgr.ResetNodes()
	time.Sleep(2 * time.Second)
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	err = s.kubeMgr.ResetPods()
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	// create cluster
	for _, n := range s.Scenario.Cluster.Nodes {
		err = s.kubeMgr.CreateNode(context.Background(), *s.kubeMgr.Clientset(), &n)
		if err != nil {
			return err
		}
	}
	return s.kubeMgr.ResetPods()
}

func (s *Simulation) summarizeMetrics() {
	s.logger.Info("===== Simulation Summary =====")

	// 1) Summarize Pod Pending Durations (pod_pending_duration metric)
	records := podPendingDurationMetric.Values()
	var total, maxVal, avg float64
	if len(records) > 0 {
		for _, r := range records {
			total += r.Value()
			if r.Value() > maxVal {
				maxVal = r.Value()
			}
		}
		avg = total / float64(len(records))
		s.logger.Info("Pod Pending Duration: count=%d, avg=%.2fs, max=%.2fs",
			len(records), avg, maxVal)
	} else {
		s.logger.Info("No pods recorded in podPendingDurationMetric.")
	}

	// 2) Summarize Pending Queue Length (pending_pods metric)
	queueRecs := pendingPodQueueMetric.Values()
	var maxQueue float64
	if len(queueRecs) > 0 {

		for _, r := range queueRecs {
			// Only look at “pending” label if that’s how you track the queue
			if statusVal, ok := r.Labels()["status"]; ok && statusVal == "pending" {
				if r.Value() > maxQueue {
					maxQueue = r.Value()
				}
			}
		}
		s.logger.Info("Max Pending Queue Length: %.0f", maxQueue)
	} else {
		s.logger.Info("No records found in pendingPodQueueMetric.")
	}

	s.summarizeNodeUtilization()

	//// 3) Summarize Final Node Resource Usage (node_resource_usage metric)
	////    Here we’ll look at each node’s last CPU/memory record we have.
	//nodeRecs := NodeResourceMetric.Values()
	//// A map of (nodeName -> resourceKind -> float64 (last usage))
	//finalUsage := make(map[string]map[string]float64)
	//for _, r := range nodeRecs {
	//	nodeName := r.Labels()["node"]     // from metric.WithLabels(nodeName, "cpu" or "memory")
	//	resource := r.Labels()["resource"] // "cpu" or "memory"
	//	if _, ok := finalUsage[nodeName]; !ok {
	//		finalUsage[nodeName] = make(map[string]float64)
	//	}
	//	finalUsage[nodeName][resource] = r.Value()
	//}
	//for node, usageMap := range finalUsage {
	//	s.logger.Info("Final usage for node %s -> CPU: %.2f, Memory: %.2f",
	//		node,
	//		usageMap["cpu"],
	//		usageMap["memory"])
	//}

	s.logger.Info("===== End of Simulation Summary =====")
}

func (s *Simulation) summarizeNodeUtilization() {
	nodeCapacities := make(map[string]map[string]float64) // nodeName -> { "cpu": <cap>, "memory": <cap> }
	type ratioStats struct {
		values []float64
	}
	nodeRatios := make(map[string]map[string]*ratioStats)

	for _, node := range s.Scenario.Cluster.Nodes {
		nodeCapacities[node.Name] = map[string]float64{
			"cpu":    node.Status.Capacity.Cpu().AsApproximateFloat64(),
			"memory": node.Status.Capacity.Memory().AsApproximateFloat64(),
		}
		nodeRatios[node.Name] = map[string]*ratioStats{
			"cpu":    &ratioStats{values: []float64{}},
			"memory": &ratioStats{values: []float64{}},
		}
	}
	usageRecords := NodeResourceMetric.Values()
	for _, r := range usageRecords {
		nodeName := r.Labels()["node"]
		resource := r.Labels()["resource"]
		capMap, nodeExists := nodeCapacities[nodeName]
		if !nodeExists {
			// Possibly we have a usage record for a node not in scenario, skip it
			continue
		}
		capVal := capMap[resource]
		if capVal == 0 {
			// Avoid division by zero if the node somehow has zero capacity
			continue
		}
		// ratio = usage / capacity
		ratio := r.Value() / capVal

		stats, ok := nodeRatios[nodeName][resource]
		if !ok {
			// If we see a resource type not in scenario, skip
			continue
		}
		stats.values = append(stats.values, ratio)
	}

	for nodeName, resMap := range nodeRatios {
		s.logger.Info("Node: %s", nodeName)
		for resource, stats := range resMap {
			values := stats.values
			if len(values) == 0 {
				s.logger.Info("  Resource: %s -> No usage records", resource)
				continue
			}
			var sum, minVal, maxVal float64
			minVal = 999999.0
			for _, v := range values {
				sum += v
				if v < minVal {
					minVal = v
				}
				if v > maxVal {
					maxVal = v
				}
			}
			avgVal := sum / float64(len(values))
			s.logger.Info("  Resource: %s -> data points=%d, min=%.2f, avg=%.2f, max=%.2f",
				resource, len(values), minVal, avgVal, maxVal)
		}
	}

}
