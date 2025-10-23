package protocol

import "time"

// Request represents an HTTP request through the tunnel
type Request struct {
	ID      string              `json:"id"`
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body,omitempty"`
}

// Response represents an HTTP response through the tunnel
type Response struct {
	ID         string              `json:"id"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// ConnectionInfo represents tunnel connection metadata
type ConnectionInfo struct {
	ClientID    string    `json:"client_id"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
}

// HealthCheck represents a health check message
type HealthCheck struct {
	Type      string    `json:"type"` // "ping" or "pong"
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// Envelope wraps different message kinds for the tunnel
// Types: "http_request", "http_response", "connect_open", "connect_ack", "connect_data", "connect_close"
type Envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// ConnectOpen requests the server to open a TCP connection to Address (host:port)
type ConnectOpen struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// ConnectAck acknowledges a ConnectOpen
type ConnectAck struct {
	ID    string `json:"id"`
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ConnectData carries a chunk of bytes for a TCP tunnel
type ConnectData struct {
	ID    string `json:"id"`
	Chunk []byte `json:"chunk"`
}

// ConnectClose signals closing a TCP tunnel
type ConnectClose struct {
	ID    string `json:"id"`
	Error string `json:"error,omitempty"`
}
