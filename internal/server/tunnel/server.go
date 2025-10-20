package tunnel

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/protocol"
	tlsutil "fluidity/internal/shared/tls"
)

// Server handles mTLS connections from agents
type Server struct {
	listener    net.Listener
	httpClient  *http.Client
	logger      *logging.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	maxConns    int
	activeConns int32
	connMutex   sync.RWMutex
	tcpConns    map[string]net.Conn
	tcpMutex    sync.RWMutex
}

// NewServer creates a new tunnel server
func NewServer(tlsConfig *tls.Config, addr string, maxConns int) (*Server, error) {
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// HTTP client for making requests to target websites
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		listener:   listener,
		httpClient: httpClient,
		logger:     logging.NewLogger("tunnel-server"),
		ctx:        ctx,
		cancel:     cancel,
		maxConns:   maxConns,
		tcpConns:   make(map[string]net.Conn),
	}, nil
}

// Start begins accepting connections
func (s *Server) Start() error {
	s.logger.Info("Tunnel server starting", "addr", s.listener.Addr())

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				s.logger.Error("Failed to accept connection", err)
				continue
			}
		}

		// Check connection limit
		s.connMutex.RLock()
		if int(s.activeConns) >= s.maxConns {
			s.connMutex.RUnlock()
			s.logger.Warn("Maximum connections reached, rejecting new connection")
			conn.Close()
			continue
		}
		s.connMutex.RUnlock()

		// Handle each connection in a goroutine
		s.wg.Add(1)
		go s.handleConnection(conn.(*tls.Conn))
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to close
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		s.logger.Warn("Timeout waiting for connections to close")
	}

	return nil
}

// handleConnection processes requests from a single agent
func (s *Server) handleConnection(conn *tls.Conn) {
	defer func() {
		conn.Close()
		s.wg.Done()

		s.connMutex.Lock()
		s.activeConns--
		s.connMutex.Unlock()
	}()

	s.connMutex.Lock()
	s.activeConns++
	s.connMutex.Unlock()

	// Complete the TLS handshake before inspecting connection state
	if err := conn.Handshake(); err != nil {
		s.logger.Error("TLS handshake failed", err)
		return
	}

	// Verify client certificate (after handshake)
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		s.logger.Warn("Client connected without certificate")
		return
	}

	clientCert := state.PeerCertificates[0]
	clientInfo := tlsutil.GetCertificateInfo(clientCert)
	s.logger.Info("Client connected", "client", clientCert.Subject.CommonName, "cert_info", clientInfo)

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		var env protocol.Envelope
		if err := decoder.Decode(&env); err != nil {
			if err != io.EOF {
				s.logger.Error("Failed to decode envelope", err)
			}
			break
		}

		switch env.Type {
		case "http_request":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var req protocol.Request
			if err := json.Unmarshal(b, &req); err != nil {
				s.logger.Error("Failed to parse http_request", err)
				continue
			}
			// Process request in a goroutine to handle concurrent requests
			go s.processRequest(&req, encoder)

		case "connect_open":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var open protocol.ConnectOpen
			if err := json.Unmarshal(b, &open); err != nil {
				s.logger.Error("Failed to parse connect_open", err)
				continue
			}
			go s.handleConnectOpen(&open, encoder)

		case "connect_data":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var data protocol.ConnectData
			if err := json.Unmarshal(b, &data); err != nil {
				s.logger.Error("Failed to parse connect_data", err)
				continue
			}
			go s.handleConnectData(&data)

		case "connect_close":
			m, _ := env.Payload.(map[string]any)
			b, _ := json.Marshal(m)
			var cls protocol.ConnectClose
			if err := json.Unmarshal(b, &cls); err != nil {
				continue
			}
			go s.handleConnectClose(&cls)

		default:
			// Ignore unknown message types
		}
	}

	s.logger.Info("Client disconnected", "client", clientCert.Subject.CommonName)
}

// processRequest handles a single HTTP request
func (s *Server) processRequest(req *protocol.Request, encoder *json.Encoder) {
	s.logger.Debug("Processing request", "id", req.ID, "method", req.Method, "url", req.URL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(s.ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		s.sendErrorResponse(req.ID, err, encoder)
		return
	}

	// Set headers
	for name, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(name, value)
		}
	}

	// Log the target endpoint (domain only)
	s.logRequest(req)

	// Make request
	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.sendErrorResponse(req.ID, err, encoder)
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		s.sendErrorResponse(req.ID, err, encoder)
		return
	}

	// Send response back through tunnel wrapped in Envelope
	resp := &protocol.Response{
		ID:         req.ID,
		StatusCode: httpResp.StatusCode,
		Headers:    convertHeaders(httpResp.Header),
		Body:       body,
	}

	env := protocol.Envelope{Type: "http_response", Payload: resp}
	if err := encoder.Encode(env); err != nil {
		s.logger.Error("Failed to send response", err, "id", req.ID)
	}

	s.logger.Debug("Response sent", "id", req.ID, "status", httpResp.StatusCode, "size", len(body))
}

// sendErrorResponse sends an error response back to the client
func (s *Server) sendErrorResponse(reqID string, err error, encoder *json.Encoder) {
	s.logger.Error("Request processing failed", err, "id", reqID)

	resp := &protocol.Response{
		ID:         reqID,
		StatusCode: 502,
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       []byte(fmt.Sprintf("Tunnel error: %v", err)),
		Error:      err.Error(),
	}

	env := protocol.Envelope{Type: "http_response", Payload: resp}
	if encodeErr := encoder.Encode(env); encodeErr != nil {
		s.logger.Error("Failed to send error response", encodeErr, "id", reqID)
	}
}

// convertHeaders converts http.Header to protocol headers format
func convertHeaders(headers http.Header) map[string][]string {
	result := make(map[string][]string)
	for name, values := range headers {
		result[name] = values
	}
	return result
}

// logRequest logs request information (domain only for privacy)
func (s *Server) logRequest(req *protocol.Request) {
	if parsedURL, err := parseURL(req.URL); err == nil {
		domain := parsedURL.Host

		// Remove port from domain for cleaner logging
		if host, _, err := net.SplitHostPort(domain); err == nil {
			domain = host
		}

		s.logger.Info("Forwarding request", "method", req.Method, "domain", domain, "id", req.ID)
	} else {
		s.logger.Warn("Invalid URL in request", "url", req.URL, "id", req.ID)
	}
}

// parseURL is a helper function to parse URLs safely
func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

// handleConnectOpen opens a TCP connection to the target address
func (s *Server) handleConnectOpen(open *protocol.ConnectOpen, encoder *json.Encoder) {
	s.logger.Info("CONNECT open request", "id", open.ID, "address", open.Address)

	// Dial target
	targetConn, err := net.DialTimeout("tcp", open.Address, 10*time.Second)
	if err != nil {
		s.logger.Error("CONNECT dial failed", err, "id", open.ID, "address", open.Address)
		// Send error via connect_close
		env := protocol.Envelope{Type: "connect_close", Payload: &protocol.ConnectClose{ID: open.ID, Error: err.Error()}}
		_ = encoder.Encode(env)
		return
	}

	// Store connection
	s.tcpMutex.Lock()
	s.tcpConns[open.ID] = targetConn
	s.tcpMutex.Unlock()

	// Send ack
	env := protocol.Envelope{Type: "connect_ack", Payload: &protocol.ConnectAck{ID: open.ID, Ok: true}}
	_ = encoder.Encode(env)

	// Start reader goroutine: read from target and send to agent
	go func() {
		defer func() {
			s.tcpMutex.Lock()
			delete(s.tcpConns, open.ID)
			s.tcpMutex.Unlock()
			targetConn.Close()
			// Send close
			closeEnv := protocol.Envelope{Type: "connect_close", Payload: &protocol.ConnectClose{ID: open.ID}}
			_ = encoder.Encode(closeEnv)
		}()

		buf := make([]byte, 32*1024)
		for {
			n, err := targetConn.Read(buf)
			if n > 0 {
				dataEnv := protocol.Envelope{Type: "connect_data", Payload: &protocol.ConnectData{ID: open.ID, Chunk: buf[:n]}}
				if encErr := encoder.Encode(dataEnv); encErr != nil {
					s.logger.Error("Failed to send connect_data", encErr, "id", open.ID)
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
}

// handleConnectData writes data to the TCP connection
func (s *Server) handleConnectData(data *protocol.ConnectData) {
	s.tcpMutex.RLock()
	targetConn := s.tcpConns[data.ID]
	s.tcpMutex.RUnlock()

	if targetConn == nil {
		return
	}

	if _, err := targetConn.Write(data.Chunk); err != nil {
		s.logger.Error("Failed to write to target conn", err, "id", data.ID)
		s.handleConnectClose(&protocol.ConnectClose{ID: data.ID})
	}
}

// handleConnectClose closes the TCP connection
func (s *Server) handleConnectClose(cls *protocol.ConnectClose) {
	s.tcpMutex.Lock()
	targetConn := s.tcpConns[cls.ID]
	delete(s.tcpConns, cls.ID)
	s.tcpMutex.Unlock()

	if targetConn != nil {
		targetConn.Close()
	}
}
