package pkg

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
		TimestampFormat: "15:04:05",
	})
}

// Manager orchestrates events
type Manager struct {
	mu           sync.Mutex
	eventQueue   EventQueue
	eventCh      chan IEvent
	syncCh       chan IEvent
	newEventCond *sync.Cond
	stopCh       chan struct{}
	logger       *logrus.Logger
	kubeClient   *kubernetes.Clientset
}

// NewManager sets up the manager.
func NewManager(opts ...ManagerOption) (*Manager, error) {
	m := &Manager{}
	for _, opt := range opts {
		opt(m)
	}
	m.eventQueue = make(EventQueue, 0)
	m.eventCh = make(chan IEvent)
	m.syncCh = make(chan IEvent)
	m.newEventCond = sync.NewCond(&m.mu)
	m.stopCh = make(chan struct{})
	heap.Init(&m.eventQueue)
	return m, nil
}

func (m *Manager) Run() {
	logrus.Infof("Starting the manager at %s", time.Now().Format("15:04:05"))
	go m.waitForEvent()
	for {
		select {
		case e, ok := <-m.syncCh:
			if !ok {
				fmt.Println("Sync channel closed")
				return
			}
			// Process the sync event
			logrus.Infof("[%s] sync event at %s", e.OfType(), time.Now().Format("15:04:05"))
			m.EnqueueEvent(e)
		case e, ok := <-m.eventCh:
			if !ok {
				return
			}
			// Process the event
			logrus.Infof("[%s] handling at %s", e.OfType(), time.Now().Format("15:04:05"))
			e.Handle(m)
		case <-m.stopCh:
			logrus.Infof("Stopping the manager")
			return
		}
	}
}

func (m *Manager) EnqueueEvent(e IEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	heap.Push(&m.eventQueue, e)
	m.newEventCond.Signal()
}

func (m *Manager) Stop() {
	close(m.stopCh)
	m.mu.Lock()
	m.newEventCond.Broadcast()
	m.mu.Unlock()
}

func (m *Manager) waitForEvent() {
	defer close(m.eventCh)

	for {
		m.mu.Lock()
		for m.eventQueue.Empty() {
			select {
			case <-m.stopCh:
				m.mu.Unlock()
				return
			default:
				m.newEventCond.Wait()
			}
		}
		event := m.eventQueue.HappenNow()
		m.mu.Unlock()
		if event != nil {
			select {
			case <-m.stopCh:
				return
			case m.eventCh <- event:
			}
		}
	}
}
