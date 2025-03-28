package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Host string `json:"host"`
	Port int    `json:"port"`

	// API configuration
	APIHost string `json:"api_host"`

	// Database configuration
	DatabasePath string `json:"database_path"`

	// API key usage settings
	DefaultQuotaLimit int `json:"default_quota_limit"`
	CostPerRequest    int `json:"cost_per_request"`
}

// Default configuration values
const (
	DefaultHost           = "localhost"
	DefaultPort           = 8080
	DefaultAPIHost        = "https://api.shodan.io"
	DefaultDatabaseDir    = "./data"
	DefaultQuotaLimit     = 100
	DefaultCostPerRequest = 0
)

// New creates a new configuration
func New() (*Config, error) {
	// Set default configuration
	cfg := &Config{
		Host:              DefaultHost,
		Port:              DefaultPort,
		APIHost:           DefaultAPIHost,
		DatabasePath:      filepath.Join(DefaultDatabaseDir, "proxy.db"),
		DefaultQuotaLimit: DefaultQuotaLimit,
		CostPerRequest:    DefaultCostPerRequest,
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(DefaultDatabaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Try to load config from file
	if err := cfg.loadFromFile(); err != nil {
		// If file doesn't exist, save the default config
		if os.IsNotExist(err) {
			if err := cfg.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return cfg, nil
}

// configFilePath returns the path to the config file
func (c *Config) configFilePath() string {
	return filepath.Join(DefaultDatabaseDir, "config.json")
}

// loadFromFile loads the configuration from a file
func (c *Config) loadFromFile() error {
	file, err := os.Open(c.configFilePath())
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}

// Save saves the configuration to a file
func (c *Config) Save() error {
	file, err := os.Create(c.configFilePath())
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}
