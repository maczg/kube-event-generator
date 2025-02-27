package scheduler

import (
	"context"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type Event interface {
	At() time.Duration
	Execute(ctx context.Context) error
	IsEvictable() bool
	Duration() time.Duration
}

type PodEvent struct {
	ID       string
	pod      *corev1.Pod
	at       *time.Duration
	duration *time.Duration
}

func NewPodEvent(pod corev1.Pod, at time.Duration, for_ time.Duration) PodEvent {
	return PodEvent{
		ID:       uuid.New().String()[0:5],
		pod:      &pod,
		at:       &at,
		duration: &for_,
	}
}

func (pe *PodEvent) At() time.Duration {
	return *pe.at
}

func (pe *PodEvent) IsEvictable() bool {
	return *pe.duration > 0
}

func (pe *PodEvent) Duration() time.Duration {
	return *pe.duration
}

func (pe *PodEvent) Execute() error {
	panic("implement me")
}
