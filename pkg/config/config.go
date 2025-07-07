package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"

	"github.com/maczg/kube-event-generator/pkg/errors"
)

// Config represents the main configuration structure.
type Config struct {
	Output     OutputConfig     `yaml:"output"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Scenario   ScenarioConfig   `yaml:"scenario"`
}

// ScenarioConfig holds scenario generation parameters.
type ScenarioConfig struct {
	Name       string           `yaml:"name"`
	OutputDir  string           `yaml:"outputDir"`
	Generation GenerationConfig `yaml:"generation"`
}

// GenerationConfig holds distribution parameters.
type GenerationConfig struct {
	NumPodEvents        int     `yaml:"numPodEvents"`
	ArrivalScale        float64 `yaml:"arrivalScale"`
	ArrivalScaleFactor  float64 `yaml:"arrivalScaleFactor"`
	DurationScale       float64 `yaml:"durationScale"`
	DurationShape       float64 `yaml:"durationShape"`
	DurationScaleFactor float64 `yaml:"durationScaleFactor"`
	PodCPUShape         float64 `yaml:"podCpuShape"`
	PodCPUFactor        float64 `yaml:"podCpuFactor"`
	PodMemScale         float64 `yaml:"podMemScale"`
	PodMemShape         float64 `yaml:"podMemShape"`
	PodMemFactor        float64 `yaml:"podMemFactor"`
}

// KubernetesConfig holds Kubernetes-related configuration.
type KubernetesConfig struct {
	Kubeconfig     string        `yaml:"kubeconfig"`
	Context        string        `yaml:"context"`
	Namespace      string        `yaml:"namespace"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
	InCluster      bool          `yaml:"inCluster"`
}

// SchedulerConfig holds scheduler-related configuration.
type SchedulerConfig struct {
	SimulatorURL string        `yaml:"simulatorUrl"`
	HttpTimeout  time.Duration `yaml:"httpTimeout"`
}

// OutputConfig holds output-related configuration.
type OutputConfig struct {
	OutputDir   string `yaml:"outputDir"`
	Format      string `yaml:"format"`
	SaveMetrics bool   `yaml:"saveMetrics"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Scenario: ScenarioConfig{
			Name:      "scenario-default",
			OutputDir: "scenarios",
			Generation: GenerationConfig{
				NumPodEvents:        100,
				ArrivalScale:        2.0,
				ArrivalScaleFactor:  5.0,
				DurationScale:       1.5,
				DurationShape:       2.0,
				DurationScaleFactor: 10.0,
				PodCPUShape:         3.0,
				PodCPUFactor:        1.0,
				PodMemScale:         2.0,
				PodMemShape:         512.0,
				PodMemFactor:        1.0,
			},
		},
		Kubernetes: KubernetesConfig{
			Namespace:      "default",
			RequestTimeout: 30 * time.Second,
			InCluster:      false,
		},
		Scheduler: SchedulerConfig{
			SimulatorURL: getEnvOrDefault("SCHEDULER_SIM_URL", "http://localhost:1212/api/v1/schedulerconfiguration"),
			HttpTimeout:  10 * time.Second,
		},
		Output: OutputConfig{
			SaveMetrics: true,
			OutputDir:   "results",
			Format:      "csv",
		},
	}
}

// Load loads configuration from a file.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults.
	cfg := DefaultConfig()
	v.SetDefault("scenario", cfg.Scenario)
	v.SetDefault("kubernetes", cfg.Kubernetes)
	v.SetDefault("scheduler", cfg.Scheduler)
	v.SetDefault("output", cfg.Output)

	// Read config file.
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into struct.
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Scenario.Name == "" {
		return errors.NewValidationError("scenario.name", c.Scenario.Name, "cannot be empty")
	}

	if c.Scenario.Generation.NumPodEvents <= 0 {
		return errors.NewValidationError("scenario.generation.numPodEvents", c.Scenario.Generation.NumPodEvents, "must be positive")
	}

	if c.Scenario.Generation.ArrivalScale <= 0 {
		return errors.NewValidationError("scenario.generation.arrivalScale", c.Scenario.Generation.ArrivalScale, "must be positive")
	}

	if c.Output.Format != "csv" && c.Output.Format != "json" {
		return errors.NewValidationError("output.format", c.Output.Format, "must be 'csv' or 'json'")
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
