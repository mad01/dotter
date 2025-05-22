package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// DefaultConfigFileName is the expected name of the configuration file.
const DefaultConfigFileName = "config.toml"

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

// GetDefaultConfigPath returns the default path for the dotter configuration file.
func GetDefaultConfigPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
		configHome = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configHome, "dotter", DefaultConfigFileName), nil
}
