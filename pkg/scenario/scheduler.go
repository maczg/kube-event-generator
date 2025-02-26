package scenario

import (
	"container/heap"
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
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
	mu    sync.Mutex
	Queue *EventQueue
	// startTime is the time when the scheduler started.
	startTime  time.Time
	startCh    chan struct{}
	stopCh     chan struct{}
	stopped    bool
	cond       *sync.Cond
	KubeClient *kubernetes.Clientset
}

func NewScheduler(opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		Queue:   &EventQueue{},
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
	s.cond = sync.NewCond(&s.mu)
	heap.Init(s.Queue)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopCh)
	close(s.startCh)
}

// Run drives the entire scenario. It processes each event in chronological order.
func (s *Scheduler) Run() {
	_now := time.Now()
	nextMinute := _now.Truncate(time.Minute).Add(time.Minute)
	fmt.Printf("Next minute: %v\n", nextMinute)
	time.Sleep(time.Until(nextMinute))

	s.startTime = time.Now()
	s.startCh <- struct{}{}
	logrus.Infof("scheduler started")

	for {
		select {
		case <-s.stopCh:
			logrus.Infof("scheduler stopped")
			return
		default:
			s.mu.Lock()

			if s.Queue.Len() == 0 {
				time.Sleep(100 * time.Millisecond)
				s.mu.Unlock()
				continue
			}

			if s.Queue.Len() > 0 {
				firstEvent := (*s.Queue)[0]
				now := time.Since(s.startTime)
				if firstEvent.RunAfter > now {
					s.mu.Unlock()
					continue
				}
				evt := heap.Pop(s.Queue).(*Event)
				s.mu.Unlock()
				err := evt.Execute(s)
				if err != nil {
					logrus.Errorf("Error executing event: %v", err)
				}
			}
		}
	}
}

// AddEvent safely pushes new events (e.g. evictions) into the queue at runtime.
func (s *Scheduler) AddEvent(evt *Event) {
	s.mu.Lock()
	heap.Push(s.Queue, evt)
	s.cond.Signal() // wake up the scheduler loop
	s.mu.Unlock()
}

func (s *Scheduler) Enqueue(e []*Event) {
	s.mu.Lock()
	for _, evt := range e {
		heap.Push(s.Queue, evt)
	}
	s.cond.Signal()
	s.mu.Unlock()
}
