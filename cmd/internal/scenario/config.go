package scenario

import (
	"github.com/spf13/viper"
	"k8s.io/apiserver/pkg/storage/names"
	"strings"
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
		NumPodEvents       int     `yaml:"numPodEvents" json:"numPodEvents"`
		ArrivalScale       float64 `yaml:"arrivalScale" json:"arrivalScale"`
		ArrivalScaleFactor float64 `yaml:"arrivalScaleFactor" json:"arrivalScaleFactor"`

		// Scale is lambda for Weibull distribution
		DurationScale float64 `yaml:"durationScale" json:"durationScale"`
		// Shape is k for Weibull distribution
		DurationShape       float64 `yaml:"durationShape" json:"durationShape"`
		DurationScaleFactor float64 `yaml:"durationScaleFactor" json:"durationScaleFactor"`

		PodCpuScale  float64 `yaml:"podCpuScale" json:"podCpuScale"`
		PodCpuShape  float64 `yaml:"podCpuShape" json:"PodCpuShape"`
		PodCpuFactor float64 `yaml:"podCpuFactor" json:"podCpuFactor"`

		PodMemScale  float64 `yaml:"podMemScale" json:"podMemScale"`
		PodMemShape  float64 `yaml:"podMemShape" json:"podMemShape"`
		PodMemFactor float64 `yaml:"podMemFactor" json:"podMemFactor"`
	}
)

func GetConfig(configfile string) *Config {
	name := names.SimpleNameGenerator.GenerateName("scenario-")
	cfg := Config{
		ScenarioName: name,
		OutputDir:    "output",
		Seed:         time.Now().Unix(),
		Generation: Generation{
			NumPodEvents:        50,
			ArrivalScale:        1.0,
			ArrivalScaleFactor:  5.0,
			DurationScale:       1.5,
			DurationShape:       2.0,
			DurationScaleFactor: 10.0,
			PodCpuShape:         1.0,
			PodCpuScale:         1.0,
			PodCpuFactor:        1.0,
			PodMemScale:         1.5,
			PodMemShape:         512.0,
			PodMemFactor:        1.0,
		},
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	_ = viper.ReadInConfig()
	_ = viper.Unmarshal(&cfg)
	cfg.OutputPath = cfg.OutputDir + "/" + cfg.ScenarioName + ".yaml"
	return &cfg
}
