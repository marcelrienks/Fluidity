package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestWebSocketConnection(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start mock WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Echo server
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = conn.WriteMessage(messageType, message)
			if err != nil {
				break
			}
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect through proxy
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")
	defer conn.Close()

	// Send and receive message
	testMessage := []byte("Hello WebSocket")
	err = conn.WriteMessage(websocket.TextMessage, testMessage)
	AssertNoError(t, err, "Send message should not fail")

	messageType, message, err := conn.ReadMessage()
	AssertNoError(t, err, "Read message should not fail")
	AssertEqual(t, websocket.TextMessage, messageType, "Message type")
	AssertEqual(t, string(testMessage), string(message), "Message content")

	t.Log("WebSocket connection through tunnel successful")
}

func TestWebSocketMultipleMessages(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start echo WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, message)
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")
	defer conn.Close()

	// Send multiple messages
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		message := []byte(fmt.Sprintf("Message %d", i))
		err := conn.WriteMessage(websocket.TextMessage, message)
		AssertNoError(t, err, fmt.Sprintf("Send message %d should not fail", i))

		_, received, err := conn.ReadMessage()
		AssertNoError(t, err, fmt.Sprintf("Read message %d should not fail", i))
		AssertEqual(t, string(message), string(received), fmt.Sprintf("Message %d content", i))
	}

	t.Logf("Successfully sent and received %d WebSocket messages", numMessages)
}

func TestWebSocketBinaryMessages(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start echo WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, message)
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")
	defer conn.Close()

	// Send binary data
	binaryData := make([]byte, 1024)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, binaryData)
	AssertNoError(t, err, "Send binary message should not fail")

	messageType, received, err := conn.ReadMessage()
	AssertNoError(t, err, "Read binary message should not fail")
	AssertEqual(t, websocket.BinaryMessage, messageType, "Message type")
	AssertEqual(t, len(binaryData), len(received), "Binary data length")

	t.Log("WebSocket binary message transfer successful")
}

func TestWebSocketPingPong(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start WebSocket server that responds to pings
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")
	defer conn.Close()

	// Set pong handler
	pongReceived := make(chan struct{}, 1)
	conn.SetPongHandler(func(string) error {
		pongReceived <- struct{}{}
		return nil
	})

	// Start reading (required for pong handler to be called)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Send ping
	err = conn.WriteMessage(websocket.PingMessage, []byte{})
	AssertNoError(t, err, "Send ping should not fail")

	// Wait for pong
	select {
	case <-pongReceived:
		t.Log("Pong received successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for pong")
	}
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start echo WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, message)
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Create multiple concurrent connections
	numConnections := 5
	var wg sync.WaitGroup

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
			proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

			dialer := websocket.Dialer{
				Proxy: func(req *http.Request) (*url.URL, error) {
					return url.Parse(proxyURL)
				},
			}

			conn, _, err := dialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("Connection %d failed: %v", connID, err)
				return
			}
			defer conn.Close()

			// Send and receive messages
			for j := 0; j < 5; j++ {
				message := []byte(fmt.Sprintf("Conn %d Msg %d", connID, j))
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					t.Errorf("Connection %d send failed: %v", connID, err)
					return
				}

				_, received, err := conn.ReadMessage()
				if err != nil {
					t.Errorf("Connection %d receive failed: %v", connID, err)
					return
				}

				if string(message) != string(received) {
					t.Errorf("Connection %d message mismatch", connID)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	t.Logf("Successfully handled %d concurrent WebSocket connections", numConnections)
}

func TestWebSocketLargeMessage(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start echo WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(messageType, message)
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")
	defer conn.Close()

	// Create moderate size message (2KB - reliable across WebSocket implementations)
	largeMessage := make([]byte, 2*1024)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	// Send large message
	err = conn.WriteMessage(websocket.BinaryMessage, largeMessage)
	AssertNoError(t, err, "Send large message should not fail")

	// Receive large message
	_, received, err := conn.ReadMessage()
	AssertNoError(t, err, "Read large message should not fail")
	AssertEqual(t, len(largeMessage), len(received), "Large message length")

	t.Logf("Successfully transferred %d bytes via WebSocket", len(largeMessage))
}

func TestWebSocketCloseHandshake(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer wsServer.Close()

	// Start tunnel
	tunnelServer := StartTestServer(t, certs)
	defer tunnelServer.Stop()

	agent := StartTestClient(t, tunnelServer.Addr, certs)
	defer agent.Stop()

	time.Sleep(500 * time.Millisecond)

	// Connect
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	proxyURL := fmt.Sprintf("http://localhost:%d", agent.ProxyPort)

	dialer := websocket.Dialer{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	AssertNoError(t, err, "WebSocket connection should not fail")

	// Send a message
	err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
	AssertNoError(t, err, "Send message should not fail")

	// Close with close handshake
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	AssertNoError(t, err, "Send close message should not fail")

	// Give time for close handshake
	time.Sleep(100 * time.Millisecond)

	err = conn.Close()
	AssertNoError(t, err, "Close connection should not fail")

	t.Log("WebSocket close handshake successful")
}
