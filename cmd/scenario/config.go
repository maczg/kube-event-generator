package scenario

import (
	"github.com/google/uuid"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/viper"
	"time"
)

type (
	Config struct {
		ScenarioName string     `yaml:"scenarioName" json:"scenarioName"`
		OutputDir    string     `yaml:"outputDir" json:"outputDir"`
		OutputPath   string     `yaml:"-" json:"-"`
		Seed         int64      `yaml:"seed" json:"seed"`
		Generation   Generation `yaml:"generation" json:"generation"`
	}

	Generation struct {
		NumPodEvents int `yaml:"numPodEvents" json:"numPodEvents"`
		// ArrivalScale is the rate (lambda) of the exponential inter-arrival distribution.
		ArrivalScale float64 `yaml:"arrivalScale" json:"arrivalScale"`
		// ArrivalScaleFactor converts sampled arrival times to seconds.
		ArrivalScaleFactor float64 `yaml:"arrivalScaleFactor" json:"arrivalScaleFactor"`

		// Scale is lambda for Weibull distribution
		DurationScale float64 `yaml:"durationScale" json:"durationScale"`
		// Shape is k for Weibull distribution
		DurationShape       float64 `yaml:"durationShape" json:"durationShape"`
		DurationScaleFactor float64 `yaml:"durationScaleFactor" json:"durationScaleFactor"`

		// CPU request in cores, Weibull(shape, scale); PodCpuFactor is not used yet.
		PodCpuScale  float64 `yaml:"podCpuScale" json:"podCpuScale"`
		PodCpuShape  float64 `yaml:"podCpuShape" json:"podCpuShape"`
		PodCpuFactor float64 `yaml:"podCpuFactor" json:"podCpuFactor"`

		// Memory request in MiB, Weibull(shape, scale) multiplied by PodMemFactor.
		PodMemScale  float64 `yaml:"podMemScale" json:"podMemScale"`
		PodMemShape  float64 `yaml:"podMemShape" json:"podMemShape"`
		PodMemFactor float64 `yaml:"podMemFactor" json:"podMemFactor"`

		// Node parameters are not used yet: nodes are created outside keg.
		NumNodes    int `yaml:"numNodes" json:"numNodes"`
		NodeCpuMean int `yaml:"nodeCpuMean" json:"nodeCpuMean"`
		NodeMemMean int `yaml:"nodeMemMean" json:"nodeMemMean"`
	}
)

func GetConfig(configfile string) *Config {
	name := "scenario-" + uuid.New().String()[0:8]
	cfg := Config{
		ScenarioName: name,
		OutputDir:    "output",
		Seed:         time.Now().Unix(),
		Generation: Generation{
			NumPodEvents:        50,
			ArrivalScale:        1.0,
			ArrivalScaleFactor:  5.0,
			DurationScale:       2.0,
			DurationShape:       1.5,
			DurationScaleFactor: 10.0,
			PodCpuShape:         1.0,
			PodCpuScale:         1.0,
			PodCpuFactor:        1.0,
			PodMemScale:         512.0,
			PodMemShape:         1.5,
			PodMemFactor:        1.0,
			NumNodes:            5,
			NodeCpuMean:         8,
			NodeMemMean:         16,
		},
	}

	if configfile != "" {
		viper.SetConfigFile(configfile)
		if err := viper.ReadInConfig(); err != nil {
			logger.Default().Fatalf("failed to read config file %s, error %v", configfile, err)
		}
		if err := viper.Unmarshal(&cfg); err != nil {
			logger.Default().Fatalf("failed to unmarshal config file %s, error %v", configfile, err)
		}
	}

	return &cfg
}
