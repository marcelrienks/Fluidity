package server

import "fmt"

// Config holds server configuration
type Config struct {
	ListenAddr         string `mapstructure:"listen_addr" yaml:"listen_addr"`
	ListenPort         int    `mapstructure:"listen_port" yaml:"listen_port"`
	CertFile           string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile            string `mapstructure:"key_file" yaml:"key_file"`
	CACertFile         string `mapstructure:"ca_cert_file" yaml:"ca_cert_file"`
	LogLevel           string `mapstructure:"log_level" yaml:"log_level"`
	MaxConnections     int    `mapstructure:"max_connections" yaml:"max_connections"`
	SecretsManagerName string `mapstructure:"secrets_manager_name" yaml:"secrets_manager_name"`
	UseSecretsManager  bool   `mapstructure:"use_secrets_manager" yaml:"use_secrets_manager"`
}

// GetListenAddress returns the full listen address
func (c *Config) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.ListenPort)
}
