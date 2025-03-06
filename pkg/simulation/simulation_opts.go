package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
)

type Option func(*Simulation)

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
