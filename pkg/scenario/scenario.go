package scenario

type Scenario struct {
	Name string `yaml:"name" json:"name"`
	// Cluster state
	Cluster Cluster `yaml:"cluster" json:"cluster"`
	// Events that will be applied to the cluster
	Events []Event `yaml:"events" json:"events"`
}
