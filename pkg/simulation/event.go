package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
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
