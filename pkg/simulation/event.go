package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1 "k8s.io/kube-scheduler/config/v1"
	"net/http"
	"time"
)

type event struct {
	scenario.Event
	scheduler scheduler.Scheduler
	mgr       *utils.KubernetesManager
}

func (e *event) ID() string { return e.Name }

func (e *event) After() time.Duration { return e.From }

func (e *event) For() time.Duration { return e.Duration }

func (e *event) At() time.Duration { return e.From }

type CreatePodEvent struct {
	event
}

func NewCreatePodEvent(e scenario.Event, mgr *utils.KubernetesManager, scheduler scheduler.Scheduler) *CreatePodEvent {
	return &CreatePodEvent{
		event: event{
			Event:     e,
			mgr:       mgr,
			scheduler: scheduler,
		},
	}
}

func (s *CreatePodEvent) Run(ctx context.Context) error {
	c := s.mgr.Clientset()
	p, err := s.mgr.CreatePod(ctx, *c, &s.Pod)
	if err != nil {
		return err
	}
	if s.For() > 0 {
		w, err := c.CoreV1().Pods(p.Namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + p.Name,
		})
		if err != nil {
			return err
		}
		for {
			select {
			case <-ctx.Done():
				return errors.New("cannot wait pod finish, context cancelled")
			case e := <-w.ResultChan():
				if pod, ok := e.Object.(*corev1.Pod); ok {
					if pod.Status.Phase == corev1.PodRunning {
						newEvEvent := NewEvictionEvent(s.Event, s.mgr)
						newEvEvent.Pod = *pod
						at := time.Since(s.scheduler.StartedAt()) + s.For()
						newEvEvent.From = at
						newEvEvent.Duration = 0
						newEvEvent.Name = "eviction-" + s.Name
						s.scheduler.Schedule(newEvEvent)
						return nil
					}
				}
			}
		}
	}
	return nil
}

type EvictPodEvent struct {
	event
}

func (e *EvictPodEvent) Run(ctx context.Context) error {
	c := e.mgr.Clientset()
	err := e.mgr.DeletePod(ctx, *c, &e.Pod)
	if err != nil {
		return err
	}
	return nil
}

func NewEvictionEvent(e scenario.Event, mgr *utils.KubernetesManager) *EvictPodEvent {
	return &EvictPodEvent{
		event: event{
			Event: e,
			mgr:   mgr,
		},
	}
}

const serverEndpoint = "http://localhost:1212/api/v1/schedulerconfiguration"

type ChangeWeightEvent struct {
	UID     string
	From    time.Duration
	Weights map[string]int
}

func NewChangeWeightEvent(from time.Duration, weights map[string]int) *ChangeWeightEvent {
	return &ChangeWeightEvent{
		UID:     "change-weight-" + time.Now().Format("20060102150405"),
		From:    from,
		Weights: weights,
	}
}

func (e *ChangeWeightEvent) ID() string {
	return e.UID
}

func (e *ChangeWeightEvent) After() time.Duration {
	return e.From
}

func (e *ChangeWeightEvent) For() time.Duration {
	return 0
}

func (e *ChangeWeightEvent) Run(ctx context.Context) error {
	// get scheduler config
	req, _ := http.NewRequest("GET", serverEndpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("failed to get scheduler config")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("failed to read response body")
	}
	var config configv1.KubeSchedulerConfiguration
	// unmarshal response body to config
	if err := json.Unmarshal(data, &config); err != nil {
		return errors.New("failed to unmarshal scheduler config")
	}
	// change weights
	for name, weight := range e.Weights {
		for i, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
			if plugin.Name == name {
				w := int32(weight)
				config.Profiles[0].Plugins.MultiPoint.Enabled[i].Weight = &w
			}
		}
	}
	// marshal config to json
	data, err = json.Marshal(config)
	if err != nil {
		return errors.New("failed to marshal scheduler config")
	}
	// send post request to server
	req, _ = http.NewRequest("POST", serverEndpoint, io.NopCloser(bytes.NewReader(data)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("failed to send post request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return errors.New("failed to change weights")
	}
	return nil
}
