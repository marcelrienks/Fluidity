package config

import "fmt"

// Config holds agent configuration
type Config struct {
	ServerIP       string `mapstructure:"server_ip" yaml:"server_ip"`
	ServerPort     int    `mapstructure:"server_port" yaml:"server_port"`
	LocalProxyPort int    `mapstructure:"local_proxy_port" yaml:"local_proxy_port"`
	CertFile       string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile        string `mapstructure:"key_file" yaml:"key_file"`
	CACertFile     string `mapstructure:"ca_cert_file" yaml:"ca_cert_file"`
	LogLevel       string `mapstructure:"log_level" yaml:"log_level"`
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerIP, c.ServerPort)
}