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

		var req protocol.Request
		if err := decoder.Decode(&req); err != nil {
			if err != io.EOF {
				s.logger.Error("Failed to decode request", err)
			}
			break
		}

		// Process request in a goroutine to handle concurrent requests
		go s.processRequest(&req, encoder)
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

	// Send response back through tunnel
	resp := &protocol.Response{
		ID:         req.ID,
		StatusCode: httpResp.StatusCode,
		Headers:    convertHeaders(httpResp.Header),
		Body:       body,
	}

	if err := encoder.Encode(resp); err != nil {
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

	if encodeErr := encoder.Encode(resp); encodeErr != nil {
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
