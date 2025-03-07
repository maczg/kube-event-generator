package simulation

import (
	"context"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/cache"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"sync"
	"time"
)

const (
	simulatorName    = "simulator"
	defaultResultDir = "results"
)

type Simulation struct {
	ID        string
	Scenario  *scenario.Scenario
	Scheduler scheduler.Scheduler
	kubeMgr   *utils.KubernetesManager
	errCh     chan error
	podMap    []string
	resultDir string
	stopCtx   context.Context
	stopFn    context.CancelCauseFunc
	startTime time.Time
	informer  *cache.Store
	mu        sync.Mutex
}

func New(scn *scenario.Scenario, manager *utils.KubernetesManager) *Simulation {
	id := fmt.Sprintf("run-%s", time.Now().Format("15_04_05_020106"))
	ctx, fn := context.WithCancelCause(context.Background())
	s := &Simulation{
		ID:        id,
		Scenario:  scn,
		Scheduler: scheduler.New(),
		kubeMgr:   manager,
		errCh:     make(chan error),
		podMap:    make([]string, 0),
		resultDir: fmt.Sprintf("%s/%s/data", defaultResultDir, id),
		informer:  cache.NewStore(manager.Clientset(), logger.LevelInfo),
		stopCtx:   ctx,
		stopFn:    fn,
	}

	for _, e := range s.Scenario.Events {
		if e.Duration != 0 {
			s.podMap = append(s.podMap, e.Pod.Name)
		} else {
			logrus.Warnf("event %s has duration 0. Simulation ends before is evicted", e.Name)
		}
		ev := NewCreatePodEvent(e, s.kubeMgr, s.Scheduler)
		s.Scheduler.Schedule(ev)
	}
	return s
}

func (s *Simulation) GetResults() cache.Stats {
	return s.informer.GetStats()
}

func (s *Simulation) Start() error {
	err := os.MkdirAll(s.resultDir, os.ModePerm)
	if err != nil {
		return err
	}
	s.informer.Start()
	s.startTime = time.Now()
	logrus.Infof("starting simulation %s", s.ID)
	go s.startScheduler()
	go s.watchState()
	return s.wait()
}

func (s *Simulation) startScheduler() {
	err := s.Scheduler.Start()
	if err != nil {
		s.errCh <- err
	}
}

func (s *Simulation) wait() error {
	for {
		select {
		case <-s.stopCtx.Done():
			err := context.Cause(s.stopCtx)
			logrus.Infof("simulation %s stopping with cause: %s ", s.ID, err)
			s.informer.Stop()
			errScl := s.Scheduler.Stop()
			if errScl != nil {
				logrus.Errorf("error stopping scheduler: %v", err)
			}
			s.DumpResult()

			if errors.Is(err, context.Canceled) || errors.Is(err, ErrAllPodFinished) {
				return nil
			}
			return err
		case err := <-s.errCh:
			logrus.Errorf("error in simulation %s: %v", s.ID, err)
			s.stopFn(err)
			//return err
		}
	}
}

func (s *Simulation) Stop() {
	s.stopFn(nil)
}

func (s *Simulation) StopWithCause(err error) {
	s.stopFn(err)
}

// watchState watches the state of the pods in the cluster.
// it will stop the simulation when all pods with duration are finished.
// it will also update the metrics for the allocated resources in the nodes.
func (s *Simulation) watchState() {
	runningPods := make(map[string]bool)
	w, err := s.kubeMgr.Clientset().CoreV1().Pods("").Watch(context.Background(), metav1.ListOptions{})
	defer w.Stop()
	if err != nil {
		logrus.Errorf("error watching pods: %v", err)
		s.stopFn(err)
		return
	}

	for {
		select {
		case <-s.stopCtx.Done():
			logrus.Infoln("watcher stopping")
			return
		case e := <-w.ResultChan():
			nodeStatus := s.informer.GetNodesInfo()
			for _, ns := range nodeStatus {
				resources := ns.GetAllocated(corev1.ResourceCPU, corev1.ResourceMemory)
				cpu := resources[corev1.ResourceCPU]
				mem := resources[corev1.ResourceMemory]
				NodeAllocatedResourceMetric.Set(float64(cpu.MilliValue()), metric.WithLabels(ns.Node.Name, "cpu"))
				NodeAllocatedResourceMetric.Set(float64(mem.MilliValue()), metric.WithLabels(ns.Node.Name, "memory"))
			}
			if p, ok := e.Object.(*corev1.Pod); ok {
				logrus.Debugf("pod %s %s - status %s", p.Name, e.Type, p.Status.Phase)
				cpuReq := p.Spec.Containers[0].Resources.Requests.Cpu().String()
				memReq := p.Spec.Containers[0].Resources.Requests.Memory().String()
				EventTimelineMetric.Set(1, metric.WithLabels(p.Name, p.Spec.NodeName, string(p.Status.Phase), string(e.Type), cpuReq, memReq))
				switch e.Type {
				case watch.Added:
					runningPods[p.Name] = true
					logrus.Infof("pod %s created", p.Name)
				case watch.Deleted:
					logrus.Infof("pod %s deleted", p.Name)
					if s.isLastPodEvent(p) {
						err = s.kubeMgr.ResetPods()
						if err != nil {
							logrus.Errorf("error resetting remaining pods: %v", err)
						}
						nodeStatus = s.informer.GetNodesInfo()
						for _, ns := range nodeStatus {
							resources := ns.GetAllocated(corev1.ResourceCPU, corev1.ResourceMemory)
							cpu := resources[corev1.ResourceCPU]
							mem := resources[corev1.ResourceMemory]
							NodeAllocatedResourceMetric.Set(float64(cpu.MilliValue()), metric.WithLabels(ns.Node.Name, "cpu"))
							NodeAllocatedResourceMetric.Set(float64(mem.MilliValue()), metric.WithLabels(ns.Node.Name, "memory"))
						}
						s.StopWithCause(ErrAllPodFinished)
						return
					}
				}
			}
		}
	}
}

func (s *Simulation) isLastPodEvent(p *corev1.Pod) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, pod := range s.podMap {
		if pod == p.Name {
			s.podMap = append(s.podMap[:i], s.podMap[i+1:]...)
			break
		}
	}
	return len(s.podMap) == 0
}

func (s *Simulation) ResetCluster() error {
	err := s.kubeMgr.ResetPods()
	if err != nil {
		return err
	}
	err = s.kubeMgr.ResetNodes()
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return s.kubeMgr.ResetPods()
}

func (s *Simulation) MakeCluster() error {
	for _, n := range s.Scenario.Cluster.Nodes {
		err := s.kubeMgr.CreateNode(context.Background(), *s.kubeMgr.Clientset(), &n)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Simulation) DumpResult() {
	logrus.Infof("dumping results")
	if err := EventTimelineMetric.ExportCSV(s.resultDir, ""); err != nil {
		logrus.Errorf("error exporting csv: %v", err)
	}
	if err := NodeAllocatedResourceMetric.ExportCSV(s.resultDir, "_"); err != nil {
		logrus.Errorf("error exporting csv: %v", err)
	}
	stats := s.informer.GetStats()
	if err := stats.ExportCSV(s.resultDir); err != nil {
		logrus.Errorf("error exporting csv: %v", err)
	}
}
