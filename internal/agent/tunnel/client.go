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
	connectCh   map[string]chan *protocol.ConnectData
	connectAcks map[string]chan *protocol.ConnectAck
	logger      *logging.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	connected   bool
	reconnectCh chan bool
}

// NewClient creates a new tunnel client
func NewClient(tlsConfig *tls.Config, serverAddr string, logLevel string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	logger := logging.NewLogger("tunnel-client")
	logger.SetLevel(logLevel)

	return &Client{
		config:      tlsConfig,
		serverAddr:  serverAddr,
		requests:    make(map[string]chan *protocol.Response),
		connectCh:   make(map[string]chan *protocol.ConnectData),
		connectAcks: make(map[string]chan *protocol.ConnectAck),
		logger:      logger,
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

	// Send request wrapped in Envelope
	encoder := json.NewEncoder(conn)
	env := protocol.Envelope{Type: "http_request", Payload: req}
	if err := encoder.Encode(env); err != nil {
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

		var env protocol.Envelope
		if err := decoder.Decode(&env); err != nil {
			c.logger.Error("Failed to decode envelope", err)
			return
		}

		switch env.Type {
		case "http_response":
			// Parse payload as Response
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var resp protocol.Response
			if err := json.Unmarshal(b, &resp); err != nil {
				c.logger.Error("Failed to parse http_response", err)
				continue
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

		case "connect_ack":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var ack protocol.ConnectAck
			if err := json.Unmarshal(b, &ack); err != nil {
				c.logger.Error("Failed to parse connect_ack", err)
				continue
			}
			c.mu.RLock()
			ackCh := c.connectAcks[ack.ID]
			c.mu.RUnlock()
			if ackCh != nil {
				select {
				case ackCh <- &ack:
				case <-time.After(1 * time.Second):
					c.logger.Warn("Connect ack channel blocked", "id", ack.ID)
				}
			}

		case "connect_data":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var data protocol.ConnectData
			if err := json.Unmarshal(b, &data); err != nil {
				c.logger.Error("Failed to parse connect_data", err)
				continue
			}
			c.mu.RLock()
			ch := c.connectCh[data.ID]
			c.mu.RUnlock()
			if ch != nil {
				select {
				case ch <- &data:
				default:
					// Channel full, drop packet (backpressure)
				}
			}

		case "connect_close":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var cls protocol.ConnectClose
			if err := json.Unmarshal(b, &cls); err != nil {
				continue
			}
			c.mu.Lock()
			if ch := c.connectCh[cls.ID]; ch != nil {
				close(ch)
				delete(c.connectCh, cls.ID)
			}
			c.mu.Unlock()

		default:
			// Ignore unknown message types
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

// ConnectOpen requests a TCP tunnel to host:port
func (c *Client) ConnectOpen(id, address string) (*protocol.ConnectAck, error) {
	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected to server")
	}
	conn := c.conn
	c.mu.RUnlock()

	// Prepare channels for this connection
	ackCh := make(chan *protocol.ConnectAck, 1)
	c.mu.Lock()
	c.connectAcks[id] = ackCh
	if _, exists := c.connectCh[id]; !exists {
		c.connectCh[id] = make(chan *protocol.ConnectData, 64)
	}
	c.mu.Unlock()

	env := protocol.Envelope{Type: "connect_open", Payload: &protocol.ConnectOpen{ID: id, Address: address}}
	if err := json.NewEncoder(conn).Encode(env); err != nil {
		c.mu.Lock()
		delete(c.connectAcks, id)
		delete(c.connectCh, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to send connect_open: %w", err)
	}

	// Wait for real ack from server
	select {
	case ack := <-ackCh:
		c.mu.Lock()
		delete(c.connectAcks, id)
		c.mu.Unlock()
		return ack, nil
	case <-time.After(10 * time.Second):
		c.mu.Lock()
		delete(c.connectAcks, id)
		delete(c.connectCh, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for connect_ack")
	case <-c.ctx.Done():
		c.mu.Lock()
		delete(c.connectAcks, id)
		delete(c.connectCh, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("connection closed")
	}
}

// ConnectSend sends a data chunk over the tunnel
func (c *Client) ConnectSend(id string, chunk []byte) error {
	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("not connected to server")
	}
	conn := c.conn
	c.mu.RUnlock()
	env := protocol.Envelope{Type: "connect_data", Payload: &protocol.ConnectData{ID: id, Chunk: chunk}}
	return json.NewEncoder(conn).Encode(env)
}

// ConnectClose closes a tunnel stream
func (c *Client) ConnectClose(id, errMsg string) error {
	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil
	}
	conn := c.conn
	c.mu.RUnlock()
	env := protocol.Envelope{Type: "connect_close", Payload: &protocol.ConnectClose{ID: id, Error: errMsg}}
	return json.NewEncoder(conn).Encode(env)
}

// ConnectDataChannel returns the data channel for a given tunnel id
func (c *Client) ConnectDataChannel(id string) <-chan *protocol.ConnectData {
	c.mu.RLock()
	ch := c.connectCh[id]
	c.mu.RUnlock()
	return ch
}
