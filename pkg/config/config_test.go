package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment variables
	origID := os.Getenv("BSKY_ID")
	origPassword := os.Getenv("BSKY_PASSWORD")
	origHost := os.Getenv("BSKY_HOST")
	origConfigFile := os.Getenv("BSKY_CONFIG_FILE")

	// Clean up after test
	defer func() {
		os.Setenv("BSKY_ID", origID)
		os.Setenv("BSKY_PASSWORD", origPassword)
		os.Setenv("BSKY_HOST", origHost)
		os.Setenv("BSKY_CONFIG_FILE", origConfigFile)
	}()

	// Test default values when environment variables are not set
	os.Unsetenv("BSKY_ID")
	os.Unsetenv("BSKY_PASSWORD")
	os.Unsetenv("BSKY_HOST")
	os.Unsetenv("BSKY_CONFIG_FILE")

	cfg := LoadConfig()
	if cfg.BskyID != "" {
		t.Errorf("Expected empty BskyID by default, got %s", cfg.BskyID)
	}
	if cfg.BskyPassword != "" {
		t.Errorf("Expected empty BskyPassword by default, got %s", cfg.BskyPassword)
	}
	if cfg.BskyHost != "https://bsky.social" {
		t.Errorf("Expected default BskyHost to be https://bsky.social, got %s", cfg.BskyHost)
	}

	// Test loading from environment variables
	os.Setenv("BSKY_ID", "test-id")
	os.Setenv("BSKY_PASSWORD", "test-password")
	os.Setenv("BSKY_HOST", "https://test.bsky.social")

	cfg = LoadConfig()
	if cfg.BskyID != "test-id" {
		t.Errorf("Expected BskyID to be test-id, got %s", cfg.BskyID)
	}
	if cfg.BskyPassword != "test-password" {
		t.Errorf("Expected BskyPassword to be test-password, got %s", cfg.BskyPassword)
	}
	if cfg.BskyHost != "https://test.bsky.social" {
		t.Errorf("Expected BskyHost to be https://test.bsky.social, got %s", cfg.BskyHost)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file for testing
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "test_config.json")

	testConfig := Config{
		BskyID:       "file-id",
		BskyPassword: "file-password",
		BskyHost:     "https://file.bsky.social",
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(testConfigFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading from file
	cfg, err := loadConfigFromFile(testConfigFile)
	if err != nil {
		t.Errorf("loadConfigFromFile failed: %v", err)
	}

	if cfg.BskyID != "file-id" {
		t.Errorf("Expected BskyID to be file-id, got %s", cfg.BskyID)
	}
	if cfg.BskyPassword != "file-password" {
		t.Errorf("Expected BskyPassword to be file-password, got %s", cfg.BskyPassword)
	}
	if cfg.BskyHost != "https://file.bsky.social" {
		t.Errorf("Expected BskyHost to be https://file.bsky.social, got %s", cfg.BskyHost)
	}

	// Test loading from nonexistent file
	_, err = loadConfigFromFile(filepath.Join(tmpDir, "nonexistent.json"))
	if err == nil {
		t.Error("Expected error when loading from nonexistent file, got nil")
	}

	// Test loading from invalid JSON file
	invalidConfigFile := filepath.Join(tmpDir, "invalid_config.json")
	if err := os.WriteFile(invalidConfigFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err = loadConfigFromFile(invalidConfigFile)
	if err == nil {
		t.Error("Expected error when loading invalid JSON, got nil")
	}
}

func TestLoadConfigWithFileOverride(t *testing.T) {
	// Save original environment variables
	origID := os.Getenv("BSKY_ID")
	origPassword := os.Getenv("BSKY_PASSWORD")
	origHost := os.Getenv("BSKY_HOST")
	origConfigFile := os.Getenv("BSKY_CONFIG_FILE")

	// Clean up after test
	defer func() {
		os.Setenv("BSKY_ID", origID)
		os.Setenv("BSKY_PASSWORD", origPassword)
		os.Setenv("BSKY_HOST", origHost)
		os.Setenv("BSKY_CONFIG_FILE", origConfigFile)
	}()

	// Set environment variables
	os.Setenv("BSKY_ID", "env-id")
	os.Setenv("BSKY_PASSWORD", "env-password")
	os.Setenv("BSKY_HOST", "https://env.bsky.social")

	// Create a temporary config file for testing
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "test_config.json")

	testConfig := Config{
		BskyID:       "file-id",
		BskyPassword: "file-password",
		BskyHost:     "https://file.bsky.social",
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(testConfigFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Set config file environment variable
	os.Setenv("BSKY_CONFIG_FILE", testConfigFile)

	// File should override environment variables
	cfg := LoadConfig()

	if cfg.BskyID != "file-id" {
		t.Errorf("Expected BskyID to be file-id, got %s", cfg.BskyID)
	}
	if cfg.BskyPassword != "file-password" {
		t.Errorf("Expected BskyPassword to be file-password, got %s", cfg.BskyPassword)
	}
	if cfg.BskyHost != "https://file.bsky.social" {
		t.Errorf("Expected BskyHost to be https://file.bsky.social, got %s", cfg.BskyHost)
	}

	// Create a config file with partial values
	partialConfig := Config{
		BskyID: "partial-id",
		// Other fields not set
	}

	partialData, err := json.Marshal(partialConfig)
	if err != nil {
		t.Fatalf("Failed to marshal partial config: %v", err)
	}

	partialConfigFile := filepath.Join(tmpDir, "partial_config.json")
	if err := os.WriteFile(partialConfigFile, partialData, 0644); err != nil {
		t.Fatalf("Failed to write partial config file: %v", err)
	}

	// Set to partial config file
	os.Setenv("BSKY_CONFIG_FILE", partialConfigFile)

	// Partial file should override only some values
	cfg = LoadConfig()

	if cfg.BskyID != "partial-id" {
		t.Errorf("Expected BskyID to be partial-id, got %s", cfg.BskyID)
	}
	if cfg.BskyPassword != "env-password" {
		t.Errorf("Expected BskyPassword to be env-password, got %s", cfg.BskyPassword)
	}
	if cfg.BskyHost != "https://env.bsky.social" {
		t.Errorf("Expected BskyHost to be https://env.bsky.social, got %s", cfg.BskyHost)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "Valid config",
			config: Config{
				BskyID:       "test-id",
				BskyPassword: "test-password",
				BskyHost:     "https://bsky.social",
			},
			wantError: false,
		},
		{
			name: "Missing host",
			config: Config{
				BskyID:       "test-id",
				BskyPassword: "test-password",
				BskyHost:     "",
			},
			wantError: true,
		},
		{
			name: "Missing ID",
			config: Config{
				BskyID:       "",
				BskyPassword: "test-password",
				BskyHost:     "https://bsky.social",
			},
			wantError: true,
		},
		{
			name: "Missing password",
			config: Config{
				BskyID:       "test-id",
				BskyPassword: "",
				BskyHost:     "https://bsky.social",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Save original environment variable
	origValue := os.Getenv("TEST_ENV_VAR")
	defer os.Setenv("TEST_ENV_VAR", origValue)

	// Test with unset variable
	os.Unsetenv("TEST_ENV_VAR")
	if value := getEnv("TEST_ENV_VAR", "default"); value != "default" {
		t.Errorf("Expected default value when env var is unset, got %s", value)
	}

	// Test with set variable
	os.Setenv("TEST_ENV_VAR", "test-value")
	if value := getEnv("TEST_ENV_VAR", "default"); value != "test-value" {
		t.Errorf("Expected test-value when env var is set, got %s", value)
	}
}