package scenario

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/maczg/kube-event-generator/pkg/logger"
)

// Config represents scenario generation configuration.
type Config struct {
	ScenarioName string           `mapstructure:"scenarioName"`
	OutputDir    string           `mapstructure:"outputDir"`
	OutputPath   string           `mapstructure:"outputPath"`
	Generation   GenerationConfig `mapstructure:"generation"`
	Seed         int64            `mapstructure:"seed"`
}

// GenerationConfig represents generation parameters.
type GenerationConfig struct {
	SchedulerEvents     []SchedulerEventConfig `mapstructure:"schedulerEvents"`
	PodCpuShape         float64                `mapstructure:"podCpuShape"`
	ArrivalScaleFactor  float64                `mapstructure:"arrivalScaleFactor"`
	DurationScale       float64                `mapstructure:"durationScale"`
	DurationShape       float64                `mapstructure:"durationShape"`
	DurationScaleFactor float64                `mapstructure:"durationScaleFactor"`
	NumPodEvents        int                    `mapstructure:"numPodEvents"`
	PodCpuScale         float64                `mapstructure:"podCpuScale"`
	PodCpuFactor        float64                `mapstructure:"podCpuFactor"`
	PodMemScale         float64                `mapstructure:"podMemScale"`
	PodMemShape         float64                `mapstructure:"podMemShape"`
	PodMemFactor        float64                `mapstructure:"podMemFactor"`
	ArrivalScale        float64                `mapstructure:"arrivalScale"`
}

// SchedulerEventConfig represents a scheduler event configuration.
type SchedulerEventConfig struct {
	Weights map[string]int32 `mapstructure:"weights"`
	Name    string           `mapstructure:"name"`
	After   time.Duration    `mapstructure:"after"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		ScenarioName: "scenario-default",
		OutputDir:    "scenarios",
		Seed:         time.Now().UnixNano(),
		Generation: GenerationConfig{
			NumPodEvents:        100,
			ArrivalScale:        2.0,
			ArrivalScaleFactor:  5.0,
			DurationScale:       1.5,
			DurationShape:       2.0,
			DurationScaleFactor: 10.0,
			PodCpuShape:         3.0,
			PodCpuScale:         1.0,
			PodCpuFactor:        1.0,
			PodMemScale:         2.0,
			PodMemShape:         512.0,
			PodMemFactor:        1.0,
		},
	}
}

// GetConfig loads configuration from a file or returns default.
func GetConfig(configFile string) Config {
	log := logger.Default()

	cfg := DefaultConfig()

	if configFile == "" {
		log.Debug("No config file specified, using defaults")
		return cfg
	}

	v := viper.New()
	v.SetConfigFile(configFile)

	// Set defaults.
	v.SetDefault("scenarioName", cfg.ScenarioName)
	v.SetDefault("outputDir", cfg.OutputDir)
	v.SetDefault("seed", cfg.Seed)
	v.SetDefault("generation", cfg.Generation)

	// Read config file.
	if err := v.ReadInConfig(); err != nil {
		log.WithFields(map[string]interface{}{
			"file":  configFile,
			"error": err,
		}).Warn("Failed to read config file, using defaults")

		return cfg
	}

	// Unmarshal config.
	if err := v.Unmarshal(&cfg); err != nil {
		log.WithFields(map[string]interface{}{
			"file":  configFile,
			"error": err,
		}).Error("Failed to parse config file")

		return DefaultConfig()
	}

	// Generate output path if not set.
	if cfg.OutputPath == "" && cfg.OutputDir != "" && cfg.ScenarioName != "" {
		cfg.OutputPath = filepath.Join(cfg.OutputDir, fmt.Sprintf("%s.yaml", cfg.ScenarioName))
	}

	log.WithFields(map[string]interface{}{
		"scenario_name": cfg.ScenarioName,
		"output_dir":    cfg.OutputDir,
		"num_pods":      cfg.Generation.NumPodEvents,
	}).Debug("Configuration loaded")

	return cfg
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.ScenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}

	if c.Generation.NumPodEvents <= 0 {
		return fmt.Errorf("number of pod events must be positive")
	}

	if c.Generation.ArrivalScale <= 0 {
		return fmt.Errorf("arrival scale must be positive")
	}

	if c.Generation.DurationScale <= 0 {
		return fmt.Errorf("duration scale must be positive")
	}

	if c.Generation.DurationShape <= 0 {
		return fmt.Errorf("duration shape must be positive")
	}

	if c.Generation.PodCpuShape <= 0 {
		return fmt.Errorf("pod CPU shape must be positive")
	}

	if c.Generation.PodCpuScale <= 0 {
		return fmt.Errorf("pod CPU scale must be positive")
	}

	if c.Generation.PodMemScale <= 0 {
		return fmt.Errorf("pod memory scale must be positive")
	}

	if c.Generation.PodMemShape <= 0 {
		return fmt.Errorf("pod memory shape must be positive")
	}

	return nil
}
