package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Events   EventsConfig   `yaml:"events"`
	LLM      LLMConfig      `yaml:"llm"`
	Slack    SlackConfig    `yaml:"slack"`
	Analyzer AnalyzerConfig `yaml:"analyzer"`
}

// EventsConfig defines which events to monitor
type EventsConfig struct {
	CrashLoopBackOff  bool `yaml:"crashLoopBackOff"`
	ImagePullBackOff  bool `yaml:"imagePullBackOff"`
	HealthCheckFailure bool `yaml:"healthCheckFailure"`
	OOMKilled         bool `yaml:"oomKilled"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider  string            `yaml:"provider"`
	Model     map[string]string `yaml:"model"`
	MaxTokens int               `yaml:"maxTokens"`
}

// SlackConfig contains Slack notification settings
type SlackConfig struct {
	Enabled bool   `yaml:"enabled"`
	Channel string `yaml:"channel"`
}

// AnalyzerConfig contains analyzer job settings
type AnalyzerConfig struct {
	Image                    string            `yaml:"image"`
	TTLSecondsAfterFinished int32             `yaml:"ttlSecondsAfterFinished"`
	Resources               ResourcesConfig   `yaml:"resources"`
}

// ResourcesConfig defines resource requests and limits
type ResourcesConfig struct {
	Requests ResourceSpec `yaml:"requests"`
	Limits   ResourceSpec `yaml:"limits"`
}

// ResourceSpec defines CPU and memory
type ResourceSpec struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
