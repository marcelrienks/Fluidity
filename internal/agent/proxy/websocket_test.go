package proxy

import (
	"testing"
)

// TestWebSocketUpgradeDetection tests if WebSocket upgrade requests are properly detected
func TestWebSocketUpgradeDetection(t *testing.T) {
	tests := []struct {
		name            string
		upgradeHeader   string
		connectionHeader string
		expected        bool
	}{
		{
			name:            "Valid WebSocket upgrade",
			upgradeHeader:   "websocket",
			connectionHeader: "Upgrade",
			expected:        true,
		},
		{
			name:            "Valid WebSocket upgrade (case insensitive)",
			upgradeHeader:   "WebSocket",
			connectionHeader: "upgrade",
			expected:        true,
		},
		{
			name:            "Valid WebSocket upgrade with multiple connection values",
			upgradeHeader:   "websocket",
			connectionHeader: "keep-alive, Upgrade",
			expected:        true,
		},
		{
			name:            "Invalid upgrade header",
			upgradeHeader:   "h2c",
			connectionHeader: "Upgrade",
			expected:        false,
		},
		{
			name:            "Missing upgrade header",
			upgradeHeader:   "",
			connectionHeader: "Upgrade",
			expected:        false,
		},
		{
			name:            "Missing connection header",
			upgradeHeader:   "websocket",
			connectionHeader: "",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock request
			req := &mockRequest{
				headers: map[string]string{
					"Upgrade":    tt.upgradeHeader,
					"Connection": tt.connectionHeader,
				},
			}

			// Create server instance
			server := &Server{}

			// Test the detection
			result := server.isWebSocketUpgrade(req.toHTTPRequest())
			if result != tt.expected {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// mockRequest is a helper struct for testing
type mockRequest struct {
	headers map[string]string
}

func (m *mockRequest) toHTTPRequest() *http.Request {
	import "net/http"
	
	req, _ := http.NewRequest("GET", "http://example.com/ws", nil)
	for k, v := range m.headers {
		req.Header.Set(k, v)
	}
	return req
}
