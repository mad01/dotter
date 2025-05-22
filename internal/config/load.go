package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// DefaultConfigFileName is the expected name of the configuration file.
const DefaultConfigFileName = "config.toml"

var (
	// GetDefaultConfigPath defines the function to get the default config path.
	// This is a variable to allow for easier testing.
	GetDefaultConfigPath = getDefaultConfigPathInternal
)

// LoadConfig attempts to load the dotter configuration from the default location.
// Default location: $XDG_CONFIG_HOME/dotter/config.toml or ~/.config/dotter/config.toml.
func LoadConfig() (*Config, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found at %s. Run 'dotter init' to create one", configPath)
	}

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file %s: %w", configPath, err)
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// getDefaultConfigPathInternal is the actual implementation for GetDefaultConfigPath.
func getDefaultConfigPathInternal() (string, error) {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(xdgConfigHome, "dotter", DefaultConfigFileName), nil
}
