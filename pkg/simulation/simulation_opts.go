package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
)

type Option func(*Simulation)

func WithLogger(l logger.Logger) Option {
	return func(s *Simulation) {
		s.logger = l
	}
}

func WithScenario(scenario *scenario.Scenario) Option {
	return func(s *Simulation) {
		s.Scenario = scenario
	}
}

func WithKubeManager(m *utils.KubernetesManager) Option {
	return func(s *Simulation) {
		s.kubeMgr = m
	}
}

func WithResultDir(dir string) Option {
	return func(s *Simulation) {
		s.resultDir = dir
	}
}
