package simulator

import (
	"context"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/cache"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

type Simulation struct {
	name      string
	scenario  *scenario.Scenario
	mu        sync.Mutex
	podMap    []string
	scheduler scheduler.Scheduler
	clientset *kubernetes.Clientset
	informer  *cache.Store
	startTime time.Time
	errCh     chan error
	stopCh    chan struct{}
}

func NewSimulation(scn *scenario.Scenario, clientset *kubernetes.Clientset, scdl scheduler.Scheduler) *Simulation {
	return &Simulation{
		name:      fmt.Sprintf("run-%s-%s", scn.Metadata.Name, time.Now().Format("15_04_05_020106")),
		clientset: clientset,
		scheduler: scdl,
		scenario:  scn,
		informer:  cache.NewStore(clientset),
		podMap:    make([]string, 0),
		errCh:     make(chan error),
		stopCh:    make(chan struct{}),
	}
}

func (s *Simulation) LoadEvents() {
	for _, event := range s.scenario.Events.Pods {
		s.scheduler.Schedule(event)
		if event.ExecuteForDuration() != 0 {
			s.podMap = append(s.podMap, event.Pod.Name)
		} else {
			logrus.Warnf("event %s has duration 0. Simulation ends before is evicted", event.Pod.Name)
		}
	}
}

func (s *Simulation) Start(ctx context.Context) error {
	s.startTime = time.Now()
	defer s.finalize()

	go s.startScheduler(ctx)
	go s.startInformer()

	watcher, err := s.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("failed to watch pods: %v", err)
		s.errCh <- err
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Debug("ctx Done, simulation stopped")
			cause := context.Cause(ctx)
			if cause != nil && !errors.Is(cause, context.Canceled) {
				logrus.Errorf("simulation stopped due to: %v", cause)
			}
			return cause
		case event := <-watcher.ResultChan():
			if pod, ok := event.Object.(*v1.Pod); ok {
				switch event.Type {
				case watch.Deleted:
					if s.isLastPodEvent(pod) {
						logrus.Infof("last pod %s deleted, stopping simulation", pod.Name)
						return nil
					} else {
						logrus.Infof("pod %s deleted", pod.Name)
					}
				}
			}
		case err = <-s.errCh:
			if err != nil {
				logrus.Errorf("error in simulation: %v", err)
				return err
			}
		case <-s.stopCh:
			logrus.Info("simulation stopped")
			return nil
		}
	}
}

func (s *Simulation) Stop() error {
	s.stopCh <- struct{}{}
	return nil
}

func (s *Simulation) startInformer() {
	s.informer.Start()
}

func (s *Simulation) startScheduler(ctx context.Context) {
	err := s.scheduler.Start(ctx)
	if err != nil {
		logrus.Errorf("failed to start scheduler: %v", err)
		s.errCh <- err
	}
}

func (s *Simulation) finalize() {
	// stop the scheduler
	if err := s.scheduler.Stop(); err != nil {
		logrus.Errorf("failed to stop scheduler: %v", err)
	}
	// stop the informer
	s.informer.Stop()
}

func (s *Simulation) GetStats() cache.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.informer.GetStats()
}

func (s *Simulation) isLastPodEvent(p *v1.Pod) bool {
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
