package tunnel

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/protocol"

	"github.com/sirupsen/logrus"
) // Client manages the tunnel connection to server
type Client struct {
	config      *tls.Config
	serverAddr  string
	conn        *tls.Conn
	mu          sync.RWMutex
	requests    map[string]chan *protocol.Response
	logger      *logging.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	connected   bool
	reconnectCh chan bool
}

// NewClient creates a new tunnel client
func NewClient(tlsConfig *tls.Config, serverAddr string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:      tlsConfig,
		serverAddr:  serverAddr,
		requests:    make(map[string]chan *protocol.Response),
		logger:      logging.NewLogger("tunnel-client"),
		ctx:         ctx,
		cancel:      cancel,
		reconnectCh: make(chan bool, 1),
	}
}

// Connect establishes mTLS connection to server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	c.logger.Info("Connecting to tunnel server", "addr", c.serverAddr)

	// Extract hostname for ServerName
	host := c.extractHost(c.serverAddr)

	// Create TLS config with client certificate
	tlsConfig := &tls.Config{
		Certificates: c.config.Certificates,
		RootCAs:      c.config.RootCAs,
		MinVersion:   c.config.MinVersion,
		ServerName:   host, // CRITICAL: Set ServerName for proper mTLS handshake
	}

	c.logger.WithFields(logrus.Fields{
		"num_certificates": len(tlsConfig.Certificates),
		"has_root_cas":     tlsConfig.RootCAs != nil,
		"server_name":      tlsConfig.ServerName,
	}).Info("TLS config for dial")

	conn, err := tls.Dial("tcp", c.serverAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Log the connection state
	state := conn.ConnectionState()
	c.logger.WithFields(logrus.Fields{
		"version":            state.Version,
		"cipher_suite":       state.CipherSuite,
		"peer_certificates":  len(state.PeerCertificates),
		"local_certificates": len(tlsConfig.Certificates),
	}).Info("TLS connection established")

	c.conn = conn
	c.connected = true
	c.logger.Info("Connected to tunnel server", "addr", c.serverAddr)

	// Start response handler
	go c.handleResponses()

	return nil
}

// Disconnect closes the connection to the server
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	c.cancel()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.logger.Info("Disconnected from tunnel server")
	return nil
}

// SendRequest sends request through tunnel and waits for response
func (c *Client) SendRequest(req *protocol.Request) (*protocol.Response, error) {
	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected to server")
	}
	conn := c.conn
	c.mu.RUnlock()

	// Create response channel
	respChan := make(chan *protocol.Response, 1)
	c.mu.Lock()
	c.requests[req.ID] = respChan
	c.mu.Unlock()

	// Send request
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		c.mu.Lock()
		delete(c.requests, req.ID)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	c.logger.Debug("Sent request through tunnel", "id", req.ID, "url", req.URL)

	// Wait for response
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.requests, req.ID)
		c.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	case <-c.ctx.Done():
		c.mu.Lock()
		delete(c.requests, req.ID)
		c.mu.Unlock()
		return nil, fmt.Errorf("connection closed")
	}
}

// handleResponses processes responses from the server
func (c *Client) handleResponses() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		// Close all pending request channels
		for id, ch := range c.requests {
			close(ch)
			delete(c.requests, id)
		}
		c.mu.Unlock()

		// Signal reconnection needed
		select {
		case c.reconnectCh <- true:
		default:
		}
	}()

	decoder := json.NewDecoder(c.conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		var resp protocol.Response
		if err := decoder.Decode(&resp); err != nil {
			c.logger.Error("Failed to decode response", err)
			return
		}

		c.logger.Debug("Received response from tunnel", "id", resp.ID, "status", resp.StatusCode)

		c.mu.RLock()
		respChan, exists := c.requests[resp.ID]
		c.mu.RUnlock()

		if exists {
			select {
			case respChan <- &resp:
			case <-time.After(1 * time.Second):
				c.logger.Warn("Response channel blocked", "id", resp.ID)
			}

			c.mu.Lock()
			delete(c.requests, resp.ID)
			c.mu.Unlock()
		} else {
			c.logger.Warn("Received response for unknown request", "id", resp.ID)
		}
	}
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// ReconnectChannel returns a channel that signals when reconnection is needed
func (c *Client) ReconnectChannel() <-chan bool {
	return c.reconnectCh
}

// extractHost extracts the host part from an address
func (c *Client) extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
