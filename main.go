package main

import (
	"context"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/simulation"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func main() {
	//cmd.Execute()
	scdl := scheduler.New(logger.Default())

	podEvent := simulation.NewCreatePodEvent(5*time.Second, 10*time.Second, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
				},
			},
			Resources: &v1.ResourceRequirements{Requests: v1.ResourceList{
				"cpu":    resource.MustParse("100m"),
				"memory": resource.MustParse("128Mi"),
			},
			},
		},
	})
	scdl.Schedule(podEvent)
	scdl.Start(context.Background())
	time.Sleep(1500 * time.Second)

}
