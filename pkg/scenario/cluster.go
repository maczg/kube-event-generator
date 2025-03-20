package scenario

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
)

type Cluster struct {
	// Nodes in the cluster
	Nodes []v1.Node `yaml:"nodes" json:"nodes"`
	// SchedulerWeights map of scheduler plugin name to weight
	SchedulerWeights map[string]int `yaml:"schedulerWeights" json:"schedulerWeights"`
}

func (c *Cluster) MarshalJSON() ([]byte, error) {
	type alias Cluster
	a := alias(*c)
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	stripEmpty(m)
	return json.Marshal(m)
}

// stripEmpty removes nil or empty slices/maps from the map.
func stripEmpty(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case nil:
			delete(m, k)
		case map[string]interface{}:
			stripEmpty(val)
			if len(val) == 0 {
				delete(m, k)
			}
		case []interface{}:
			if len(val) == 0 {
				delete(m, k)
			} else {
				// Optionally, you could strip empty from slice elements as well.
				for i := range val {
					if subMap, ok := val[i].(map[string]interface{}); ok {
						stripEmpty(subMap)
						if len(subMap) == 0 {
							val[i] = nil
						}
					}
				}
			}
		case string:
			if val == "" {
				delete(m, k)
			}
		}
	}
}
