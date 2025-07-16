package simulation

import (
	"context"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/cache"
	kube "github.com/maczg/kube-event-generator/pkg/kubernetes"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

type Simulation interface {
	GetID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetStats() *cache.Stats
}

type simulation struct {
	ID               string
	logger           *logger.Logger
	startTime        time.Time
	scenario         *Scenario
	scheduler        scheduler.Scheduler
	clientset        *kubernetes.Clientset
	schedulerManager kube.SchedulerManager
	cache            *cache.Store

	mu      sync.Mutex
	running bool
	errCh   chan error
	stopCh  chan struct{}
	podMap  []string
}

func NewSimulation(scn *Scenario, clientset *kubernetes.Clientset, sm kube.SchedulerManager, logger *logger.Logger) Simulation {
	scdl := scheduler.New(logger)
	sim := &simulation{
		ID:               fmt.Sprintf("sim-%s-%s", scn.Metadata.Name, time.Now().Format("15_04_05_020106")),
		logger:           logger,
		startTime:        time.Now(),
		scenario:         scn,
		scheduler:        scdl,
		clientset:        clientset,
		schedulerManager: sm,
		cache:            cache.NewStore(clientset),
		stopCh:           make(chan struct{}),
		errCh:            make(chan error, 1),
		podMap:           make([]string, 0),
	}
	return sim
}

func (s *simulation) Start(ctx context.Context) error {
	s.logger.Infoln("starting simulation")
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("simulation is already running")
	}
	s.running = true
	s.startTime = time.Now()
	s.mu.Unlock()

	if err := s.loadEvents(); err != nil {
		s.logger.Errorln("failed to load events:", err)
		return err
	}

	s.initialize(ctx)
	defer s.finalize(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Infoln("context done, stopping simulation")
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return nil
		case err := <-s.errCh:
			s.logger.Errorf("error in simulation: %v", err)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return err
		case <-s.stopCh:
			s.logger.Infoln("stop signal received, stopping simulation")
			s.mu.Lock()
			if !s.running {
				s.logger.Warnln("simulation is not running, nothing to stop")
				s.mu.Unlock()
				return nil
			}
			s.running = false
			s.mu.Unlock()
			return nil
		}
	}
}

func (s *simulation) Stop(ctx context.Context) error {
	<-s.stopCh
	return nil
}

func (s *simulation) GetID() string {
	return s.ID
}

func (s *simulation) GetStats() *cache.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cache == nil {
		return nil
	}
	stats := s.cache.GetStats()
	return &stats
}

func (s *simulation) loadEvents() error {
	if s.scenario == nil {
		err := fmt.Errorf("simulation %s has no events", s.ID)
		s.logger.Errorln(err)
		return err
	}

	s.logger.Debugf("loading events from %s", s.scenario.Metadata.Name)

	if s.scenario.Events.Pods != nil {
		for _, event := range s.scenario.Events.Pods {
			event.SetClientset(s.clientset)
			if err := s.scheduler.Schedule(&event); err != nil {
				s.logger.Errorln(err)
				return err
			}
			if event.Eviction() != 0 {
				s.podMap = append(s.podMap, event.PodSpec.Name)
			} else {
				s.logger.Warnf("event %s has duration 0. Simulation ends before it is evicted", event.PodSpec.Name)
			}
		}
	}

	if s.scenario.Events.Scheduler != nil {
		for _, event := range s.scenario.Events.Scheduler {
			if err := s.scheduler.Schedule(&event); err != nil {
				s.logger.Errorln(err)
				return err
			}
		}
	}
	s.logger.Infof("loaded %d pod events and %d scheduler events", len(s.scenario.Events.Pods), len(s.scenario.Events.Scheduler))
	return nil
}

func (s *simulation) initialize(ctx context.Context) {
	go s.startCache()
	go s.startScheduler(ctx)
	go s.podWatcher(ctx)
}

func (s *simulation) podWatcher(ctx context.Context) {
	watcher, err := s.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Errorf("failed to watch pods: %v", err)
		s.errCh <- err
		return
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Infoln("context done, stopping pod watcher")
			return
		case <-s.errCh:
			s.logger.Infoln("error channel closed, stopping pod watcher")
			return
		case event := <-watcher.ResultChan():
			if pod, ok := event.Object.(*v1.Pod); ok {
				switch event.Type {
				case watch.Deleted:
					if s.isLastPodEvent(pod) {
						s.logger.Info("last pod deleted, stopping simulation")
						s.stopCh <- struct{}{}
						return
					}
					s.logger.Debugf("pod %s deleted", pod.Name)
				}
			}
		}
	}
}

func (s *simulation) isLastPodEvent(pod *v1.Pod) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existingPod := range s.podMap {
		if existingPod == pod.Name {
			s.podMap = append(s.podMap[:i], s.podMap[i+1:]...)
			break
		}
	}

	return len(s.podMap) == 0
}

func (s *simulation) startCache() {
	s.logger.Infoln("starting cache")
	s.cache.Start()
}

func (s *simulation) startScheduler(ctx context.Context) {
	err := s.scheduler.Start(ctx)
	if err != nil {
		s.logger.Errorf("failed to start scheduler: %v", err)
		s.errCh <- err
	}
}

func (s *simulation) finalize(ctx context.Context) {
	s.logger.Infoln("finalizing simulation")
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.scheduler.Stop()
	if err != nil {
		s.logger.Errorf("failed to stop scheduler: %v", err)
	}
	s.cache.Stop()
	s.logger.Infof("simulation %s finalized at %v", s.ID, time.Now())
	ctx.Done()
}
