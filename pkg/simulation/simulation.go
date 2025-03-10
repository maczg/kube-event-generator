package simulation

import (
	"context"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sync"
	"time"
)

type Simulation struct {
	// ID of the simulation
	ID string
	// Scenario
	Scenario *scenario.Scenario
	//Scheduler to schedule events
	Scheduler scheduler.Scheduler
	// kubeMgr is the kubernetes manager
	kubeMgr      *utils.KubernetesManager
	nodeResource NodeResource

	// podMap is used to keep track of the pods that have eviction time set.
	// When the len of the map is zero, no pods have to be scheduled for eviction.
	mu      sync.Mutex
	podMap  []string
	errCh   chan error
	stopCtx context.Context
	stopFn  context.CancelCauseFunc
}

func New(scn *scenario.Scenario, manager *utils.KubernetesManager) *Simulation {
	ctx, fn := context.WithCancelCause(context.Background())
	s := &Simulation{
		ID:           fmt.Sprintf("sim-%s-%s", time.Now().Format("15-04-05"), scn.Name),
		Scenario:     scn,
		Scheduler:    scheduler.New(),
		kubeMgr:      manager,
		nodeResource: NewNodeResource(),
		errCh:        make(chan error),
		podMap:       make([]string, 0),
		stopCtx:      ctx,
		stopFn:       fn,
	}
	for _, e := range s.Scenario.Events {
		if e.Duration != 0 {
			s.podMap = append(s.podMap, e.Pod.Name)
		}
		ev := NewSimEvent(e, s.kubeMgr, s.Scheduler)
		s.Scheduler.Schedule(ev)
	}

	for _, n := range s.Scenario.Cluster.Nodes {
		s.nodeResource.AddNode(n.Name)
	}

	return s
}

func (s *Simulation) Start() error {
	logrus.Info("Resetting cluster")
	err := s.ResetCluster()
	if err != nil {
		s.stopFn(err)
	}
	time.Sleep(5 * time.Second)

	logrus.Info("Creating cluster")
	err = s.CreateCluster()
	if err != nil {
		s.stopFn(err)
	}

	time.Sleep(5 * time.Second)

	logrus.Infof("starting simulation %s", s.ID)

	// init
	for _, n := range s.Scenario.Cluster.Nodes {
		s.nodeResource.AddNode(n.Name)
		s.nodeResource.AddResource(n.Name, "cpu", n.Status.Capacity.Cpu().AsApproximateFloat64())
		s.nodeResource.AddResource(n.Name, "memory", n.Status.Capacity.Memory().AsApproximateFloat64())
	}

	go func() {
		err = s.Scheduler.Start()
		if err != nil {
			s.errCh <- err
		}
	}()
	go s.watchState()
	go s.allPodAreScheduled()

	for {
		select {
		case <-s.stopCtx.Done():
			logrus.Infof("simulation %s stopping. Cause: %s ", s.ID, s.stopCtx.Err())
			s.nodeResource.Status()
			if errors.Is(s.stopCtx.Err(), context.Canceled) || errors.Is(s.stopCtx.Err(), errors.New("all pods are scheduled")) {
				return nil
			}
			return s.stopCtx.Err()
		case err := <-s.errCh:
			logrus.Errorf("error in simulation %s: %v", s.ID, err)
			s.stopFn(err)
			return err
		}
	}
}

func (s *Simulation) Stop() {
	s.stopFn(nil)
}

// allPodAreScheduled checks if the inner pods in queue are all finished.
// this ensure that all pod with duration have done
func (s *Simulation) allPodAreScheduled() {
	for {
		select {
		case <-s.stopCtx.Done():
			return
		default:
			s.mu.Lock()
			if len(s.podMap) == 0 {
				s.mu.Unlock()
				s.stopFn(errors.New("all pods are scheduled"))
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
		logrus.Errorf("error watching pods: %v", err)
		s.stopFn(err)
		return
	}
	runningPod := make(map[string]bool)

	for {
		select {
		case <-s.stopCtx.Done():
			logrus.Infof("watcher %s stopping", s.ID)
			return
		case e := <-w.ResultChan():
			p, ok := e.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			//logrus.Infof("[watcher] - pod %s %s status %s", e.Type, p.Name, p.Status.Phase)
			switch {
			case e.Type == "ADDED":
				// is in pending state. Should track pending start
				logrus.Infof("[watcher] - pod %s -  %s  -  status %s", e.Type, p.Name, p.Status.Phase)
			case e.Type == watch.Modified:
				if p.Status.Phase == corev1.PodRunning && p.ObjectMeta.DeletionTimestamp == nil {
					if _, exists := runningPod[p.Name]; !exists {
						// first transition to running state
						logrus.Infof("[watcher] - pod %s -  %s  -  status %s", e.Type, p.Name, p.Status.Phase)
						runningPod[p.Name] = true
						// here we know where the pod is running (node name) and we can measure the node status
						nodeName := p.Spec.NodeName
						s.mu.Lock()
						cpu := p.Spec.Containers[0].Resources.Requests.Cpu().AsApproximateFloat64()
						mem := p.Spec.Containers[0].Resources.Requests.Memory().AsApproximateFloat64()
						lastNodeCpu := s.nodeResource.LastRecord(nodeName, "cpu")
						lastNodeMem := s.nodeResource.LastRecord(nodeName, "memory")
						residualCpu := lastNodeCpu.Value - cpu
						residualMem := lastNodeMem.Value - mem
						s.nodeResource.AddResource(nodeName, "cpu", residualCpu)
						s.nodeResource.AddResource(nodeName, "memory", residualMem)
						s.mu.Unlock()
					}
				}
			case e.Type == watch.Deleted:
				// pod is deleted
				logrus.Infof("[watcher] - pod %s -  %s  -  status %s, was in %s", e.Type, p.Name, p.Status.Phase, p.Spec.NodeName)
				// release the resources
				s.mu.Lock()
				nodeName := p.Spec.NodeName
				cpuLast := s.nodeResource.LastRecord(nodeName, "cpu")
				memLast := s.nodeResource.LastRecord(nodeName, "memory")
				cpu := p.Spec.Containers[0].Resources.Requests.Cpu().AsApproximateFloat64()
				mem := p.Spec.Containers[0].Resources.Requests.Memory().AsApproximateFloat64()
				residualCpu := cpuLast.Value + cpu
				residualMem := memLast.Value + mem
				s.nodeResource.AddResource(nodeName, "cpu", residualCpu)
				s.nodeResource.AddResource(nodeName, "memory", residualMem)
				//s.mu.Unlock()
				//
				//s.mu.Lock()
				for i, pod := range s.podMap {
					if pod == p.Name {
						logrus.Infof("pod %s is done", p.Name)
						s.podMap = append(s.podMap[:i], s.podMap[i+1:]...)
						break
					}
				}
				logrus.Infof("queue len %d", len(s.podMap))
				s.mu.Unlock()
				//default:
				//	logrus.Infof("[watcher] - pod %s -  %s  -  status %s", e.Type, p.Name, p.Status.Phase)
			}

			//logrus.Infof("[watcher] - pod %s %s status %s", e.Type, p.Name, p.Status.Phase)
			//if p.Status.Phase == corev1.PodRunning && p.ObjectMeta.DeletionTimestamp != nil {
			//s.mu.Lock()
			//for i, pod := range s.podMap {
			//	if pod == p.Name {
			//		logrus.Infof("pod %s is done", p.Name)
			//		s.podMap = append(s.podMap[:i], s.podMap[i+1:]...)
			//		break
			//	}
			//}
			//logrus.Infof("queue len %d", len(s.podMap))
			//s.mu.Unlock()
			//}
		}
	}
}

func (s *Simulation) ResetCluster() error {
	err := s.kubeMgr.ResetNodes()
	if err != nil {
		return err
	}
	return s.kubeMgr.ResetPods()
}

func (s *Simulation) CreateCluster() error {
	for _, n := range s.Scenario.Cluster.Nodes {
		err := s.kubeMgr.CreateNode(context.Background(), *s.kubeMgr.Clientset(), &n)
		if err != nil {
			return err
		}
	}
	return nil
}
