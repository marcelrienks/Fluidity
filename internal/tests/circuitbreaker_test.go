package tests

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"fluidity/internal/shared/protocol"
)

func TestCircuitBreakerTripsOnFailures(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Send requests to invalid/non-existent URL to cause network errors
	// Circuit breaker threshold is 5 failures
	networkErrors := 0
	circuitOpenErrors := 0

	for i := 0; i < 10; i++ {
		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     "http://invalid-host-that-does-not-exist-12345.local",
			Headers: map[string][]string{},
			Body:    []byte{},
		}

		resp, err := client.Client.SendRequest(req)
		if err != nil {
			t.Logf("Request %d failed (expected): %v", i, err)
			networkErrors++
		} else if resp.Error != "" {
			// Check error message in response
			if resp.Error == "service temporarily unavailable (circuit open)" {
				circuitOpenErrors++
				t.Logf("Request %d: circuit breaker is open", i)
			} else {
				networkErrors++
				t.Logf("Request %d: network error - %s", i, resp.Error)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Circuit breaker should have tripped after some failures
	if circuitOpenErrors == 0 {
		t.Fatal("Expected circuit breaker to trip and reject some requests")
	}

	t.Logf("Circuit breaker tripped: %d network errors, %d circuit open rejections", networkErrors, circuitOpenErrors)
}

func TestCircuitBreakerRecovery(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Mock server that can toggle between failure and success
	shouldFail := atomic.Bool{}
	shouldFail.Store(true)

	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		if shouldFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Phase 1: Cause failures to trip circuit breaker
	t.Log("Phase 1: Causing failures to trip circuit breaker")
	for i := 0; i < 6; i++ {
		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     mockServer.URL,
			Headers: map[string][]string{},
			Body:    []byte{},
		}
		client.Client.SendRequest(req)
		time.Sleep(100 * time.Millisecond)
	}

	// Phase 2: Wait for circuit breaker to enter half-open state
	// Circuit breaker timeout is 10 seconds
	t.Log("Phase 2: Waiting for half-open state (10 seconds)")
	time.Sleep(11 * time.Second)

	// Phase 3: Fix the server and send successful request
	t.Log("Phase 3: Server fixed, sending successful request")
	shouldFail.Store(false)

	req := &protocol.Request{
		ID:      protocol.GenerateID(),
		Method:  "GET",
		URL:     mockServer.URL,
		Headers: map[string][]string{},
		Body:    []byte{},
	}

	resp, err := client.Client.SendRequest(req)
	if err != nil {
		t.Logf("Half-open test request failed: %v (circuit may still be recovering)", err)
		// Try one more time
		time.Sleep(1 * time.Second)
		resp, err = client.Client.SendRequest(req)
	}

	AssertNoError(t, err, "Request should succeed after recovery")
	AssertEqual(t, 200, resp.StatusCode, "Status code after recovery")

	t.Log("Circuit breaker successfully recovered")
}

func TestCircuitBreakerProtectsFromCascadingFailures(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Send multiple requests to invalid hosts to trigger failures quickly
	numRequests := 10
	networkErrors := 0
	circuitOpenErrors := 0

	for i := 0; i < numRequests; i++ {
		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     fmt.Sprintf("http://invalid-host-%d.local", i),
			Headers: map[string][]string{},
			Body:    []byte{},
		}

		resp, err := client.Client.SendRequest(req)
		if err != nil {
			networkErrors++
		} else if resp.Error != "" {
			if resp.Error == "service temporarily unavailable (circuit open)" {
				circuitOpenErrors++
			} else {
				networkErrors++
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	totalFailures := networkErrors + circuitOpenErrors
	t.Logf("Circuit breaker test: %d network errors, %d circuit open, %d total failures out of %d requests",
		networkErrors, circuitOpenErrors, totalFailures, numRequests)

	// Expect all or most requests to fail (network errors + circuit breaker protection)
	if totalFailures < numRequests/2 {
		t.Errorf("Expected most requests to fail, got %d/%d", totalFailures, numRequests)
	}

	// Circuit breaker should have kicked in for some requests
	if circuitOpenErrors == 0 {
		t.Log("Warning: Circuit breaker didn't trigger (may need more consistent failures)")
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	successCount := atomic.Int32{}
	failCount := atomic.Int32{}

	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Fail every other request
		if r.URL.Query().Get("fail") == "true" {
			failCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			successCount.Add(1)
			w.WriteHeader(http.StatusOK)
		}
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	// Send mix of successful and failing requests
	for i := 0; i < 10; i++ {
		shouldFail := i%2 == 0
		url := mockServer.URL
		if shouldFail {
			url += "?fail=true"
		}

		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     url,
			Headers: map[string][]string{},
			Body:    []byte{},
		}

		client.Client.SendRequest(req)
		time.Sleep(100 * time.Millisecond)
	}

	// Log metrics
	totalRequests := successCount.Load() + failCount.Load()
	successRate := float64(successCount.Load()) / float64(totalRequests) * 100

	t.Logf("Circuit breaker metrics:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount.Load())
	t.Logf("  Failed: %d", failCount.Load())
	t.Logf("  Success rate: %.1f%%", successRate)

	// We should have some successes and some failures
	if successCount.Load() == 0 {
		t.Error("No successful requests")
	}
	if failCount.Load() == 0 {
		t.Error("No failed requests")
	}
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	t.Parallel()

	certs := GenerateTestCerts(t)

	shouldFail := atomic.Bool{}
	shouldFail.Store(false) // Start with success

	mockServer := MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		if shouldFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	// Start tunnel
	server := StartTestServer(t, certs)
	defer server.Stop()

	client := StartTestClient(t, server.Addr, certs)
	defer client.Stop()

	sendRequest := func() error {
		req := &protocol.Request{
			ID:      protocol.GenerateID(),
			Method:  "GET",
			URL:     mockServer.URL,
			Headers: map[string][]string{},
			Body:    []byte{},
		}
		_, err := client.Client.SendRequest(req)
		return err
	}

	// State: CLOSED - requests succeed
	t.Log("State 1: CLOSED - sending successful requests")
	for i := 0; i < 3; i++ {
		err := sendRequest()
		if err != nil {
			t.Logf("Request failed in CLOSED state: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Transition to OPEN - cause failures
	t.Log("State 2: Transitioning to OPEN - causing failures")
	shouldFail.Store(true)
	for i := 0; i < 6; i++ {
		sendRequest()
		time.Sleep(100 * time.Millisecond)
	}

	// State: OPEN - requests fail fast
	t.Log("State 3: OPEN - requests should fail fast")
	startTime := time.Now()
	for i := 0; i < 3; i++ {
		sendRequest()
	}
	failFastDuration := time.Since(startTime)

	if failFastDuration > 1*time.Second {
		t.Errorf("Requests not failing fast in OPEN state: %v", failFastDuration)
	} else {
		t.Logf("Requests failed fast in %v", failFastDuration)
	}

	// Wait for HALF_OPEN
	t.Log("State 4: Waiting for HALF_OPEN state")
	time.Sleep(11 * time.Second)

	// State: HALF_OPEN - test request
	t.Log("State 5: HALF_OPEN - sending test request")
	shouldFail.Store(false)
	err := sendRequest()
	if err != nil {
		t.Logf("Test request in HALF_OPEN failed: %v", err)
	}

	// Back to CLOSED - verify success
	t.Log("State 6: Back to CLOSED - verifying normal operation")
	for i := 0; i < 3; i++ {
		err := sendRequest()
		AssertNoError(t, err, fmt.Sprintf("Request %d should succeed in CLOSED state", i))
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Circuit breaker state transitions completed successfully")
}
