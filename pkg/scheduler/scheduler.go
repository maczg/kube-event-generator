package scheduler

import (
	"container/heap"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Scheduler interface {
	// Start starts scheduling queue. It blocks until the scheduler is stopped.
	Start(ctx context.Context) error
	// Stop stops scheduling queue. It returns an error if the scheduler is not running.
	Stop() error
	Schedule(e Schedulable)
	StartedAt() time.Time
	GetEvents() []Schedulable
}

type scheduler struct {
	mu        sync.Mutex
	queue     *Queue[Schedulable]
	startTime time.Time
	running   bool
	stopCh    chan struct{}
}

func (s *scheduler) GetEvents() []Schedulable {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.queue.items
}

func (s *scheduler) Start(ctx context.Context) error {
	logrus.Info("scheduler started")
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.running = true
	s.startTime = time.Now()
	s.stopCh = make(chan struct{})
	s.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			logrus.Debug("ctx Done, scheduler stopped")
			cause := ctx.Err()
			if cause != nil && !errors.Is(cause, context.Canceled) {
				logrus.Errorf("scheduler stopped due to: %v", cause)
			}
			return nil
		case <-s.stopCh:
			logrus.Info("scheduler stopped")
			return nil
		default:
			s.mu.Lock()
			if s.queue.Len() == 0 {
				s.mu.Unlock()
				continue
			}
			if s.startTime.Add((*s.queue.Peek()).ExecuteAfterDuration()).After(time.Now()) {
				s.mu.Unlock()
				continue
			}
			e := heap.Pop(s.queue).(Schedulable)
			s.mu.Unlock()
			go func() {
				if err := e.Execute(ctx); err != nil {
					logrus.Errorf("failed to execute event %s: %v", e.ID(), err)
				}
			}()
		}
	}
}

func (s *scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return ErrNotRunning
	}
	close(s.stopCh)
	return nil
}

func (s *scheduler) Schedule(e Schedulable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue.Push(e)
}

func (s *scheduler) StartedAt() time.Time {
	return s.startTime
}

func New() Scheduler {
	s := scheduler{
		queue:  NewQueue[Schedulable](),
		stopCh: make(chan struct{}),
	}
	return &s
}
