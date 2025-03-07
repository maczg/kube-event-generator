package scheduler

import (
	"container/heap"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	ErrAlreadyRunning = errors.New("scheduler already running")
	ErrNotRunning     = errors.New("scheduler not running")
)

type Scheduler interface {
	// Start starts scheduling queue
	Start() error
	// Stop stops scheduling queue
	Stop() error
	// Schedule an Schedulable on the scheduler queue
	Schedule(e Schedulable)
	// StartedAt returns the time when the scheduler started
	StartedAt() time.Time
}

type scheduler struct {
	mu        sync.Mutex
	queue     Queue[Schedulable]
	startTime time.Time
	cancelCtx context.Context
	cancelFn  context.CancelFunc
	running   bool
	stopCh    chan struct{}
	// eventCh is used to receive Schedulable popped from the heap
	eventCh chan Schedulable
}

func New() Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := scheduler{
		queue:     *NewQueue[Schedulable](SchedulableCmpFn),
		cancelCtx: ctx,
		cancelFn:  cancel,
		running:   false,
		stopCh:    make(chan struct{}),
		eventCh:   make(chan Schedulable),
	}
	return &s
}

func (s *scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.running = true
	s.mu.Unlock()
	s.startTime = time.Now()
	logrus.Infoln("starting scheduler")

	go s.run()

	for {
		select {
		case <-s.cancelCtx.Done():
			return nil
		case e := <-s.eventCh:
			go s.processEvent(e)
		}
	}
}

func (s *scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrNotRunning
	}
	s.cancelFn()
	s.running = false
	s.mu.Unlock()
	return nil
}

func (s *scheduler) run() {
	for {
		select {
		case <-s.cancelCtx.Done():
			logrus.Infoln("ctx Done, stopping")
			return
		default:
			s.mu.Lock()
			if s.queue.Len() == 0 {
				s.mu.Unlock()
				continue
			}
			if s.startTime.Add(s.queue.Peek().After()).After(time.Now()) {
				s.mu.Unlock()
				continue
			}
			e := heap.Pop(&s.queue).(Schedulable)
			s.mu.Unlock()
			s.eventCh <- e
		}
	}
}

func (s *scheduler) processEvent(e Schedulable) {
	err := e.Run(s.cancelCtx)
	if err != nil {
		logrus.Errorf("event [%s] return error: %v", e.ID(), err)
	}
	logrus.Debugf("event [%s] finished", e.ID())
}

func (s *scheduler) Schedule(e Schedulable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue.Add(e)
}

func (s *scheduler) StartedAt() time.Time { return s.startTime }
