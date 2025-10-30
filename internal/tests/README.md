# Integration Tests

This directory contains integration tests for the Fluidity project. These tests verify the interaction between multiple components without requiring full deployment.

## Test Organization

```
internal/integration/
├── README.md              # This file
├── testutil.go            # Shared test utilities and helpers
├── tunnel_test.go         # Agent <-> Server tunnel integration
├── http_proxy_test.go     # HTTP proxy functionality
├── connect_test.go        # HTTPS CONNECT tunneling
├── websocket_test.go      # WebSocket tunneling
└── circuitbreaker_integration_test.go  # Circuit breaker integration
```

## Running Integration Tests

### Run all integration tests:
```bash
go test ./internal/integration/... -v
```

### Run specific test:
```bash
go test ./internal/integration/... -v -run TestTunnelConnection
```

### Run with race detector:
```bash
go test ./internal/integration/... -v -race
```

### Run with coverage:
```bash
go test ./internal/integration/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Approach

Integration tests in this directory:

1. **Start in-memory servers**: No external processes needed
2. **Use real TLS certificates**: Tests with actual mTLS
3. **Mock external HTTP calls**: Fast and reliable
4. **Test component interactions**: Focus on integration points
5. **Clean up resources**: Proper teardown in defer statements

## Key Differences from E2E Tests

| Aspect | Integration Tests | E2E Tests (scripts) |
|--------|------------------|---------------------|
| **Scope** | Component interactions | Full system deployment |
| **Speed** | Fast (< 1s per test) | Slow (10-30s) |
| **Dependencies** | In-memory only | Requires binaries/Docker |
| **Network** | Mocked/local only | Real external services |
| **Purpose** | Development feedback | Deployment validation |
| **When to run** | Every commit | Before deploy/PR merge |

## Test Patterns

### Setting up test server:
```go
func TestExample(t *testing.T) {
    // Start test server
    server, cleanup := startTestServer(t)
    defer cleanup()
    
    // Your test code here
}
```

### Testing with timeout:
```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Your test code with ctx
}
```

## Adding New Integration Tests

1. Create a new `*_test.go` file in this directory
2. Use `testutil.go` helpers for common setup
3. Follow the existing test patterns
4. Add cleanup in `defer` statements
5. Use table-driven tests for multiple scenarios

## Best Practices

- ✅ Use `t.Parallel()` for independent tests
- ✅ Clean up resources in defer statements
- ✅ Use meaningful test names (TestComponent_Scenario_ExpectedOutcome)
- ✅ Test both success and failure cases
- ✅ Mock external dependencies
- ✅ Validate error messages and types
- ❌ Don't test implementation details
- ❌ Don't depend on test execution order
- ❌ Don't use real external services

## Debugging Failed Tests

### Verbose output:
```bash
go test ./internal/integration/... -v
```

### Run specific test with logging:
```bash
go test ./internal/integration/... -v -run TestName 2>&1 | tee test.log
```

### Debug with delve:
```bash
dlv test ./internal/integration/... -- -test.run TestName
```
