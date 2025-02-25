package pkg

import (
	"container/heap"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

// Manager orchestrates events and the KubeClient interactions.
type Manager struct {
	kubeClient *kubernetes.Clientset
	// A priority queue of events that we pop in chronological order.
	eventQueue EventQueue
	// Protects eventQueue for concurrent access
	mu sync.Mutex
	// A signal to stop the Run loop
	StopCh chan struct{}

	podMetrics     map[string]*PodMetrics // Key: "namespace/name"
	pqObservations []PendingQueueObservation
	metricsMu      sync.Mutex // Protects podMetrics and pqObservations
}

// NewManager sets up the manager.
func NewManager(opts ...ManagerOption) (*Manager, error) {
	m := &Manager{}
	for _, opt := range opts {
		opt(m)
	}
	m.eventQueue = make(EventQueue, 0)
	m.StopCh = make(chan struct{})
	m.podMetrics = make(map[string]*PodMetrics)
	heap.Init(&m.eventQueue)
	return m, nil
}

// Run starts an infinite loop that processes events in chronological order.
// Call Run in a goroutine, or in main(), depending on your architecture.
func (m *Manager) Run() {
	logrus.Infof("Manager started at %s ", time.Now().Format("15:04:05"))
	for {
		m.mu.Lock()
		if len(m.eventQueue) == 0 {
			m.mu.Unlock()
			// Sleep briefly, then check again
			select {
			case <-time.After(1 * time.Second):
				// continue loop
			case <-m.StopCh:
				return
			}
			continue
		}
		// Look at the next event
		nextEvent := m.eventQueue[0]
		now := time.Now()
		waitDuration := nextEvent.At.Sub(now)
		if waitDuration <= 0 {
			// It's time for this event, pop and process
			heap.Pop(&m.eventQueue)
			m.mu.Unlock()
			// Process
			m.processEvent(nextEvent)
			continue
		} else {
			// Not time yet, unlock and sleep for the difference
			// TODO may want to return to the top of the loop if a new event is added
			m.mu.Unlock()
			select {
			case <-time.After(waitDuration):
				// after sleeping, loop again
			case <-m.StopCh:
				err := m.Cleanup(true)
				m.DumpMetrics()
				if err != nil {
					return
				}
				return
			}
		}
	}
}

// Stop signals the manager to exit the Run loop
func (m *Manager) Stop() {
	close(m.StopCh)
}

// processEvent routes the event to the right handler
func (m *Manager) processEvent(e *Event) {
	switch e.Type {
	case Submit:
		m.handleSubmit(e)
	case Evict:
		m.handleEvict(e)
	}
}

// handleSubmit creates the pod in Kubernetes, then launches a watcher to see when it runs.
func (m *Manager) handleSubmit(e *Event) {
	logrus.Infof("Submitting pod %s at %s", e.Pod.Name, time.Now().Format("15:04:05"))

	key := fmt.Sprintf("%s-%s", e.Pod.Namespace, e.Pod.Name)
	m.metricsMu.Lock()
	pm := &PodMetrics{
		PodNamespace: e.Pod.Namespace,
		PodName:      e.Pod.Name,
		CreationTime: time.Now(),
	}
	m.podMetrics[key] = pm
	m.metricsMu.Unlock()

	_, err := m.kubeClient.CoreV1().Pods(e.Pod.Namespace).Create(
		context.TODO(),
		e.Pod,
		metav1.CreateOptions{},
	)
	if err != nil {
		logrus.Errorf("Error creating pod %s: %v", e.Pod.Name, err)
		return
	}

	// Watch for the pod to become running, then schedule its eviction
	go m.waitForPodRunning(e.Pod, e.Duration)
}

// handleEvict removes the pod from Kubernetes
func (m *Manager) handleEvict(e *Event) {
	logrus.Warnf("Evicting pod %s at %s", e.Pod.Name, time.Now().Format("15:04:05"))
	err := m.kubeClient.CoreV1().Pods(e.Pod.Namespace).Delete(
		context.TODO(),
		e.Pod.Name,
		metav1.DeleteOptions{},
	)
	if err != nil {
		logrus.Errorf("Error evicting pod %s: %v", e.Pod.Name, err)
	}
}

// waitForPodRunning watches the pod until it reaches Running, then adds an Evict event
// for (time.Now() + duration).
func (m *Manager) waitForPodRunning(pod *corev1.Pod, duration time.Duration) {
	fieldSelector := fmt.Sprintf("metadata.name=%s", pod.Name)
	w, err := m.kubeClient.CoreV1().Pods(pod.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		logrus.Errorf("Error watching pod %s: %v", pod.Name, err)
		return
	}
	defer w.Stop()

	for event := range w.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			logrus.Errorf("Unexpected type in watch: %T", event.Object)
			continue
		}

		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			logrus.Infof("Pod %s is running at %s", pod.Name, time.Now().Format("15:04:05"))

			// Update metrics: scheduled time = now. pending time = scheduled - creation
			key := fmt.Sprintf("%s-%s", p.Namespace, p.Name)
			now := time.Now()

			m.metricsMu.Lock()
			if pm, exists := m.podMetrics[key]; exists {
				pm.ScheduledTime = now
				pm.PendingDuration = pm.ScheduledTime.Sub(pm.CreationTime)
			}
			m.metricsMu.Unlock()

			evictionTime := time.Now().Add(duration)
			evictEvent := &Event{
				At:   evictionTime,
				Type: Evict,
				Pod:  p,
			}
			m.addEvent(evictEvent)
			return
		}
	}
}

// addEvent thread-safely pushes an event into the eventQueue
func (m *Manager) addEvent(e *Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	heap.Push(&m.eventQueue, e)
}

// SubmitPod schedules a future Submit event.
// `arrivalDelay` is how many seconds in the *future* you want to create the pod.
// `lifetime` is how long the pod should run before eviction once it hits Running.
func (m *Manager) SubmitPod(pod *corev1.Pod, arrivalDelay, lifetime time.Duration) {
	arrivalTime := time.Now().Add(arrivalDelay)
	submitEvent := &Event{
		At:       arrivalTime,
		Type:     Submit,
		Pod:      pod,
		Duration: lifetime, // The time from Running to Evict
	}
	m.addEvent(submitEvent)
}

func (m *Manager) SubmitPodInSeconds(pod *corev1.Pod, arrivalDelay, lifetime int) {
	m.SubmitPod(pod, time.Duration(arrivalDelay), time.Duration(lifetime))
}

// CreateNode allows you to create a new fake node (useful for KWOK).
func (m *Manager) CreateNode(node *corev1.Node) error {
	logrus.Infof("Creating node %s ...", node.Name)
	_, err := m.kubeClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("Error creating node %s: %v", node.Name, err)
		return err
	}
	logrus.Infof("Node %s created successfully", node.Name)
	return nil
}

// Cleanup removes all Pods (and optionally all Nodes) from the cluster.
func (m *Manager) Cleanup(removeNodes bool) error {
	logrus.Info("Cleaning up all Pods...")
	namespaces, err := m.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("Error listing namespaces: %v", err)
		return err
	}

	for _, namespace := range namespaces.Items {
		err := m.kubeClient.CoreV1().Pods(namespace.Namespace).DeleteCollection(
			context.TODO(),
			metav1.DeleteOptions{},
			metav1.ListOptions{},
		)
		if err != nil {
			logrus.Errorf("Error deleting pods in namespace %s: %v", namespace.Name, err)
		}
	}

	if err != nil {
		logrus.Errorf("Error deleting pods: %v", err)
		return err
	}

	// Wait a bit for them to be fully deleted (or use a watch)
	time.Sleep(2 * time.Second)

	if removeNodes {
		logrus.Info("Cleaning up all Nodes...")

		// 2. List all nodes and remove them
		nodeList, err := m.kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			logrus.Errorf("Error listing nodes: %v", err)
			return err
		}

		for _, node := range nodeList.Items {
			logrus.Infof("Deleting node %s", node.Name)
			err := m.kubeClient.CoreV1().Nodes().Delete(context.TODO(), node.Name, metav1.DeleteOptions{})
			if err != nil {
				logrus.Errorf("Error deleting node %s: %v", node.Name, err)
			}
		}
	}
	return nil
}
