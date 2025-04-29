package cache

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	cc "k8s.io/client-go/tools/cache"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu *sync.RWMutex
	//log       logger.Logger
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
	logrus.Infoln("starting store")
	factory := informers.NewSharedInformerFactory(s.clientset, 0)
	nodeInformer := factory.Core().V1().Nodes().Informer()
	podInformer := factory.Core().V1().Pods().Informer()

	nodeInformer.AddEventHandler(cc.ResourceEventHandlerFuncs{
		AddFunc:    s.onAddNode,
		UpdateFunc: s.onUpdateNode,
		DeleteFunc: s.onDeleteNode,
	})

	podInformer.AddEventHandler(cc.ResourceEventHandlerFuncs{
		AddFunc:    s.addPod,
		UpdateFunc: s.updatePod,
		DeleteFunc: s.deletePod,
	})
	factory.Start(s.stopCh)
}

func (s *Store) Stop() {
	logrus.Infoln("stopping")
	for _, nodeInfo := range s.nodesInfo {
		s.stats.UpdateHistory(nodeInfo.Copy())
	}
	close(s.stopCh)
}

func (s *Store) onAddNode(obj interface{}) {
	node := obj.(*v1.Node)
	s.mu.Lock()
	defer s.mu.Unlock()

	newNodeInfo := NewNodeStore(node)
	s.nodesInfo[node.Name] = newNodeInfo
	s.stats.UpdateHistory(newNodeInfo.Copy())

	logrus.Debugf("[onAdd] node %s added", node.Name)
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
		logrus.Debugf("[onAdd] pod %s added to pending queue", pod.Name)
		s.stats.UpdatePendingQ(pod, AddPodToPendingQ)
	}
	if nodeName := pod.Spec.NodeName; nodeName != "" {
		logrus.Debugf("[onAdd] pod %s added to node %s", pod.Name, nodeName)
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
			logrus.Debugf("[onUpdate] pod %s added to pending queue", newPod.Name)
			s.stats.UpdatePendingQ(newPod, AddPodToPendingQ)
		}
	}
	// pod is running on another node or from "" to newPodNodeName
	if oldPodNodeName != newPodNodeName {
		// from pending to running so remove from pending queue and track pendingTime
		if oldPodNodeName == "" && newPodNodeName != "" {
			logrus.Debugf("[onUpdate] %s is now on %s, removing from pending queue", oldPod.Name, newPodNodeName)
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
			logrus.Warnf("[onUpdate] pod %s has newNodePodName empty. Deleted?", newPod.Name)
		}
	}
}

func (s *Store) deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	s.stats.UpdatePodEvent(NewPodEvent(pod, "delete"))

	s.mu.Lock()
	defer s.mu.Unlock()
	logrus.Debugf("[onDelete] pod %s deleted", pod.Name)

	key := NewKey(pod)

	switch pod.Status.Phase {
	case v1.PodRunning:
		logrus.Debugf("[onDelete] running pod %s delete", pod.Name)
		if nodeName := pod.Spec.NodeName; nodeName != "" {
			s.stats.RunningDurations[key] = time.Since(pod.GetCreationTimestamp().Time)
			if nodeInfo, ok := s.nodesInfo[nodeName]; ok {
				nodeInfo.deletePod(pod)
				s.stats.UpdateHistory(nodeInfo.Copy())
			}
		}
	case v1.PodPending:
		// pod was in pending queue, remove from pending queue
		logrus.Debugf("[onDelete] pending pod %s delete", pod.Name)
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

func (s *Store) GetNodeInfo(nodeName string) NodeStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.nodesInfo[nodeName]
}

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
	logrus.Infof("logging every %d seconds", seconds)
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
			logrus.Infoln(statusbuilder.String())
			s.mu.RUnlock()
		case <-s.stopCh:
			return
		}
	}
}
