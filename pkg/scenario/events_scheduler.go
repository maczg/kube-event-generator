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

func getOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

var schedulerSimUrl = getOrDefault("SCHEDULER_SIM_URL", "http://localhost:1212/api/v1/schedulerconfiguration")

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
	logrus.Infof("executing scheduler event %s", e.Name)
	logrus.Debugf("schedulerSim url: %s", schedulerSimUrl)
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
	data, err := io.ReadAll(resp.Body)
	var config kubescheduler.KubeSchedulerConfiguration
	// unmarshal response body to config
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, errors.New("failed to unmarshal scheduler config")
	}
	return &config, nil
}

func (e *SchedulerEvent) UpdateSchedulerConfig(config *kubescheduler.KubeSchedulerConfiguration) error {
	matched := []string{}
	for name, weight := range e.Weights {
		for i, plugin := range config.Profiles[0].Plugins.MultiPoint.Enabled {
			if plugin.Name == name {
				matched = append(matched, name)
				config.Profiles[0].Plugins.MultiPoint.Enabled[i].Weight = &weight
			}
		}
	}
	if len(matched) == 0 {
		logrus.Warnf("no matched plugin found in scheduler config, please check the plugin name")
	}
	// marshal config to json
	data, err := json.Marshal(config)
	if err != nil {
		return errors.New("failed to marshal scheduler config")
	}
	req, err := http.NewRequest(http.MethodPost, schedulerSimUrl, io.NopCloser(bytes.NewReader(data)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to update eventScheduler config: %s", resp.Status)
	}
	return nil
}
