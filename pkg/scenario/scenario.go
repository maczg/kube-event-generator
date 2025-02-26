package scenario

import "gopkg.in/yaml.v3"

type Scenario struct {
	Events []Event `json:"events"`
}

func Load(scenarioData []byte) (*Scenario, error) {
	var scenario Scenario
	err := yaml.Unmarshal(scenarioData, &scenario)
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}
