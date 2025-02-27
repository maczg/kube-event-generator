package scenario

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

type Scenario struct {
	Events []Event `json:"events"`
}

func Load(data []byte) (*Scenario, error) {
	var scenario Scenario
	err := yaml.Unmarshal(data, &scenario)
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

func LoadFromPath(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sc, err := Load(data)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Loaded scenario with %d events", len(sc.Events))
	return sc, nil
}
