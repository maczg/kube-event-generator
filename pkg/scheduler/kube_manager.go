package scheduler

import (
	"context"
	"github.com/maczg/kube-event-generator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//const WatchPodTimeout = 1 * time.Minute

type KubeManager struct {
	client *kubernetes.Clientset
}

func NewKubeManager() (*KubeManager, error) {
	c, err := utils.GetClientset()
	if err != nil {
		return nil, err
	}
	return &KubeManager{client: c}, nil
}

func (k *KubeManager) Clientset() *kubernetes.Clientset {
	return k.client
}

func (km *KubeManager) CreatePod(pod *corev1.Pod) error {
	_, err := km.client.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	return err
}

func (km *KubeManager) DeletePod(namespace, name string) error {
	return km.client.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (km *KubeManager) GetPod(namespace, name string) (*corev1.Pod, error) {
	return km.client.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (km *KubeManager) ListPods(namespace string) (*corev1.PodList, error) {
	return km.client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
}

func (km *KubeManager) CreateNode(node *corev1.Node) error {
	_, err := km.client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	return err
}

func (km *KubeManager) DeleteNode(name string) error {
	return km.client.CoreV1().Nodes().Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (km *KubeManager) GetNode(name string) (*corev1.Node, error) {
	return km.client.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
}

func (km *KubeManager) ListNodes() (*corev1.NodeList, error) {
	return km.client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
}

func (km *KubeManager) WaitForPodReady(namespace, name string) error {
	//timeout := time.After(WatchPodTimeout)
	w, err := km.client.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	defer w.Stop()
	for {
		select {
		case e := <-w.ResultChan():
			pod, ok := e.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			if pod.Name == name && pod.Status.Phase == corev1.PodRunning {
				return nil
			}
			//case <-timeout:
			//	return fmt.Errorf("timeout waiting for pod %s to be ready", name)
		}
	}
}
