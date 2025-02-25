package main

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	// 3. Create 5 fake nodes for KWOK:
	//    - 2 with disktype=ssd
	//    - 3 with zone=us-central1-a

	mgr, err := pkg.NewManager(pkg.WithSimulationEnd(480))

	if err != nil {
		logrus.Fatalf("Failed to create manager: %v", err)
	}

	for i := 1; i <= 2; i++ {
		nodeName := fmt.Sprintf("kwok-ssd-node-%d", i)
		labels := map[string]string{
			"disktype":              "ssd",
			"kwok.sigs.k8s.io/node": "true", // helps identify for cleanup
		}
		node := pkg.NewNodeWithLabels(nodeName, "4", "16Gi", "110", labels)
		if err := mgr.CreateNode(node); err != nil {
			logrus.Errorf("CreateNode %s failed: %v", nodeName, err)
		}
	}

	for i := 1; i <= 3; i++ {
		nodeName := fmt.Sprintf("kwok-zone-node-%d", i)
		labels := map[string]string{
			"zone":                  "us-central1-a",
			"kwok.sigs.k8s.io/node": "true",
		}
		node := pkg.NewNodeWithLabels(nodeName, "4", "16Gi", "110", labels)
		if err := mgr.CreateNode(node); err != nil {
			logrus.Errorf("CreateNode %s failed: %v", nodeName, err)
		}
	}

	var withSoftZoneAffinity = map[string]string{"zone": "us-central1-a"}
	batchInterval := 30 * time.Second
	numBatches := 10

	for batch := 1; batch <= numBatches; batch++ {
		arrivalDelay := time.Duration(batch) * batchInterval
		logrus.Infof("Scheduling batch %d at +%s", batch, arrivalDelay)

		// 5 small pods
		for i := 1; i <= 5; i++ {
			pod := pkg.NewPod(
				pkg.WitMetadata(fmt.Sprintf("small-pod-b%d-%d", batch, i), "default"),
				pkg.WithResource("0.5", "256Mi"),
				pkg.WithAffinity(nil, withSoftZoneAffinity), // small pods have soft preference for zone=us-central1-a

			)
			mgr.SubmitPod(pod, arrivalDelay, 90*time.Second)
		}

		// 3 medium pods
		for i := 1; i <= 3; i++ {
			pod := pkg.NewPod(
				pkg.WitMetadata(fmt.Sprintf("medium-pod-b%d-%d", batch, i), "default"),
				pkg.WithResource("1", "512Mi"),
				pkg.WithAffinity(nil, nil), // no soft affinity
			)
			mgr.SubmitPod(pod, arrivalDelay, 90*time.Second)
		}

		// 2 large pods
		for i := 1; i <= 2; i++ {
			pod := pkg.NewPod(
				pkg.WitMetadata(fmt.Sprintf("large-pod-b%d-%d", batch, i), "default"),
				pkg.WithResource("2", "2Gi"),
				pkg.WithAffinity(map[string]string{"disktype": "ssd"}, nil), // large pods require disktype=ssd
			)
			mgr.SubmitPod(pod, arrivalDelay, 90*time.Second)
		}
	}

}
