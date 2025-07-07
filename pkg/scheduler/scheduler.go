package scheduler

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
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
	startTime time.Time
	queue     *Queue[Schedulable]
	stopCh    chan struct{}
	log       *logger.Logger
	mu        sync.Mutex
	running   bool
}

func (s *scheduler) GetEvents() []Schedulable {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queue.items
}

func (s *scheduler) Start(ctx context.Context) error {
	s.log.Info("scheduler started")
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}

	s.running = true
	s.startTime = time.Now()
	s.stopCh = make(chan struct{})
	stopCh := s.stopCh
	s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			s.log.Debug("ctx Done, scheduler stopped")

			cause := ctx.Err()
			if cause != nil && !errors.Is(cause, context.Canceled) {
				s.log.Errorf("scheduler stopped due to: %v", cause)
			}

			return nil
		case <-stopCh:
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			s.log.Info("scheduler stopped")

			return nil
		default:
			s.mu.Lock()
			if s.queue.Len() == 0 {
				s.mu.Unlock()
				time.Sleep(10 * time.Millisecond)

				continue
			}

			if s.startTime.Add((*s.queue.Peek()).ExecuteAfterDuration()).After(time.Now()) {
				s.mu.Unlock()
				time.Sleep(10 * time.Millisecond)

				continue
			}

			e := heap.Pop(s.queue).(Schedulable)
			s.mu.Unlock()

			go func() {
				eventCtx := logger.WithEventID(ctx, e.ID())
				if err := e.Execute(eventCtx); err != nil {
					s.log.WithContext(eventCtx).WithFields(map[string]interface{}{
						"event_id": e.ID(),
					}).Errorf("failed to execute event: %v", err)
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

	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}

	s.running = false

	return nil
}

func (s *scheduler) Schedule(e Schedulable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue.Push(e)
	s.log.WithFields(map[string]interface{}{
		"event_id":      e.ID(),
		"execute_after": e.ExecuteAfterDuration(),
	}).Debug("event scheduled")
}

func (s *scheduler) StartedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startTime
}

func New() Scheduler {
	s := &scheduler{
		queue: NewQueue[Schedulable](),
		log:   logger.Default(),
	}

	return s
}
