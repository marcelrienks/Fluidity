package metrics

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds CloudWatch metrics configuration
type Config struct {
	// Region is the AWS region for CloudWatch
	Region string

	// Namespace is the CloudWatch namespace for custom metrics
	Namespace string

	// ServiceName is used as a dimension for metrics
	ServiceName string

	// ClusterName is used as a dimension for metrics
	ClusterName string

	// EmitInterval is how often to emit metrics
	EmitInterval time.Duration

	// Enabled indicates if metrics emission is enabled
	Enabled bool
}

// LoadConfig loads metrics configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Region:       getEnvOrDefault("AWS_REGION", "us-east-1"),
		Namespace:    getEnvOrDefault("METRICS_NAMESPACE", "Fluidity"),
		ServiceName:  getEnvOrDefault("ECS_SERVICE_NAME", "fluidity-server"),
		ClusterName:  getEnvOrDefault("ECS_CLUSTER_NAME", "fluidity"),
		EmitInterval: getEnvDuration("METRICS_EMIT_INTERVAL", 60*time.Second),
		Enabled:      getEnvBool("METRICS_ENABLED", true),
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Region == "" {
		return fmt.Errorf("AWS_REGION is required when metrics are enabled")
	}

	if c.Namespace == "" {
		return fmt.Errorf("METRICS_NAMESPACE is required when metrics are enabled")
	}

	if c.EmitInterval < 10*time.Second {
		return fmt.Errorf("METRICS_EMIT_INTERVAL must be at least 10 seconds")
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
