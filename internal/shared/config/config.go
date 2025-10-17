package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration with CLI override support
func LoadConfig[T any](configFile string, overrides map[string]interface{}) (*T, error) {
	// Initialize viper
	v := viper.New()
	
	// Set config file
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Look for config in current directory and home directory
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.fluidity")
	}
	
	// Set defaults
	setDefaults(v)
	
	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults and environment variables
	}
	
	// Apply CLI overrides
	for key, value := range overrides {
		if value != nil {
			v.Set(key, value)
		}
	}
	
	// Environment variable support
	v.AutomaticEnv()
	v.SetEnvPrefix("FLUIDITY")
	
	var config T
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// SaveConfig saves updated configuration
func SaveConfig(configFile string, config interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	v := viper.New()
	v.Set("config", config)
	return v.WriteConfigAs(configFile)
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Agent defaults
	v.SetDefault("agent.local_proxy_port", 8080)
	v.SetDefault("agent.server_port", 8443)
	v.SetDefault("agent.log_level", "info")
	v.SetDefault("agent.cert_file", "./certs/client.crt")
	v.SetDefault("agent.key_file", "./certs/client.key")
	v.SetDefault("agent.ca_cert_file", "./certs/ca.crt")
	
	// Server defaults
	v.SetDefault("server.listen_addr", "0.0.0.0")
	v.SetDefault("server.listen_port", 8443)
	v.SetDefault("server.log_level", "info")
	v.SetDefault("server.cert_file", "./certs/server.crt")
	v.SetDefault("server.key_file", "./certs/server.key")
	v.SetDefault("server.ca_cert_file", "./certs/ca.crt")
	v.SetDefault("server.max_connections", 100)
}