package main

import (
	"flag"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"os"
)

const (
	MaxCpuRequest    = 2000
	MinCpuRequest    = 100
	MaxMemoryRequest = 2000
	MinMemoryRequest = 100
)

// NewPod creates a new pod with the given name, cpu, and memory
func NewPod(name string, namespace string, cpu string, memory string) *corev1.Pod {
	p := &corev1.Pod{}
	p.Name = name
	p.Namespace = namespace
	p.Spec.Containers = []corev1.Container{
		{
			Name:  "server",
			Image: "nginx:latest",
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 80,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(cpu),
					corev1.ResourceMemory: resource.MustParse(memory),
				},
			},
		},
	}
	return p
}

func NewRandomPodSpec() *corev1.Pod {
	randomCpuRequest := rand.Intn(MaxCpuRequest-MinCpuRequest) + MinCpuRequest
	randomMemoryRequest := rand.Intn(MaxMemoryRequest-MinMemoryRequest) + MinMemoryRequest
	name := names.SimpleNameGenerator.GenerateName("random-")
	return NewPod(name, "default", fmt.Sprintf("%dm", randomCpuRequest), fmt.Sprintf("%dMi", randomMemoryRequest))
}

func BuildClientSetFromFlag() *kubernetes.Clientset {
	var kubeconfig string
	flag.StringVar(&kubeconfig, "k", "", "absolute path to the kubeconfig file")
	flag.Parse()
	if kubeconfig == "" {
		kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return clientset
}
