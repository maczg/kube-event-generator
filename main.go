package main

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg"
	"github.com/sirupsen/logrus"
	"time"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
	})
}

func main() {
	mgr, err := pkg.NewManager(
		pkg.WithSimulationEnd(480),
		pkg.WithKubeClient(),
	)
	if err != nil {
		logrus.Fatalf("Failed to create manager: %v", err)
	}
	go mgr.Run()
	mgr.StartMonitoring(1 * time.Second)

	var withSoftZoneAffinity = map[string]string{"zone": "us-central1-a"}
	batchInterval := 10 * time.Second
	numBatches := 1

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

		//// 3 medium pods
		//for i := 1; i <= 3; i++ {
		//	pod := pkg.NewPod(
		//		pkg.WitMetadata(fmt.Sprintf("medium-pod-b%d-%d", batch, i), "default"),
		//		pkg.WithResource("1", "512Mi"),
		//		pkg.WithAffinity(nil, nil), // no soft affinity
		//	)
		//	mgr.SubmitPod(pod, arrivalDelay, 90*time.Second)
		//}
		//
		//// 2 large pods
		//for i := 1; i <= 2; i++ {
		//	pod := pkg.NewPod(
		//		pkg.WitMetadata(fmt.Sprintf("large-pod-b%d-%d", batch, i), "default"),
		//		pkg.WithResource("2", "2Gi"),
		//		pkg.WithAffinity(map[string]string{"disktype": "ssd"}, nil), // large pods require disktype=ssd
		//	)
		//	mgr.SubmitPod(pod, arrivalDelay, 90*time.Second)
		//}
	}

	<-mgr.StopCh
}
