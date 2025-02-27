package scenario

import (
	"container/heap"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	cond       *sync.Cond
	KubeClient *kubernetes.Clientset
	Metrics    *MetricCollector
}

func NewScheduler(opts ...SchedulerOption) *Scheduler {

	mc := NewMetricCollector()

	s := &Scheduler{
		Queue:   &EventQueue{},
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
		Metrics: mc,
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
	go s.RecordPendingPodQueue()

	for {
		select {
		case <-s.stopCh:
			logrus.Infof("scheduler stopped")
			err1 := s.Metrics.PlotPendingDuration()
			err2 := s.Metrics.PlotPendingPodsOverTime()
			if err1 != nil || err2 != nil {
				logrus.Errorf("Error plotting metrics: %v, %v", err1, err2)
			}
			return
		default:
			s.mu.Lock()

			if s.Queue.Len() == 0 {
				time.Sleep(50 * time.Millisecond)
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

func (s *Scheduler) LogPodPendingQueueLength() {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.KubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("Error listing pods: %v", err)
		return
	}
	pending := 0
	for _, pod := range p.Items {
		if pod.Status.Phase == "Pending" {
			pending++
		}
	}
	s.Metrics.RecordPendingQueueLength(pending, time.Now())
}

func (s *Scheduler) RecordPendingPodQueue() {
	for {
		select {
		case <-s.stopCh:
			return
		default:
			s.LogPodPendingQueueLength()
			time.Sleep(1 * time.Second)
		}
	}
}
