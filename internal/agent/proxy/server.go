package proxy

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"fluidity/internal/agent/tunnel"
	"fluidity/internal/shared/logging"
	"fluidity/internal/shared/protocol"
)

// Server handles local HTTP proxy requests
type Server struct {
	server      *http.Server
	tunnelConn  *tunnel.Client
	logger      *logging.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewServer creates a new proxy server
func NewServer(port int, tunnelConn *tunnel.Client) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	
	proxy := &Server{
		tunnelConn: tunnelConn,
		logger:     logging.NewLogger("proxy-server"),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.handleRequest)
	
	proxy.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return proxy
}

// Start begins serving HTTP proxy requests
func (p *Server) Start() error {
	p.logger.Info("Starting HTTP proxy server", "addr", p.server.Addr)
	
	listener, err := net.Listen("tcp", p.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}
	
	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			p.logger.Error("Proxy server error", err)
		}
	}()
	
	p.logger.Info("HTTP proxy server started", "addr", p.server.Addr)
	return nil
}

// Stop gracefully shuts down the proxy server
func (p *Server) Stop() error {
	p.logger.Info("Stopping HTTP proxy server")
	p.cancel()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return p.server.Shutdown(ctx)
}

// handleRequest processes incoming HTTP requests
func (p *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Log the request (domain only for privacy)
	p.logRequest(r)
	
	// Handle CONNECT method for HTTPS tunneling
	if r.Method == "CONNECT" {
		p.handleConnect(w, r)
		return
	}
	
	// Handle regular HTTP requests
	p.handleHTTPRequest(w, r)
}

// handleHTTPRequest processes regular HTTP requests
func (p *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// Generate request ID
	reqID := p.generateRequestID()
	
	// Ensure URL is absolute
	if !r.URL.IsAbs() {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		r.URL.Scheme = scheme
		r.URL.Host = r.Host
	}
	
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.logger.Error("Failed to read request body", err, "id", reqID)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	
	// Convert HTTP request to tunnel protocol
	tunnelReq := &protocol.Request{
		ID:      reqID,
		Method:  r.Method,
		URL:     r.URL.String(),
		Headers: convertHeaders(r.Header),
		Body:    body,
	}
	
	// Send through tunnel and get response
	resp, err := p.tunnelConn.SendRequest(tunnelReq)
	if err != nil {
		p.logger.Error("Failed to send request through tunnel", err, "id", reqID)
		http.Error(w, "Tunnel error", http.StatusBadGateway)
		return
	}
	
	// Write response back to client
	p.writeResponse(w, resp)
}

// handleConnect handles HTTPS CONNECT requests for tunneling
func (p *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	// For now, return 501 Not Implemented for CONNECT
	// This will be enhanced in later phases to support HTTPS tunneling
	p.logger.Warn("CONNECT method not yet implemented", "host", r.Host)
	http.Error(w, "CONNECT method not implemented", http.StatusNotImplemented)
}

// writeResponse writes the tunnel response back to the HTTP client
func (p *Server) writeResponse(w http.ResponseWriter, resp *protocol.Response) {
	// Set headers
	for name, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	
	// Set status code
	w.WriteHeader(resp.StatusCode)
	
	// Write body
	if len(resp.Body) > 0 {
		w.Write(resp.Body)
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

// generateRequestID generates a unique request ID
func (p *Server) generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// logRequest logs request information (domain only for privacy)
func (p *Server) logRequest(r *http.Request) {
	var domain string
	if r.URL != nil && r.URL.Host != "" {
		domain = r.URL.Host
	} else {
		domain = r.Host
	}
	
	// Remove port from domain for cleaner logging
	if host, _, err := net.SplitHostPort(domain); err == nil {
		domain = host
	}
	
	p.logger.Info("Proxying request", "method", r.Method, "domain", domain)
}