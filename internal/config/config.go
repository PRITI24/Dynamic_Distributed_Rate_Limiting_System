package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Configuration represents the main configuration structure
type Configuration struct {
	RateLimits []RateLimit `yaml:"rateLimits"`
}

// RateLimit represents rate limiting configuration for an API key
type RateLimit struct {
	APIKey    string           `yaml:"apiKey"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

// EndpointConfig represents configuration for a specific endpoint
type EndpointConfig struct {
	Path string `yaml:"path"`
	RPM  int    `yaml:"rpm"`
	TPM  int    `yaml:"tpm"`
}

// Load reads and parses the configuration file
func Load(filepath string) (*Configuration, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
