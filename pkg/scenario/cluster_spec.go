package scenario

type Node struct {
	Name            string            `yaml:"name" json:"name"`
	CPUAllocatable  string            `yaml:"cpuAllocatable" json:"cpuAllocatable"`
	MemAllocatable  string            `yaml:"memAllocatable" json:"memAllocatable"`
	PodsAllocatable string            `yaml:"pod" json:"pod"`
	Labels          map[string]string `yaml:"labels" json:"labels"`
}
