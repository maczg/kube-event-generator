package scenario

import (
	"context"
	"errors"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
)

type PodEvent struct {
	scheduler       scheduler.Scheduler
	Pod             *v1.Pod `yaml:"pod" json:"pod"`
	clientset       *kubernetes.Clientset
	Name            string        `yaml:"name" json:"name"`
	ExecuteAfter    EventDuration `yaml:"after" json:"after"`
	ExecuteDuration EventDuration `yaml:"duration" json:"duration"`
}

func NewPodEvent(name string, after time.Duration, duration time.Duration, pod *v1.Pod, clientset *kubernetes.Clientset, scheduler scheduler.Scheduler) *PodEvent {
	return &PodEvent{
		Name:            name,
		ExecuteAfter:    EventDuration(after),
		ExecuteDuration: EventDuration(duration),
		Pod:             pod,
		clientset:       clientset,
		scheduler:       scheduler,
	}
}

func (p *PodEvent) SetScheduler(scheduler scheduler.Scheduler) {
	p.scheduler = scheduler
}

func (p *PodEvent) SetClientset(clientset *kubernetes.Clientset) {
	p.clientset = clientset
}

func (p *PodEvent) ID() string { return p.Name }

func (p *PodEvent) ExecuteAfterDuration() time.Duration { return time.Duration(p.ExecuteAfter) }

func (p *PodEvent) ExecuteForDuration() time.Duration { return time.Duration(p.ExecuteDuration) }

func (p *PodEvent) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"event_type": "pod",
		"pod_name":   p.Pod.Name,
		"namespace":  p.Pod.Namespace,
	})

	log.Info("executing pod event")

	if _, err := p.clientset.CoreV1().Pods(p.Pod.Namespace).Create(ctx, p.Pod, metav1.CreateOptions{}); err != nil {
		return err
	}

	if p.ExecuteForDuration() > 0 {
		if w, err := p.clientset.CoreV1().Pods(p.Pod.Namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + p.Name,
		}); err != nil {
			return err
		} else {
			for {
				select {
				case <-ctx.Done():
					return errors.New("cannot wait pod finish, context canceled")
				case kubeEvent := <-w.ResultChan():
					if pod, ok := kubeEvent.Object.(*v1.Pod); ok {
						if pod.Status.Phase == v1.PodRunning {
							evictionTime := time.Since(p.scheduler.StartedAt()) + p.ExecuteForDuration()
							evictEvent := NewEvictPodEvent("eviction-"+p.Name, evictionTime, p.clientset, pod)
							p.scheduler.Schedule(evictEvent)

							log.WithFields(map[string]interface{}{
								"eviction_time": evictionTime,
							}).Debug("scheduled pod eviction")

							return nil
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *PodEvent) ComparePriority(other scheduler.Schedulable) bool {
	return p.ExecuteAfterDuration() < other.ExecuteAfterDuration()
}

type EvictPodEvent struct {
	PodEvent
}

func NewEvictPodEvent(name string, from time.Duration, clientset *kubernetes.Clientset, pod *v1.Pod) *EvictPodEvent {
	return &EvictPodEvent{
		PodEvent: PodEvent{
			Name:         name,
			ExecuteAfter: EventDuration(from),
			Pod:          pod,
			clientset:    clientset,
		},
	}
}

func (ev *EvictPodEvent) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"event_type": "eviction",
		"pod_name":   ev.Pod.Name,
		"namespace":  ev.Pod.Namespace,
	})

	log.Info("executing eviction event")

	if err := ev.clientset.CoreV1().Pods(ev.Pod.Namespace).Delete(ctx, ev.Pod.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
