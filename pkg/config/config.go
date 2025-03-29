package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	BskyID       string
	BskyPassword string
	BskyHost     string
}

func LoadConfig() Config {
	// Load from environment variables or use defaults
	bskyID := getEnv("BSKY_ID", "")
	bskyPassword := getEnv("BSKY_PASSWORD", "")
	bskyHost := getEnv("BSKY_HOST", "https://bsky.social")

	// Create config
	cfg := Config{
		BskyID:       bskyID,
		BskyPassword: bskyPassword,
		BskyHost:     bskyHost,
	}

	// Try to load config from file if BSKY_CONFIG_FILE is set
	if configFile := os.Getenv("BSKY_CONFIG_FILE"); configFile != "" {
		if fileCfg, err := loadConfigFromFile(configFile); err == nil {
			// Override with file values if they exist
			if fileCfg.BskyID != "" {
				cfg.BskyID = fileCfg.BskyID
			}
			if fileCfg.BskyPassword != "" {
				cfg.BskyPassword = fileCfg.BskyPassword
			}
			if fileCfg.BskyHost != "" {
				cfg.BskyHost = fileCfg.BskyHost
			}
		}
	}

	return cfg
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(path string) (Config, error) {
	var cfg Config

	// Expand path if needed
	expandedPath, err := filepath.Abs(path)
	if err != nil {
		return cfg, err
	}

	// Read and parse config file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg Config) error {
	if cfg.BskyHost == "" {
		return errors.New("missing Bluesky host in configuration")
	}

	// Authentication is required for most operations
	if cfg.BskyID == "" || cfg.BskyPassword == "" {
		return fmt.Errorf("missing Bluesky credentials in configuration")
	}

	return nil
}

// Helper function to get environment variable or default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
