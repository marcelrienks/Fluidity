package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"fluidity/internal/shared/protocol"
)

func TestTunnelConnection(t *testing.T) {
	t.Parallel()

	// Generate test certificates
	certs := GenerateTestCerts(t)

	// Start test server
	server := StartTestServer(t, certs)
	defer server.Stop()

	// Start test client
	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Verify connection is established
	if !client.Client.IsConnected() {
		t.Fatal("Client should be connected")
	}

	t.Log("Tunnel connection established successfully")
}

func TestTunnelReconnection(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start server
	server := StartTestServer(t, certs)
	defer server.Stop()

	// Start client
	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Verify initial connection
	if !client.Client.IsConnected() {
		t.Fatal("Client should be connected initially")
	}

	// Disconnect client
	err := client.Client.Disconnect()
	AssertNoError(t, err, "Disconnect should not fail")

	// Verify disconnection
	if client.Client.IsConnected() {
		t.Fatal("Client should be disconnected")
	}

	// Reconnect
	err = client.Client.Connect()
	AssertNoError(t, err, "Reconnect should not fail")

	// Verify reconnection
	if !client.Client.IsConnected() {
		t.Fatal("Client should be reconnected")
	}

	t.Log("Tunnel reconnection successful")
}

func TestHTTPRequestThroughTunnel(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start mock HTTP server
	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from mock server"))
	})

	// Start tunnel server
	server := StartTestServer(t, certs)
	defer server.Stop()

	// Start tunnel client
	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Create request
	req := &protocol.Request{
		ID:      "test-req-1",
		Method:  "GET",
		URL:     mockServer.URL,
		Headers: map[string][]string{"User-Agent": {"test"}},
		Body:    []byte{},
	}

	// Send request through tunnel
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a channel to get the response
	respChan := make(chan *protocol.Response, 1)
	errChan := make(chan error, 1)

	go func() {
		resp, err := client.Client.SendRequest(req)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- resp
	}()

	// Wait for response
	select {
	case resp := <-respChan:
		AssertEqual(t, 200, resp.StatusCode, "HTTP status code")
		if !bytes.Contains(resp.Body, []byte("Hello from mock server")) {
			t.Fatalf("Unexpected response body: %s", string(resp.Body))
		}
		t.Log("HTTP request through tunnel successful")

	case err := <-errChan:
		t.Fatalf("Request failed: %v", err)

	case <-ctx.Done():
		t.Fatal("Request timeout")
	}
}

func TestHTTPRequestTimeout(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start mock server that delays response
	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than tunnel timeout
		w.WriteHeader(http.StatusOK)
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Create request
	req := &protocol.Request{
		ID:      "test-timeout",
		Method:  "GET",
		URL:     mockServer.URL,
		Headers: map[string][]string{},
		Body:    []byte{},
	}

	// Send request
	_, err := client.Client.SendRequest(req)

	// Should timeout
	AssertError(t, err, "Request should timeout")
	t.Logf("Request correctly timed out: %v", err)
}

func TestMultipleConcurrentRequests(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start mock server
	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Response")
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Send multiple concurrent requests
	numRequests := 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := &protocol.Request{
				ID:      protocol.GenerateID(),
				Method:  "GET",
				URL:     mockServer.URL,
				Headers: map[string][]string{},
				Body:    []byte{},
			}

			resp, err := client.Client.SendRequest(req)
			if err != nil {
				results <- err
				return
			}

			if resp.StatusCode != 200 {
				results <- err
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	t.Logf("All %d concurrent requests completed successfully", numRequests)
}

func TestTunnelWithDifferentHTTPMethods(t *testing.T) {
	t.Parallel()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			certs := GenerateTestCerts(t)

			// Start mock server
			receivedMethod := ""
			mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			})

			// Start tunnel
			server := StartTestServer(t, certs)
			defer server.Stop()

			client := StartTestClient(t, server.Addr, certs)
			defer client.Stop()

			// Create request
			req := &protocol.Request{
				ID:      protocol.GenerateID(),
				Method:  method,
				URL:     mockServer.URL,
				Headers: map[string][]string{},
				Body:    []byte("test body"),
			}

			// Send request
			resp, err := client.Client.SendRequest(req)
			AssertNoError(t, err, "Request should not fail")
			AssertEqual(t, 200, resp.StatusCode, "HTTP status code")
			AssertEqual(t, method, receivedMethod, "HTTP method")
		})
	}
}

func TestTunnelWithLargePayload(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Create large payload (1MB)
	largeBody := make([]byte, 1024*1024)
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	// Start mock server
	var receivedBodySize int
	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBodySize = len(body)
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody) // Send large response back
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Create request with large body
	req := &protocol.Request{
		ID:      protocol.GenerateID(),
		Method:  "POST",
		URL:     mockServer.URL,
		Headers: map[string][]string{"Content-Type": {"application/octet-stream"}},
		Body:    largeBody,
	}

	// Send request
	resp, err := client.Client.SendRequest(req)
	AssertNoError(t, err, "Large payload request should not fail")
	AssertEqual(t, 200, resp.StatusCode, "HTTP status code")
	AssertEqual(t, len(largeBody), receivedBodySize, "Request body size")
	AssertEqual(t, len(largeBody), len(resp.Body), "Response body size")

	t.Logf("Successfully transferred %d bytes in both directions", len(largeBody))
}

func TestTunnelDisconnectDuringRequest(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start mock server with delay
	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Start request
	errChan := make(chan error, 1)
	go func() {
		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     mockServer.URL,
			Headers: map[string][]string{},
			Body:    []byte{},
		}
		_, err := client.Client.SendRequest(req)
		errChan <- err
	}()

	// Disconnect during request
	time.Sleep(500 * time.Millisecond)
	client.Client.Disconnect()

	// Should get error
	select {
	case err := <-errChan:
		AssertError(t, err, "Should get error when disconnecting during request")
		t.Logf("Got expected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for error")
	}
}
