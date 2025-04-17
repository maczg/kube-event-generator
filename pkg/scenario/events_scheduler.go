package scenario

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/sirupsen/logrus"
	"io"
	kubescheduler "k8s.io/kube-scheduler/config/v1"
	"net/http"
	"os"
	"time"
)

var schedulerSimUrl string

func init() {
	schedulerSimUrl = os.Getenv("SCHEDULER_SIM_URL")
	if schedulerSimUrl == "" {
		schedulerSimUrl = "http://localhost:1212/api/v1/schedulerconfiguration"
	}
}

type SchedulerEvent struct {
	Name         string           `yaml:"name" json:"name"`
	ExecuteAfter EventDuration    `yaml:"after" json:"after"`
	Weights      map[string]int32 `yaml:"weights" json:"weights"`
	httpClient   *http.Client
}

func (e *SchedulerEvent) ID() string { return e.Name }

func (e *SchedulerEvent) ExecuteAfterDuration() time.Duration {
	return time.Duration(e.ExecuteAfter)
}

func (e *SchedulerEvent) ExecuteForDuration() time.Duration {
	return 0
}

func (e *SchedulerEvent) ComparePriority(other scheduler.Schedulable) bool {
	return e.ExecuteAfterDuration() < other.ExecuteAfterDuration()
}

func (e *SchedulerEvent) Execute(ctx context.Context) error {
	logrus.Debugf("Executing scheduler event %s", e.Name)
	if e.httpClient == nil {
		e.httpClient = &http.Client{}
	}
	for {
		select {
		case <-ctx.Done():
			return errors.New("cannot wait scheduler event finish, context cancelled")
		default:
			if currentConfig, err := e.GetCurrentSchedulerConfig(); err != nil {
				return err
			} else {
				if err = e.UpdateSchedulerConfig(currentConfig); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func (e *SchedulerEvent) GetCurrentSchedulerConfig() (*kubescheduler.KubeSchedulerConfiguration, error) {
	req, err := http.NewRequest(http.MethodGet, schedulerSimUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get eventScheduler config: %s", resp.Status)
	}
	var config kubescheduler.KubeSchedulerConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (e *SchedulerEvent) UpdateSchedulerConfig(config *kubescheduler.KubeSchedulerConfiguration) error {
	for name, weight := range e.Weights {
		for i, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
			if plugin.Name == name {
				config.Profiles[0].Plugins.MultiPoint.Enabled[i].Weight = &weight
			} else {
				logrus.Debugf("plugin %s not found in eventScheduler config", name)
			}
		}
	}

	req, err := http.NewRequest(http.MethodPut, schedulerSimUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	body, err := json.Marshal(config)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update eventScheduler config: %s", resp.Status)
	}
	return nil
}
