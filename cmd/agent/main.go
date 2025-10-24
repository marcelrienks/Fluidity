package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	agentConfig "fluidity/internal/agent/config"
	"fluidity/internal/agent/proxy"
	"fluidity/internal/agent/tunnel"
	"fluidity/internal/shared/config"
	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/tls"
)

var (
	configFile string
	serverIP   string
	serverPort int
	proxyPort  int
	logLevel   string
	certFile   string
	keyFile    string
	caCertFile string
)

func main() {
	// Note: GODEBUG must be set BEFORE the Go runtime initializes
	// Use run-agent-debug.cmd to launch with TLS debug logging enabled

	rootCmd := &cobra.Command{
		Use:   "fluidity-agent",
		Short: "Fluidity tunnel agent",
		Long:  "Fluidity tunnel agent - HTTP proxy that forwards traffic through secure tunnel",
		RunE:  runAgent,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().StringVar(&serverIP, "server-ip", "", "Tunnel server IP address")
	rootCmd.Flags().IntVar(&serverPort, "server-port", 0, "Tunnel server port")
	rootCmd.Flags().IntVar(&proxyPort, "proxy-port", 0, "Local proxy port")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&certFile, "cert", "", "Client certificate file")
	rootCmd.Flags().StringVar(&keyFile, "key", "", "Client private key file")
	rootCmd.Flags().StringVar(&caCertFile, "ca", "", "CA certificate file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Create logger
	logger := logging.NewLogger("agent")

	// Build configuration overrides from CLI flags
	overrides := make(map[string]interface{})
	if serverIP != "" {
		overrides["server_ip"] = serverIP
	}
	if serverPort != 0 {
		overrides["server_port"] = serverPort
	}
	if proxyPort != 0 {
		overrides["local_proxy_port"] = proxyPort
	}
	if logLevel != "" {
		overrides["log_level"] = logLevel
	}
	if certFile != "" {
		overrides["cert_file"] = certFile
	}
	if keyFile != "" {
		overrides["key_file"] = keyFile
	}
	if caCertFile != "" {
		overrides["ca_cert_file"] = caCertFile
	}

	// Load configuration
	cfg, err := config.LoadConfig[agentConfig.Config](configFile, overrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set log level
	logger.SetLevel(cfg.LogLevel)

	logger.Info("Starting Fluidity tunnel agent",
		"server", cfg.GetServerAddress(),
		"proxy_port", cfg.LocalProxyPort,
		"log_level", cfg.LogLevel)

	// Validate required configuration
	if cfg.ServerIP == "" {
		return fmt.Errorf("server IP address is required (use --server-ip or config file)")
	}

	// Save updated configuration if server IP was provided via CLI
	if serverIP != "" && configFile != "" {
		if err := config.SaveConfig(configFile, cfg); err != nil {
			logger.Warn("Failed to save updated configuration", "error", err.Error())
		} else {
			logger.Info("Updated configuration saved", "file", configFile)
		}
	}

	// Load TLS configuration
	tlsConfig, err := tls.LoadClientTLSConfig(cfg.CertFile, cfg.KeyFile, cfg.CACertFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS configuration: %w", err)
	}

	logger.Info("Loaded TLS configuration",
		"cert_file", cfg.CertFile,
		"key_file", cfg.KeyFile,
		"ca_file", cfg.CACertFile)

	// Create tunnel client
	tunnelClient := tunnel.NewClient(tlsConfig, cfg.GetServerAddress(), cfg.LogLevel)

	// Create proxy server
	proxyServer := proxy.NewServer(cfg.LocalProxyPort, tunnelClient, cfg.LogLevel)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start proxy server
	if err := proxyServer.Start(); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	// Connection management goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Connect to tunnel server
			if !tunnelClient.IsConnected() {
				logger.Info("Attempting to connect to tunnel server")
				if err := tunnelClient.Connect(); err != nil {
					logger.Error("Failed to connect to tunnel server", err)
					time.Sleep(5 * time.Second)
					continue
				}
			}

			// Wait for disconnection or shutdown
			select {
			case <-tunnelClient.ReconnectChannel():
				logger.Warn("Connection lost, will attempt to reconnect")
				time.Sleep(2 * time.Second)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, stopping agent...")

	// Graceful shutdown
	cancel()

	// Stop proxy server
	if err := proxyServer.Stop(); err != nil {
		logger.Error("Error stopping proxy server", err)
	}

	// Disconnect tunnel client
	if err := tunnelClient.Disconnect(); err != nil {
		logger.Error("Error disconnecting tunnel client", err)
	}

	logger.Info("Agent stopped")
	return nil
}
