package scheduler

import (
	"container/heap"
	"context"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
}

type Scheduler struct {
	mu          sync.Mutex
	queue       *EventQueue
	KubeManager *KubeManager
	startTime   time.Time
	stopCh      chan struct{}
	eventCh     chan Event
}

func NewScheduler(opts ...SchedulerOpts) *Scheduler {
	q := NewEventQueue()
	s := &Scheduler{
		queue:   q,
		stopCh:  make(chan struct{}),
		eventCh: make(chan Event),
	}

	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Scheduler) Run() error {
	logrus.Infof("scheduler started")
	s.startTime = time.Now()
	go s.loop()

	for {
		select {
		case <-s.stopCh:
			logrus.Infof("scheduler stopped")
			return nil
		case e, ok := <-s.eventCh:
			if !ok {
				return nil
			}
			ctx := context.WithValue(context.Background(), SchedulerCtxKey, s)
			err := e.Execute(ctx)
			if err != nil {
				logrus.Errorf("error executing event: %v", err)
			}
		}
	}
}

func (s *Scheduler) loop() {
	for {
		s.mu.Lock()
		if s.queue.Len() == 0 {
			s.mu.Unlock()
			time.Sleep(100 * time.Microsecond)
			continue
		}
		if s.startTime.Add(s.queue.Peek().At()).After(time.Now()) {
			s.mu.Unlock()
			continue
		}
		e := heap.Pop(s.queue).(Event)
		s.mu.Unlock()
		s.eventCh <- e
	}
}

func (s *Scheduler) AddEvent(e Event) {
	s.mu.Lock()
	heap.Push(s.queue, e)
	s.mu.Unlock()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) StartTime() time.Time {
	return s.startTime
}
