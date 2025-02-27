package scheduler

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"time"
)

type SchedulerOpts func(s *Scheduler)

func WithQueue(q *EventQueue) SchedulerOpts {
	return func(s *Scheduler) {
		s.queue = q
	}
}

func WithKubeManager(km *KubeManager) SchedulerOpts {
	return func(s *Scheduler) {
		s.KubeManager = km
	}
}

func WithDeadline(seconds int) SchedulerOpts {
	return func(s *Scheduler) {
		time.AfterFunc(time.Duration(seconds)*time.Second, func() {
			s.Stop()
		})
	}
}

func WithScenario(sc scenario.Scenario) SchedulerOpts {
	return func(s *Scheduler) {
		for _, e := range sc.Events {
			event := NewCreatePodEvent(e.Pod, e.After, e.Duration)
			s.AddEvent(event)
		}
	}
}
