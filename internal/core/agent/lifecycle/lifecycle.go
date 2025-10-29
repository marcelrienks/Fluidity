package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fluidity/internal/shared/circuitbreaker"
	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/retry"
)

// Client manages ECS service lifecycle through Lambda APIs
type Client struct {
	config         *Config
	httpClient     *http.Client
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *logging.Logger
}

// WakeRequest represents the request to Wake Lambda
type WakeRequest struct {
	ClusterName string `json:"clusterName,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
}

// WakeResponse represents the response from Wake Lambda
type WakeResponse struct {
	StatusCode         int    `json:"statusCode"`
	Message            string `json:"message"`
	EstimatedStartTime string `json:"estimatedStartTime,omitempty"`
}

// KillRequest represents the request to Kill Lambda
type KillRequest struct {
	ClusterName string `json:"clusterName,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
}

// KillResponse represents the response from Kill Lambda
type KillResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// NewClient creates a new lifecycle management client
func NewClient(config *Config, logger *logging.Logger) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		logger = logging.NewLogger("lifecycle")
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	// Create circuit breaker for API calls
	cb := circuitbreaker.New(circuitbreaker.Config{
		MaxFailures:     3,
		ResetTimeout:    30 * time.Second,
		HalfOpenTimeout: 10 * time.Second,
		MaxHalfOpenReqs: 2,
	})

	return &Client{
		config:         config,
		httpClient:     httpClient,
		circuitBreaker: cb,
		logger:         logger,
	}, nil
}

// Wake calls the Wake Lambda to start the ECS service
func (c *Client) Wake(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("Lifecycle management disabled, skipping wake")
		return nil
	}

	c.logger.Info("Waking ECS service",
		"endpoint", c.config.WakeEndpoint,
		"cluster", c.config.ClusterName,
		"service", c.config.ServiceName,
	)

	// Prepare request body
	reqBody := WakeRequest{
		ClusterName: c.config.ClusterName,
		ServiceName: c.config.ServiceName,
	}

	// Call Wake API with retry
	var response *WakeResponse
	retryConfig := retry.Config{
		MaxAttempts:  c.config.MaxRetries,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	err := retry.Execute(ctx, retryConfig, retry.AlwaysRetry(), func() error {
		var err error
		response, err = c.callWakeAPI(ctx, reqBody)
		return err
	})

	if err != nil {
		c.logger.Error("Failed to wake ECS service", err)
		return fmt.Errorf("wake failed: %w", err)
	}

	c.logger.Info("ECS service wake successful",
		"message", response.Message,
		"estimatedStartTime", response.EstimatedStartTime,
	)

	return nil
}

// callWakeAPI makes the HTTP request to Wake Lambda
func (c *Client) callWakeAPI(ctx context.Context, reqBody WakeRequest) (*WakeResponse, error) {
	// Execute with circuit breaker
	var response *WakeResponse
	err := c.circuitBreaker.Execute(func() error {
		// Marshal request body
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", c.config.WakeEndpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.config.APIKey)

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode >= 400 {
			return fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(respBody))
		}

		// Parse response
		if err := json.Unmarshal(respBody, &response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Kill calls the Kill Lambda to stop the ECS service
func (c *Client) Kill(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("Lifecycle management disabled, skipping kill")
		return nil
	}

	c.logger.Info("Killing ECS service",
		"endpoint", c.config.KillEndpoint,
		"cluster", c.config.ClusterName,
		"service", c.config.ServiceName,
	)

	// Prepare request body
	reqBody := KillRequest{
		ClusterName: c.config.ClusterName,
		ServiceName: c.config.ServiceName,
	}

	// Call Kill API with retry
	var response *KillResponse
	retryConfig := retry.Config{
		MaxAttempts:  c.config.MaxRetries,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	err := retry.Execute(ctx, retryConfig, retry.AlwaysRetry(), func() error {
		var err error
		response, err = c.callKillAPI(ctx, reqBody)
		return err
	})

	if err != nil {
		c.logger.Error("Failed to kill ECS service", err)
		return fmt.Errorf("kill failed: %w", err)
	}

	c.logger.Info("ECS service kill successful", "message", response.Message)

	return nil
}

// callKillAPI makes the HTTP request to Kill Lambda
func (c *Client) callKillAPI(ctx context.Context, reqBody KillRequest) (*KillResponse, error) {
	// Execute with circuit breaker
	var response *KillResponse
	err := c.circuitBreaker.Execute(func() error {
		// Marshal request body
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", c.config.KillEndpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.config.APIKey)

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode >= 400 {
			return fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(respBody))
		}

		// Parse response
		if err := json.Unmarshal(respBody, &response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// WaitForConnection waits for the agent to establish server connection after wake
func (c *Client) WaitForConnection(ctx context.Context, checkFn func() bool) error {
	if !c.config.Enabled {
		return nil
	}

	c.logger.Info("Waiting for server connection",
		"timeout", c.config.ConnectionTimeout,
		"retryInterval", c.config.ConnectionRetryInterval,
	)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.ConnectionTimeout)
	defer cancel()

	ticker := time.NewTicker(c.config.ConnectionRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("connection timeout after %v", c.config.ConnectionTimeout)
		case <-ticker.C:
			if checkFn() {
				c.logger.Info("Server connection established")
				return nil
			}
			c.logger.Debug("Connection not ready, retrying...")
		}
	}
}
