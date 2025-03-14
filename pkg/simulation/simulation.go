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

const simulatorName = "simulator"

type Simulation struct {
	ID        string
	logger    logger.Logger
	Scenario  *scenario.Scenario
	Scheduler scheduler.Scheduler
	kubeMgr   *utils.KubernetesManager
	errCh     chan error
	podMap    []string
	stopCtx   context.Context
	stopFn    context.CancelCauseFunc
	mu        sync.Mutex
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
	if err := NodeResourceMetric.ExportCSV(s.ID); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := eventTimelineMetric.ExportCSV(s.ID); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := podPendingDurationMetric.ExportCSV(s.ID); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := pendingPodQueueMetric.ExportCSV(s.ID); err != nil {
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
