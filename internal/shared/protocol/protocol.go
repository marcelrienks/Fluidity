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