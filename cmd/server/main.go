package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	serverConfig "fluidity/internal/server/config"
	"fluidity/internal/server/tunnel"
	"fluidity/internal/shared/config"
	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/tls"
)

var (
	configFile     string
	listenAddr     string
	listenPort     int
	maxConnections int
	logLevel       string
	certFile       string
	keyFile        string
	caCertFile     string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fluidity-server",
		Short: "Fluidity tunnel server",
		Long:  "Fluidity tunnel server - Accepts secure connections from agents and forwards HTTP requests",
		RunE:  runServer,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().StringVar(&listenAddr, "listen-addr", "", "Address to listen on")
	rootCmd.Flags().IntVar(&listenPort, "listen-port", 0, "Port to listen on")
	rootCmd.Flags().IntVar(&maxConnections, "max-connections", 0, "Maximum number of concurrent connections")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&certFile, "cert", "", "Server certificate file")
	rootCmd.Flags().StringVar(&keyFile, "key", "", "Server private key file")
	rootCmd.Flags().StringVar(&caCertFile, "ca", "", "CA certificate file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	// Create logger
	logger := logging.NewLogger("server")

	// Build configuration overrides from CLI flags
	overrides := make(map[string]interface{})
	if listenAddr != "" {
		overrides["listen_addr"] = listenAddr
	}
	if listenPort != 0 {
		overrides["listen_port"] = listenPort
	}
	if maxConnections != 0 {
		overrides["max_connections"] = maxConnections
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
	cfg, err := config.LoadConfig[serverConfig.Config](configFile, overrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set log level
	logger.SetLevel(cfg.LogLevel)

	logger.Info("Starting Fluidity tunnel server",
		"listen_addr", cfg.GetListenAddress(),
		"max_connections", cfg.MaxConnections,
		"log_level", cfg.LogLevel)

	// Load TLS configuration
	tlsConfig, err := tls.LoadServerTLSConfig(cfg.CertFile, cfg.KeyFile, cfg.CACertFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS configuration: %w", err)
	}

	logger.Info("Loaded TLS configuration",
		"cert_file", cfg.CertFile,
		"key_file", cfg.KeyFile,
		"ca_file", cfg.CACertFile)

	// Create tunnel server
	tunnelServer, err := tunnel.NewServer(tlsConfig, cfg.GetListenAddress(), cfg.MaxConnections)
	if err != nil {
		return fmt.Errorf("failed to create tunnel server: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := tunnelServer.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	logger.Info("Tunnel server started successfully")

	// Wait for shutdown signal or server error
	select {
	case <-sigChan:
		logger.Info("Shutdown signal received, stopping server...")
	case err := <-serverErrChan:
		logger.Error("Server error", err)
		return err
	}

	// Graceful shutdown
	cancel()

	// Stop server
	if err := tunnelServer.Stop(); err != nil {
		logger.Error("Error stopping tunnel server", err)
		return err
	}

	logger.Info("Server stopped")
	return nil
}