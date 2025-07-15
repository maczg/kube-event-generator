// Package cache provides a cache for nodes and pods in a Kubernetes cluster.
package cache

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/maczg/kube-event-generator/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	cc "k8s.io/client-go/tools/cache"
)

// Store is a cache for nodes and pods in the cluster.
type Store struct {
	mu *sync.RWMutex
	// log       logger.Logger
	clientset *kubernetes.Clientset
	// nodesInfo is a map of node name to nodeInfo
	nodesInfo map[string]*NodeStore
	// stats contains the cluster state statistics
	stats  *Stats
	stopCh chan struct{}
}

// NewStore creates a new Store instance and starts the informers immediately.
func NewStore(clientset *kubernetes.Clientset) *Store {
	ni := &Store{
		mu:        &sync.RWMutex{},
		clientset: clientset,
		nodesInfo: make(map[string]*NodeStore),
		stats:     NewStats(),
		stopCh:    make(chan struct{}),
	}

	return ni
}

// Start starts the informers for nodes and pods.
// Returns immediately after starting the informers.
func (s *Store) Start() {
	logger.Default().Info("starting store")

	factory := informers.NewSharedInformerFactory(s.clientset, 0)
	nodeInformer := factory.Core().V1().Nodes().Informer()
	podInformer := factory.Core().V1().Pods().Informer()

	_, _ = nodeInformer.AddEventHandler(cc.ResourceEventHandlerFuncs{
		AddFunc:    s.onAddNode,
		UpdateFunc: s.onUpdateNode,
		DeleteFunc: s.onDeleteNode,
	})

	_, _ = podInformer.AddEventHandler(cc.ResourceEventHandlerFuncs{
		AddFunc:    s.addPod,
		UpdateFunc: s.updatePod,
		DeleteFunc: s.deletePod,
	})
	factory.Start(s.stopCh)
}

// Stop stops the store and updates the history with the current state of nodes.
func (s *Store) Stop() {
	logger.Default().Info("stopping")

	for _, nodeInfo := range s.nodesInfo {
		s.stats.UpdateHistory(nodeInfo.Copy())
	}

	close(s.stopCh)
}

// onAddNode is called when a new node is added to the cluster.
func (s *Store) onAddNode(obj interface{}) {
	node := obj.(*v1.Node)

	s.mu.Lock()
	defer s.mu.Unlock()

	newNodeInfo := NewNodeStore(node)
	s.nodesInfo[node.Name] = newNodeInfo
	s.stats.UpdateHistory(newNodeInfo.Copy())

	logger.Default().Debugf("[onAdd] node %s added", node.Name)
}

func (s *Store) onUpdateNode(oldObj, newObj interface{}) {
	newNode := newObj.(*v1.Node)

	s.mu.Lock()
	defer s.mu.Unlock()

	if n, exists := s.nodesInfo[newNode.Name]; exists {
		n.UpdateNodeSpec(newNode)
		s.stats.UpdateHistory(n.Copy())
	} else {
		newNodeInfo := NewNodeStore(newNode)
		s.nodesInfo[newNode.Name] = newNodeInfo
		s.stats.UpdateHistory(newNodeInfo.Copy())
	}
}

func (s *Store) onDeleteNode(obj interface{}) {
	node := obj.(*v1.Node)

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodesInfo, node.Name)
}

func (s *Store) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.UpdatePodEvent(NewPodEvent(pod, "add"))

	if pod.Status.Phase == v1.PodPending {
		logger.Default().Debugf("[onAdd] pod %s added to pending queue", pod.Name)
		s.stats.UpdatePendingQ(pod, AddPodToPendingQ)
	}

	if nodeName := pod.Spec.NodeName; nodeName != "" {
		logger.Default().Debugf("[onAdd] pod %s added to node %s", pod.Name, nodeName)

		if nodeInfo, ok := s.nodesInfo[nodeName]; ok {
			nodeInfo.addPod(pod)
			s.stats.UpdateHistory(nodeInfo.Copy())
		}
	}
}

func (s *Store) updatePod(oldObj, newObj interface{}) {
	oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)
	oldPodNodeName := oldPod.Spec.NodeName
	newPodNodeName := newPod.Spec.NodeName

	s.stats.UpdatePodEvent(NewPodEvent(newPod, "update"))

	s.mu.Lock()
	defer s.mu.Unlock()

	if newPod.Status.Phase == v1.PodPending {
		if _, ok := s.stats.PendingQ[NewKey(newPod)]; !ok {
			logger.Default().Debugf("[onUpdate] pod %s added to pending queue", newPod.Name)
			s.stats.UpdatePendingQ(newPod, AddPodToPendingQ)
		}
	}
	// pod is running on another node or from "" to newPodNodeName
	if oldPodNodeName != newPodNodeName {
		// from pending to running so remove from pending queue and track pendingTime
		if oldPodNodeName == "" && newPodNodeName != "" {
			logger.Default().Debugf("[onUpdate] %s is now on %s, removing from pending queue", oldPod.Name, newPodNodeName)
			s.stats.UpdatePendingQ(oldPod, RemovePodFromPendingQ)

			if nodeInfo, ok := s.nodesInfo[newPodNodeName]; ok {
				nodeInfo.addPod(newPod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}
		// pod is running on another node, remove from old node
		if oldPodNodeName != "" && newPodNodeName != "" {
			if nodeInfo, ok := s.nodesInfo[oldPodNodeName]; ok {
				nodeInfo.deletePod(oldPod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}

			if nodeInfo, ok := s.nodesInfo[newPodNodeName]; ok {
				nodeInfo.addPod(newPod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}
		// simple update - just resync nodeStatus
		if oldPodNodeName == newPodNodeName {
			if nodeInfo, ok := s.nodesInfo[newPodNodeName]; ok {
				nodeInfo.UpdateAllocated()
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}

		if newPodNodeName == "" {
			logger.Default().Warnf("[onUpdate] pod %s has newNodePodName empty. Deleted?", newPod.Name)
		}
	}
}

func (s *Store) deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	s.stats.UpdatePodEvent(NewPodEvent(pod, "delete"))

	s.mu.Lock()
	defer s.mu.Unlock()
	logger.Default().Debugf("[onDelete] pod %s deleted", pod.Name)

	key := NewKey(pod)

	switch pod.Status.Phase {
	case v1.PodRunning:
		logger.Default().Debugf("[onDelete] running pod %s delete", pod.Name)

		if nodeName := pod.Spec.NodeName; nodeName != "" {
			s.stats.RunningDurations[key] = time.Since(pod.GetCreationTimestamp().Time)
			if nodeInfo, ok := s.nodesInfo[nodeName]; ok {
				nodeInfo.deletePod(pod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}
	case v1.PodPending:
		// pod was in pending queue, remove from pending queue
		logger.Default().Debugf("[onDelete] pending pod %s delete", pod.Name)
		s.stats.UpdatePendingQ(pod, RemovePodFromPendingQ)

		if nodeName := pod.Spec.NodeName; nodeName != "" {
			if nodeInfo, ok := s.nodesInfo[nodeName]; ok {
				nodeInfo.deletePod(pod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}

		if nodeInfo, exist := s.nodesInfo[pod.Spec.NodeName]; exist {
			s.stats.UpdateHistory(nodeInfo.Copy())
		}
	}
}

// GetStats returns the statistics of the cluster
func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.stats
}

// GetNodeInfo returns the NodeStore for the given node name.
func (s *Store) GetNodeInfo(nodeName string) NodeStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.nodesInfo[nodeName]
}

// GetNodesInfo returns a sorted slice of NodeStore for all nodes in the cluster.
func (s *Store) GetNodesInfo() []NodeStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make([]NodeStore, 0, len(s.nodesInfo))
	for _, nodeStatus := range s.nodesInfo {
		status = append(status, *nodeStatus)
	}

	sort.Slice(status, func(i, j int) bool {
		return status[i].Node.Name < status[j].Node.Name
	})

	return status
}

// TODO improve me

// WatchEvery logs the status of all nodes every `seconds` seconds.
// It blocks until the stop channel is closed.
func (s *Store) WatchEvery(seconds int) {
	logger.Default().Infof("logging every %d seconds", seconds)

	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()

			status := make([]NodeStore, 0, len(s.nodesInfo))
			for _, nodeStatus := range s.nodesInfo {
				status = append(status, *nodeStatus)
			}

			sort.Slice(status, func(i, j int) bool {
				return status[i].Node.Name < status[j].Node.Name
			})

			var statusbuilder strings.Builder
			for _, ns := range status {
				statusbuilder.WriteString(ns.String())
			}

			logger.Default().Info(statusbuilder.String())
			s.mu.RUnlock()
		case <-s.stopCh:
			return
		}
	}
}
