package main

//
//import (
//	"github.com/maczg/kube-event-generator/internal"
//	"github.com/maczg/kube-event-generator/pkg"
//	"github.com/sirupsen/logrus"
//	"time"
//)
//
//func _test() {
//	mgr, err := pkg.NewManager()
//	if err != nil {
//		logrus.Fatalf("Failed to create manager: %v", err)
//	}
//	err = mgr.Cleanup(true)
//	if err != nil {
//		logrus.Fatalf("Failed to cleanup: %v", err)
//	}
//
//	fakeNode := internal.NewNode("fake-node", "4", "4Gi", "110")
//
//	// Create the node in the cluster
//	if err := mgr.CreateNode(fakeNode); err != nil {
//		logrus.Errorf("CreateNode failed: %v", err)
//	}
//
//	go mgr.Run()
//
//	pod1 := NewPod("pod-1", "default", "1", "100Mi")
//	pod2 := NewPod("pod-2", "default", "1", "100Mi")
//
//	mgr.SubmitPod(pod1, 5*time.Second, 20*time.Second)
//	mgr.SubmitPod(pod2, 10*time.Second, 25*time.Second)
//
//	logrus.Info("Manager is running. Press Ctrl+C to stop.")
//}
