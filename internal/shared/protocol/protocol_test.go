package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	// Generate IDs and ensure they are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()

		// Check length (16 bytes = 32 hex characters)
		if len(id) != 32 {
			t.Errorf("Expected ID length 32, got %d", len(id))
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestRequestSerialization(t *testing.T) {
	req := &Request{
		ID:     "test-id",
		Method: "POST",
		URL:    "http://example.com/api",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: []byte(`{"key":"value"}`),
	}

	// Serialize
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Deserialize
	var decoded Request
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify
	if decoded.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, decoded.ID)
	}
	if decoded.Method != req.Method {
		t.Errorf("Expected method %s, got %s", req.Method, decoded.Method)
	}
	if decoded.URL != req.URL {
		t.Errorf("Expected URL %s, got %s", req.URL, decoded.URL)
	}
	if string(decoded.Body) != string(req.Body) {
		t.Errorf("Expected body %s, got %s", req.Body, decoded.Body)
	}
}

func TestResponseSerialization(t *testing.T) {
	resp := &Response{
		ID:         "test-id",
		StatusCode: 200,
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
		},
		Body:  []byte("Success"),
		Error: "",
	}

	// Serialize
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Deserialize
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify
	if decoded.ID != resp.ID {
		t.Errorf("Expected ID %s, got %s", resp.ID, decoded.ID)
	}
	if decoded.StatusCode != resp.StatusCode {
		t.Errorf("Expected status code %d, got %d", resp.StatusCode, decoded.StatusCode)
	}
	if string(decoded.Body) != string(resp.Body) {
		t.Errorf("Expected body %s, got %s", resp.Body, decoded.Body)
	}
}

func TestResponseWithError(t *testing.T) {
	resp := &Response{
		ID:         "test-id",
		StatusCode: 500,
		Error:      "Internal server error",
	}

	// Serialize
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Deserialize
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify error field
	if decoded.Error != resp.Error {
		t.Errorf("Expected error %s, got %s", resp.Error, decoded.Error)
	}
}

func TestEnvelopeSerialization(t *testing.T) {
	req := &Request{
		ID:     GenerateID(),
		Method: "GET",
		URL:    "http://example.com",
	}

	envelope := &Envelope{
		Type:    "http_request",
		Payload: req,
	}

	// Serialize
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Failed to marshal envelope: %v", err)
	}

	// Deserialize (two-step process)
	var rawEnvelope struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	err = json.Unmarshal(data, &rawEnvelope)
	if err != nil {
		t.Fatalf("Failed to unmarshal envelope: %v", err)
	}

	if rawEnvelope.Type != "http_request" {
		t.Errorf("Expected type http_request, got %s", rawEnvelope.Type)
	}

	// Deserialize payload
	var decodedReq Request
	err = json.Unmarshal(rawEnvelope.Payload, &decodedReq)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if decodedReq.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, decodedReq.ID)
	}
}

func TestConnectMessages(t *testing.T) {
	// Test ConnectOpen
	open := &ConnectOpen{
		ID:      GenerateID(),
		Address: "example.com:443",
	}

	data, _ := json.Marshal(open)
	var decodedOpen ConnectOpen
	json.Unmarshal(data, &decodedOpen)

	if decodedOpen.Address != open.Address {
		t.Errorf("Expected address %s, got %s", open.Address, decodedOpen.Address)
	}

	// Test ConnectAck
	ack := &ConnectAck{
		ID:    open.ID,
		Ok:    true,
		Error: "",
	}

	data, _ = json.Marshal(ack)
	var decodedAck ConnectAck
	json.Unmarshal(data, &decodedAck)

	if !decodedAck.Ok {
		t.Error("Expected Ok to be true")
	}

	// Test ConnectData
	connectData := &ConnectData{
		ID:    open.ID,
		Chunk: []byte("test data"),
	}

	data, _ = json.Marshal(connectData)
	var decodedData ConnectData
	json.Unmarshal(data, &decodedData)

	if string(decodedData.Chunk) != "test data" {
		t.Errorf("Expected chunk 'test data', got %s", decodedData.Chunk)
	}

	// Test ConnectClose
	close := &ConnectClose{
		ID:    open.ID,
		Error: "",
	}

	data, _ = json.Marshal(close)
	var decodedClose ConnectClose
	json.Unmarshal(data, &decodedClose)

	if decodedClose.ID != open.ID {
		t.Errorf("Expected ID %s, got %s", open.ID, decodedClose.ID)
	}
}

func TestWebSocketMessages(t *testing.T) {
	// Test WebSocketOpen
	wsOpen := &WebSocketOpen{
		ID:  GenerateID(),
		URL: "ws://example.com/socket",
		Headers: map[string][]string{
			"Origin": {"http://example.com"},
		},
	}

	data, _ := json.Marshal(wsOpen)
	var decodedOpen WebSocketOpen
	json.Unmarshal(data, &decodedOpen)

	if decodedOpen.URL != wsOpen.URL {
		t.Errorf("Expected URL %s, got %s", wsOpen.URL, decodedOpen.URL)
	}

	// Test WebSocketAck
	wsAck := &WebSocketAck{
		ID:    wsOpen.ID,
		Ok:    true,
		Error: "",
	}

	data, _ = json.Marshal(wsAck)
	var decodedAck WebSocketAck
	json.Unmarshal(data, &decodedAck)

	if !decodedAck.Ok {
		t.Error("Expected Ok to be true")
	}

	// Test WebSocketMessage
	wsMsg := &WebSocketMessage{
		ID:          wsOpen.ID,
		MessageType: 1, // TextMessage
		Data:        []byte("Hello WebSocket"),
	}

	data, _ = json.Marshal(wsMsg)
	var decodedMsg WebSocketMessage
	json.Unmarshal(data, &decodedMsg)

	if decodedMsg.MessageType != 1 {
		t.Errorf("Expected message type 1, got %d", decodedMsg.MessageType)
	}
	if string(decodedMsg.Data) != "Hello WebSocket" {
		t.Errorf("Expected data 'Hello WebSocket', got %s", decodedMsg.Data)
	}

	// Test WebSocketClose
	wsClose := &WebSocketClose{
		ID:    wsOpen.ID,
		Code:  1000,
		Error: "",
	}

	data, _ = json.Marshal(wsClose)
	var decodedClose WebSocketClose
	json.Unmarshal(data, &decodedClose)

	if decodedClose.Code != 1000 {
		t.Errorf("Expected close code 1000, got %d", decodedClose.Code)
	}
}

func TestConnectionInfo(t *testing.T) {
	now := time.Now()
	info := &ConnectionInfo{
		ClientID:    "client-123",
		ConnectedAt: now,
		LastSeen:    now,
	}

	// Serialize
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal connection info: %v", err)
	}

	// Deserialize
	var decoded ConnectionInfo
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal connection info: %v", err)
	}

	if decoded.ClientID != info.ClientID {
		t.Errorf("Expected ClientID %s, got %s", info.ClientID, decoded.ClientID)
	}
}

func TestHealthCheck(t *testing.T) {
	ping := &HealthCheck{
		Type:      "ping",
		Timestamp: time.Now(),
		Message:   "are you there?",
	}

	// Serialize
	data, err := json.Marshal(ping)
	if err != nil {
		t.Fatalf("Failed to marshal health check: %v", err)
	}

	// Deserialize
	var decoded HealthCheck
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal health check: %v", err)
	}

	if decoded.Type != "ping" {
		t.Errorf("Expected type ping, got %s", decoded.Type)
	}
	if decoded.Message != "are you there?" {
		t.Errorf("Expected message 'are you there?', got %s", decoded.Message)
	}
}
