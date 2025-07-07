// Package simulator provides functionality to simulate Kubernetes scenarios.
package simulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/maczg/kube-event-generator/pkg/cache"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
)

// Simulation represents a simulation of a scenario in Kubernetes.
type Simulation struct {
	startTime time.Time
	scheduler scheduler.Scheduler
	scenario  *scenario.Scenario
	clientset *kubernetes.Clientset
	informer  *cache.Store
	errCh     chan error
	stopCh    chan struct{}
	log       *logger.Logger
	name      string
	podMap    []string
	mu        sync.Mutex
}

// NewSimulation creates a new Simulation instance.
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
		log:       logger.Default(),
	}
}

func (s *Simulation) LoadEvents() {
	s.log.WithFields(map[string]interface{}{
		"simulation": s.name,
		"scenario":   s.scenario.Metadata.Name,
	}).Info("Loading events")

	for _, event := range s.scenario.Events.Pods {
		s.scheduler.Schedule(event)

		if event.ExecuteForDuration() != 0 {
			s.podMap = append(s.podMap, event.Pod.Name)
		} else {
			s.log.WithFields(map[string]interface{}{
				"event": event.Pod.Name,
			}).Warn("event has duration 0. Simulation ends before is evicted")
		}
	}

	if s.scenario.Events.SchedulerConfigs != nil {
		for _, event := range s.scenario.Events.SchedulerConfigs {
			s.scheduler.Schedule(event)
		}
	}

	s.log.WithFields(map[string]interface{}{
		"pod_events":       len(s.scenario.Events.Pods),
		"scheduler_events": len(s.scenario.Events.SchedulerConfigs),
	}).Info("Events loaded successfully")
}

func (s *Simulation) Start(ctx context.Context) error {
	// Add simulation ID to context
	ctx = logger.WithSimulationID(ctx, s.name)

	s.startTime = time.Now()
	defer s.finalize()

	s.log.WithContext(ctx).Info("Starting simulation")

	go s.startScheduler(ctx)
	go s.startInformer()

	watcher, err := s.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		s.log.WithContext(ctx).Errorf("failed to watch pods: %v", err)
		s.errCh <- err
	}

	for {
		select {
		case <-ctx.Done():
			s.log.WithContext(ctx).Debug("ctx Done, simulation stopped")

			cause := context.Cause(ctx)
			if cause != nil && !errors.Is(cause, context.Canceled) {
				s.log.WithContext(ctx).Errorf("simulation stopped due to: %v", cause)
			}

			return cause
		case event := <-watcher.ResultChan():
			if pod, ok := event.Object.(*v1.Pod); ok {
				switch event.Type {
				case watch.Deleted:
					if s.isLastPodEvent(pod) {
						s.log.WithContext(ctx).WithFields(map[string]interface{}{
							"pod": pod.Name,
						}).Info("last pod deleted, stopping simulation")

						return nil
					}

					s.log.WithContext(ctx).WithFields(map[string]interface{}{"pod": pod.Name}).Info("pod deleted")
				}
			}
		case err = <-s.errCh:
			if err != nil {
				s.log.WithContext(ctx).Errorf("error in simulation: %v", err)
				return err
			}
		case <-s.stopCh:
			s.log.WithContext(ctx).Info("simulation stopped")
			return nil
		}
	}
}

// Stop stops the simulation and cleans up resources.
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
		s.log.WithContext(ctx).Errorf("failed to start scheduler: %v", err)
		s.errCh <- err
	}
}

func (s *Simulation) finalize() {
	// stop the scheduler
	if err := s.scheduler.Stop(); err != nil {
		s.log.Errorf("failed to stop scheduler: %v", err)
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
