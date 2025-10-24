package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestAgentConfig for testing agent configuration
type TestAgentConfig struct {
	Agent struct {
		LocalProxyPort int    `mapstructure:"local_proxy_port"`
		ServerAddr     string `mapstructure:"server_addr"`
		ServerPort     int    `mapstructure:"server_port"`
		LogLevel       string `mapstructure:"log_level"`
		CertFile       string `mapstructure:"cert_file"`
		KeyFile        string `mapstructure:"key_file"`
		CACertFile     string `mapstructure:"ca_cert_file"`
	} `mapstructure:"agent"`
}

// TestServerConfig for testing server configuration
type TestServerConfig struct {
	Server struct {
		ListenAddr     string `mapstructure:"listen_addr"`
		ListenPort     int    `mapstructure:"listen_port"`
		LogLevel       string `mapstructure:"log_level"`
		CertFile       string `mapstructure:"cert_file"`
		KeyFile        string `mapstructure:"key_file"`
		CACertFile     string `mapstructure:"ca_cert_file"`
		MaxConnections int    `mapstructure:"max_connections"`
	} `mapstructure:"server"`
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Load config without file (should use defaults)
	config, err := LoadConfig[TestAgentConfig]("", nil)
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Verify defaults
	if config.Agent.LocalProxyPort != 8080 {
		t.Errorf("Expected default local_proxy_port 8080, got %d", config.Agent.LocalProxyPort)
	}
	if config.Agent.LogLevel != "info" {
		t.Errorf("Expected default log_level 'info', got '%s'", config.Agent.LogLevel)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configData := `
agent:
  local_proxy_port: 9090
  server_addr: "test.example.com"
  server_port: 8443
  log_level: "debug"
  cert_file: "./certs/test-client.crt"
  key_file: "./certs/test-client.key"
  ca_cert_file: "./certs/test-ca.crt"
`

	err := os.WriteFile(configFile, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load config from file
	config, err := LoadConfig[TestAgentConfig](configFile, nil)
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Verify values from file
	if config.Agent.LocalProxyPort != 9090 {
		t.Errorf("Expected local_proxy_port 9090, got %d", config.Agent.LocalProxyPort)
	}
	if config.Agent.ServerAddr != "test.example.com" {
		t.Errorf("Expected server_addr 'test.example.com', got '%s'", config.Agent.ServerAddr)
	}
	if config.Agent.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", config.Agent.LogLevel)
	}
}

func TestLoadConfigWithOverrides(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configData := `
agent:
  local_proxy_port: 8080
  server_addr: "original.example.com"
  log_level: "info"
`

	err := os.WriteFile(configFile, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Override some values
	overrides := map[string]interface{}{
		"agent.server_addr": "override.example.com",
		"agent.log_level":   "warn",
	}

	config, err := LoadConfig[TestAgentConfig](configFile, overrides)
	if err != nil {
		t.Fatalf("Failed to load config with overrides: %v", err)
	}

	// Verify overridden values
	if config.Agent.ServerAddr != "override.example.com" {
		t.Errorf("Expected overridden server_addr 'override.example.com', got '%s'", config.Agent.ServerAddr)
	}
	if config.Agent.LogLevel != "warn" {
		t.Errorf("Expected overridden log_level 'warn', got '%s'", config.Agent.LogLevel)
	}

	// Verify non-overridden values remain from file
	if config.Agent.LocalProxyPort != 8080 {
		t.Errorf("Expected local_proxy_port 8080, got %d", config.Agent.LocalProxyPort)
	}
}

func TestLoadConfigServerDefaults(t *testing.T) {
	config, err := LoadConfig[TestServerConfig]("", nil)
	if err != nil {
		t.Fatalf("Failed to load server config: %v", err)
	}

	// Verify server defaults
	if config.Server.ListenAddr != "0.0.0.0" {
		t.Errorf("Expected default listen_addr '0.0.0.0', got '%s'", config.Server.ListenAddr)
	}
	if config.Server.ListenPort != 8443 {
		t.Errorf("Expected default listen_port 8443, got %d", config.Server.ListenPort)
	}
	if config.Server.MaxConnections != 100 {
		t.Errorf("Expected default max_connections 100, got %d", config.Server.MaxConnections)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "output.yaml")

	// Create config
	var config TestAgentConfig
	config.Agent.LocalProxyPort = 7777
	config.Agent.ServerAddr = "save-test.example.com"
	config.Agent.LogLevel = "error"

	// Save config
	err := SaveConfig(configFile, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Read back and verify
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded TestAgentConfig
	err = yaml.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if loaded.Agent.LocalProxyPort != 7777 {
		t.Errorf("Expected saved local_proxy_port 7777, got %d", loaded.Agent.LocalProxyPort)
	}
	if loaded.Agent.ServerAddr != "save-test.example.com" {
		t.Errorf("Expected saved server_addr 'save-test.example.com', got '%s'", loaded.Agent.ServerAddr)
	}
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

	var config TestAgentConfig
	config.Agent.LocalProxyPort = 8888

	// Save config (should create directories)
	err := SaveConfig(configFile, config)
	if err != nil {
		t.Fatalf("Failed to save config with directory creation: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created in nested directory")
	}

	// Verify directory was created
	dir := filepath.Dir(configFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("Nested directory was not created")
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create invalid YAML
	invalidYAML := `
agent:
  local_proxy_port: not_a_number
  server_addr: [invalid
`
	os.WriteFile(configFile, []byte(invalidYAML), 0644)

	// Try to load - should fail
	_, err := LoadConfig[TestAgentConfig](configFile, nil)
	if err == nil {
		t.Fatal("Expected error when loading invalid config, got nil")
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	// Try to load from specific non-existent file
	_, err := LoadConfig[TestAgentConfig]("/nonexistent/path/config.yaml", nil)

	// Should return error for non-existent file
	if err == nil {
		t.Fatal("Expected error for non-existent config file, got nil")
	}
}

func TestLoadConfigWithNilOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configData := `
agent:
  local_proxy_port: 5555
`
	os.WriteFile(configFile, []byte(configData), 0644)

	// Pass overrides with nil value (should be ignored)
	overrides := map[string]interface{}{
		"agent.local_proxy_port": nil,
		"agent.log_level":        "warn",
	}

	config, err := LoadConfig[TestAgentConfig](configFile, overrides)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Nil override should be ignored, value from file should remain
	if config.Agent.LocalProxyPort != 5555 {
		t.Errorf("Expected local_proxy_port 5555 (nil override ignored), got %d", config.Agent.LocalProxyPort)
	}

	// Non-nil override should apply
	if config.Agent.LogLevel != "warn" {
		t.Errorf("Expected log_level 'warn', got '%s'", config.Agent.LogLevel)
	}
}
