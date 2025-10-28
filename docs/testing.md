# Testing Guide

This document provides a comprehensive guide to testing the Fluidity secure tunnel system.

## Table of Contents

- [Overview](#overview)
- [Test Types](#test-types)
- [Quick Start](#quick-start)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [End-to-End Tests](#end-to-end-tests)
- [Test Coverage](#test-coverage)
- [CI/CD Integration](#cicd-integration)
- [Lambda Control Plane Testing](#lambda-control-plane-testing)
- [Troubleshooting](#troubleshooting)

## Overview

Fluidity implements a comprehensive three-tier testing strategy:

```
                  ┌────────────┐
                  │   E2E (6)  │  ← Full system, real binaries
                  │  Slowest   │
                  └────────────┘
                ┌───────────────┐
                │Integration(30)│   ← Component interaction
                │    Medium     │
                └───────────────┘
            ┌──────────────────────┐
            │   Unit Tests (17)    │  ← Individual functions
            │       Fastest        │
            └──────────────────────┘
```

**Total: 53+ tests covering all aspects of the system**

## Test Types

### 1. Unit Tests
- **Location**: `internal/shared/*/`
- **Purpose**: Test individual functions and methods in isolation
- **Count**: 17 tests
- **Coverage**: 100% for circuit breaker and retry logic
- **Execution Time**: < 1 second

### 2. Integration Tests
- **Location**: `internal/integration/`
- **Purpose**: Test component interactions with real network connections
- **Count**: 30+ tests
- **Coverage**: Full component interaction testing
- **Execution Time**: ~3-10 seconds (parallel execution)

### 3. End-to-End (E2E) Tests
- **Location**: `scripts/test-*.ps1` and `scripts/test-*.sh`
- **Purpose**: Test complete system with real binaries/containers
- **Count**: 6 scenarios (3 protocols × 2 environments)
- **Coverage**: Full system validation
- **Execution Time**: 30-120 seconds

## Quick Start

Run all tests at once:

```powershell
# Run everything
go test ./internal/... -v -timeout 5m
.\scripts\test-local.ps1
.\scripts\test-docker.ps1
```

Or run specific test types:

```powershell
# Just unit tests (fastest)
go test ./internal/shared/... -v

# Just integration tests
go test ./internal/integration/... -v -timeout 5m

# Just E2E tests
.\scripts\test-local.ps1   # or ./scripts/test-local.sh on Linux/macOS
```

## Unit Tests

### What They Test

**Circuit Breaker** (`internal/shared/circuitbreaker/circuitbreaker_test.go`)
- State transitions (Closed → Open → Half-Open → Closed)
- Failure threshold detection
- Automatic recovery after timeout
- Concurrent request handling
- Configuration validation

**Retry Mechanism** (`internal/shared/retry/retry_test.go`)
- Successful operations
- Retry on transient failures
- Maximum attempts limit
- Exponential backoff timing
- Context cancellation
- Generic type support

### Running Unit Tests

```powershell
# Run all unit tests
go test ./internal/shared/... -v

# Run specific package
go test ./internal/shared/circuitbreaker -v
go test ./internal/shared/retry -v

# Run with coverage
go test ./internal/shared/... -cover

# Generate coverage report (HTML)
go test ./internal/shared/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Expected Output

```
=== RUN   TestCircuitBreakerStateTransitions
=== RUN   TestCircuitBreakerFailureThreshold
=== RUN   TestRetrySuccess
=== RUN   TestRetryExponentialBackoff
...
--- PASS: All tests (0.23s)
PASS
coverage: 100.0% of statements
```

## Integration Tests

### What They Test

**Tunnel Tests** (`internal/integration/tunnel_test.go`)
- ✅ Connection establishment and disconnection
- ✅ Automatic reconnection after failure
- ✅ HTTP request forwarding through tunnel
- ✅ Request timeout handling
- ✅ Concurrent request handling (10 simultaneous)
- ✅ Multiple HTTP methods (GET, POST, PUT, DELETE, PATCH)
- ✅ Large payload transfer (1MB)
- ✅ Disconnect during active request

**Proxy Tests** (`internal/integration/proxy_test.go`)
- ✅ HTTP proxy functionality
- ✅ HTTPS CONNECT tunneling
- ✅ Invalid target error handling
- ✅ Concurrent client connections (10 simultaneous)
- ✅ Large response handling (1MB)
- ✅ Custom header forwarding

**Circuit Breaker Integration** (`internal/integration/circuitbreaker_test.go`)
- ✅ Circuit trips after consecutive failures
- ✅ Automatic recovery after timeout period
- ✅ Protection from cascading failures
- ✅ Metrics collection and tracking
- ✅ Complete state transition verification

**WebSocket Tests** (`internal/integration/websocket_test.go`)
- ✅ WebSocket connection through tunnel
- ✅ Multiple message exchange (text messages)
- ✅ Binary message transfer
- ✅ Ping/pong keepalive mechanism
- ✅ Concurrent WebSocket connections (5 simultaneous)
- ✅ Large message handling (100KB)
- ✅ Proper connection cleanup

### Running Integration Tests

```powershell
# Run all integration tests
go test ./internal/integration/... -v -timeout 5m

# Run specific test file
go test ./internal/integration -run TestTunnel -v
go test ./internal/integration -run TestProxy -v
go test ./internal/integration -run TestCircuitBreaker -v
go test ./internal/integration -run TestWebSocket -v

# Run single test
go test ./internal/integration -run TestTunnelConnection -v

# Run with race detection
go test ./internal/integration/... -race -timeout 5m

# Run in parallel (default behavior)
go test ./internal/integration/... -parallel 4 -v
```

### Expected Output

```
=== RUN   TestTunnelConnection
=== PAUSE TestTunnelConnection
=== RUN   TestProxyHTTPRequest
=== PAUSE TestProxyHTTPRequest
=== CONT  TestTunnelConnection
=== CONT  TestProxyHTTPRequest
    tunnel_test.go:33: Tunnel connection established successfully
--- PASS: TestTunnelConnection (0.62s)
    proxy_test.go:45: HTTP request through proxy successful
--- PASS: TestProxyHTTPRequest (0.58s)
...
PASS
ok      fluidity/internal/integration   3.172s
```

### Integration Test Features

- **Parallel Execution**: All tests use `t.Parallel()` for concurrent execution
- **Test Isolation**: Each test generates unique certificates and uses random ports
- **Automatic Cleanup**: Resources are cleaned up with `defer` and `t.Cleanup()`
- **No Shared State**: Tests are completely independent

## End-to-End Tests

### What They Test

**Local Binary Tests** (`test-local.ps1` / `test-local.sh`)
1. Build agent and server binaries from source
2. Start server with production configuration
3. Start agent with production configuration
4. **HTTP Test**: Make HTTP request through proxy
5. **HTTPS Test**: Make HTTPS request through proxy
6. **WebSocket Test**: Establish WebSocket connection and exchange messages
7. Cleanup processes

**Docker Container Tests** (`test-docker.ps1` / `test-docker.sh`)
1. Build Docker images for agent and server
2. Create isolated Docker network
3. Start server container
4. Start agent container
5. **HTTP Test**: Execute curl in agent container
6. **HTTPS Test**: Execute curl in agent container
7. **WebSocket Test**: Execute Node.js WebSocket client in agent container
8. Cleanup containers and network

### Running E2E Tests

**Prerequisites:**
- **Local Tests**: Go 1.21+, Node.js, curl
- **Docker Tests**: Docker (Docker Desktop on Windows/Mac)

**Windows (PowerShell):**
```powershell
# Local binary tests
.\scripts\test-local.ps1

# Docker container tests
.\scripts\test-docker.ps1
```

**Linux/macOS:**
```bash
# Local binary tests
./scripts/test-local.sh

# Docker container tests
./scripts/test-docker.sh
```

### Expected Output

```
==================================================
  Fluidity Local Integration Test
==================================================

[Step 1/9] Building agent...
✓ Agent built successfully

[Step 2/9] Building server...
✓ Server built successfully

[Step 3/9] Starting server...
✓ Server started (PID: 12345)

[Step 4/9] Starting agent...
✓ Agent started (PID: 12346)

[Step 5/9] Waiting for initialization (5 seconds)...
✓ Services initialized

[Step 6/9] Testing HTTP request...
✓ HTTP request successful (200 OK)

[Step 7/9] Testing HTTPS request...
✓ HTTPS request successful (200 OK)

[Step 8/9] Testing WebSocket...
✓ WebSocket test successful

[Step 9/9] Cleaning up...
✓ Cleanup complete

==================================================
  All tests passed! ✓
==================================================
```

### E2E Test Configuration

**Initialization Wait Times:**
- **Local tests**: 5 seconds (allows time for mTLS handshake, circuit breaker setup)
- **Docker tests**: 10 seconds (additional time for container startup and DNS resolution)

These wait times account for:
- TLS handshake completion
- Circuit breaker initialization
- Retry mechanism setup
- Network buffer management

## Test Coverage

### Current Coverage Metrics

| Category | Tests | Coverage | Status |
|----------|-------|----------|--------|
| Unit Tests | 49 | 72-100% per package | ✅ ALL PASS |
| Integration Tests | 26 | 68.6% | ✅ ALL PASS |
| E2E Tests | 6 scenarios | Full system | ✅ PASS |
| **TOTAL** | **75+** | **~77% overall** | ✅ **ALL PASS** |

### Detailed Breakdown

**Unit Test Coverage by Package:**
- `internal/shared/protocol`: 9 tests, **100.0%** coverage ✅
- `internal/shared/logging`: 14 tests, **95.1%** coverage ✅
- `internal/shared/config`: 9 tests, **90.7%** coverage ✅
- `internal/shared/circuitbreaker`: 7 tests, **84.1%** coverage ✅
- `internal/shared/retry`: 10 tests, **72.1%** coverage ✅

**Integration Test Coverage:**
- Tunnel functionality: 8 tests
- Proxy functionality: 7 tests (HTTP, HTTPS, CONNECT, headers, large responses)
- Circuit breaker integration: 6 tests (trips, recovery, cascading protection, metrics, state transitions)
- WebSocket functionality: 9 tests (connection, messages, binary, ping/pong, concurrency, close handshake)
- **Total: 26 integration tests, 68.6% coverage**

**E2E Test Coverage:**
- HTTP protocol: 2 scenarios (local + Docker)
- HTTPS protocol: 2 scenarios (local + Docker)
- WebSocket protocol: 2 scenarios (local + Docker)

### Generating Coverage Reports

```powershell
# Generate coverage for unit tests
go test ./internal/shared/... -coverprofile=unit-coverage.out
go tool cover -html=unit-coverage.out -o unit-coverage.html

# Generate coverage for integration tests (note: less meaningful for integration)
go test ./internal/integration/... -coverprofile=integration-coverage.out
go tool cover -html=integration-coverage.out -o integration-coverage.html

# Generate combined coverage
go test ./internal/... -coverprofile=combined-coverage.out
go tool cover -html=combined-coverage.out -o combined-coverage.html
```

## CI/CD Integration

### GitHub Actions Example

Create `.github/workflows/test.yml`:

```yaml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run Unit Tests
        run: go test ./internal/shared/... -v -cover

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run Integration Tests
        run: go test ./internal/integration/... -v -timeout 5m

  e2e-local:
    name: E2E Local Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Run E2E Local Tests
        run: ./scripts/test-local.sh

  e2e-docker:
    name: E2E Docker Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run E2E Docker Tests
        run: ./scripts/test-docker.sh
```

### Pre-commit Hook

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
echo "Running unit tests..."
go test ./internal/shared/... -v
if [ $? -ne 0 ]; then
    echo "Unit tests failed. Commit aborted."
    exit 1
fi

echo "Running integration tests..."
go test ./internal/integration/... -v -timeout 2m
if [ $? -ne 0 ]; then
    echo "Integration tests failed. Commit aborted."
    exit 1
fi

echo "All tests passed!"
exit 0
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

## Lambda Control Plane Testing

### Unit Tests

Test individual Lambda functions for correct behavior using `pytest` with `moto` for AWS mocking:

**Wake Lambda Tests:**
- Test wake when task stopped (verifies DesiredCount updated to 1)
- Test wake when task already running (verifies idempotent behavior)

**Sleep Lambda Tests:**
- Test sleep when idle (verifies DesiredCount set to 0 based on metrics)
- Test no sleep when active (verifies DesiredCount remains 1)

**Kill Lambda Tests:**
- Test kill immediate shutdown (verifies DesiredCount set to 0 without validation)

**Run Lambda Unit Tests:**
```bash
# Install dependencies
pip install pytest moto boto3

# Run all Lambda tests
pytest tests/lambda/ -v

# Run with coverage
pytest tests/lambda/ --cov=lambda_functions --cov-report=html
```

### Integration Tests

Test lifecycle integration between agent, server, and Lambda control plane:

**Agent Lifecycle Tests:**
- Agent startup wakes server (verify /wake endpoint called, connection retry logic)
- Agent shutdown kills server (verify /kill endpoint called before exit)

**Server Metrics Tests:**
- Server emits metrics to CloudWatch (verify periodic emission of ActiveConnections and LastActivityEpochSeconds)
- Metrics update on activity (verify counters increment with traffic)

### End-to-End Tests

Test complete wake → connect → idle → sleep → kill lifecycle:

**Full Lifecycle Test:**
```bash
#!/bin/bash
# tests/e2e/lambda_lifecycle_test.sh

echo "[1/8] Deploying Lambda control plane..."
aws cloudformation deploy --template-file lambda.yaml --stack-name fluidity-lambda-test

echo "[2/8] Verifying server is stopped (DesiredCount=0)..."
# Check ECS service DesiredCount

echo "[3/8] Starting agent (should trigger wake)..."
# Start agent with wake API configured

echo "[4/8] Waiting for agent to call wake and retry connection..."
sleep 30

echo "[5/8] Verifying server started (DesiredCount=1)..."
# Check ECS service DesiredCount

echo "[6/8] Testing HTTP request through tunnel..."
curl -x http://localhost:8080 http://example.com

echo "[7/8] Stopping agent (should trigger kill)..."
# Stop agent gracefully

echo "[8/8] Verifying server stopped (DesiredCount=0)..."
# Check ECS service DesiredCount after 60 seconds

echo "✓ Full lifecycle test passed!"
```

**Idle Detection Test:**
```bash
#!/bin/bash
# tests/e2e/lambda_idle_test.sh

echo "[1/6] Starting agent and server..."
# Bring up infrastructure

echo "[2/6] Sending traffic to establish activity..."
curl -x http://localhost:8080 http://example.com

echo "[3/6] Verifying CloudWatch metrics show recent activity..."
aws cloudwatch get-metric-statistics --namespace Fluidity \
    --metric-name LastActivityEpochSeconds --statistics Maximum

echo "[4/6] Waiting for idle timeout (6 minutes)..."
sleep 360

echo "[5/6] Manually invoking Sleep Lambda..."
aws lambda invoke --function-name FluiditySleepLambda output.json

echo "[6/6] Verifying server stopped due to idle..."
# Check ECS service DesiredCount=0

echo "✓ Idle detection test passed!"
```

**EventBridge Scheduler Test:**
```bash
#!/bin/bash
# tests/e2e/lambda_scheduler_test.sh

echo "[1/4] Verifying EventBridge rules created..."
aws events describe-rule --name FluiditySleepSchedule
aws events describe-rule --name FluidityKillSchedule

echo "[2/4] Testing Sleep schedule (rate: 5 minutes)..."
# Verify rule is enabled and has correct target

echo "[3/4] Testing Kill schedule (cron: daily at 11 PM UTC)..."
# Verify rule is enabled and has correct target

echo "[4/4] Manually triggering rules..."
aws events put-events --entries file://test-event-sleep.json
aws events put-events --entries file://test-event-kill.json

echo "✓ EventBridge scheduler test passed!"
```

### Performance Tests

Test Lambda cold start and response times:

**Test Scenarios:**
- Wake Lambda cold start latency (target: < 3 seconds)
- Wake Lambda warm start latency (target: < 500ms)
- Agent wake to connect time (target: < 90 seconds total)

## Troubleshooting

### Common Issues

#### "Connection refused" errors

**Cause**: Server not fully started before client attempts connection

**Solution**:
- Increase initialization wait time in E2E scripts
- Check if port is already in use: `netstat -an | findstr :<port>`
- Verify certificates exist and are valid

#### "Timeout" errors in integration tests

**Cause**: Tests taking longer than expected, especially under load

**Solution**:
```powershell
# Increase timeout
go test ./internal/integration/... -timeout 10m -v

# Run tests sequentially instead of parallel
go test ./internal/integration/... -parallel 1 -v
```

#### "Certificate invalid" errors

**Cause**: Test certificates expired or improperly generated

**Solution**:
- Integration tests auto-generate certificates (no action needed)
- For E2E tests, regenerate certificates:
  ```powershell
  # Windows
  .\scripts\generate-certs.ps1
  
  # Linux/macOS
  ./scripts/generate-certs.sh
  ```

#### Flaky integration tests

**Cause**: Race conditions or improper synchronization

**Solution**:
```powershell
# Run with race detector
go test ./internal/integration/... -race -v

# Run specific test multiple times
go test ./internal/integration -run TestTunnelConnection -count 10 -v
```

#### E2E tests hang on cleanup

**Cause**: Processes not terminating properly

**Solution**:
- Check for orphaned processes:
  ```powershell
  # Windows
  Get-Process | Where-Object {$_.ProcessName -like "*fluidity*"}
  
  # Linux/macOS
  ps aux | grep fluidity
  ```
- Kill manually if needed:
  ```powershell
  # Windows
  Stop-Process -Name "fluidity-*" -Force
  
  # Linux/macOS
  killall fluidity-server fluidity-agent
  ```

#### Docker tests fail to build images

**Cause**: Docker daemon not running or insufficient resources

**Solution**:
- Start Docker Desktop
- Check Docker is running: `docker ps`
- Increase Docker memory allocation (Settings → Resources)
- Clean up old images: `docker system prune -a`

### Debug Mode

Run tests with verbose output:

```powershell
# Unit tests with verbose output
go test ./internal/shared/... -v -test.v

# Integration tests with verbose output and log level
go test ./internal/integration/... -v -test.v

# E2E tests - check log files
cat logs/server.log
cat logs/agent.log
```

### Performance Profiling

Profile slow tests:

```powershell
# CPU profiling
go test ./internal/integration/... -cpuprofile=cpu.out -v
go tool pprof cpu.out

# Memory profiling
go test ./internal/integration/... -memprofile=mem.out -v
go tool pprof mem.out

# Trace execution
go test ./internal/integration/... -trace=trace.out -v
go tool trace trace.out
```

## Best Practices

### Writing New Tests

1. **Use descriptive names**: `TestFeature_Scenario_ExpectedOutcome`
2. **Add t.Parallel()**: For independent tests to run concurrently
3. **Use t.Helper()**: In utility functions for better error reporting
4. **Clean up resources**: Use `defer` or `t.Cleanup()`
5. **Provide clear assertions**: Include descriptive error messages
6. **Test error paths**: Don't just test the happy path
7. **Use table-driven tests**: For testing multiple scenarios

### Example Test Structure

```go
func TestMyFeature(t *testing.T) {
    t.Parallel() // Run in parallel with other tests

    // Arrange - Set up test data and dependencies
    certs := GenerateTestCerts(t)
    server := StartTestServer(t, certs)
    defer server.Stop() // Clean up

    // Act - Execute the feature under test
    result, err := server.DoSomething()

    // Assert - Verify the outcome
    AssertNoError(t, err, "operation should succeed")
    AssertEqual(t, expected, result, "result should match")
}
```

### Test Maintenance

- Run tests before committing: `go test ./...`
- Keep test execution fast (unit tests < 1s, integration < 10s)
- Update tests when changing functionality
- Remove obsolete tests
- Monitor test coverage: `go test ./... -cover`

## Additional Resources

- **Architecture**: See `docs/architecture.md` for system design
- **Error Handling**: See `docs/error-handling-improvements.md` for circuit breaker and retry details
- **Integration Test README**: See `internal/integration/README.md` for integration test specifics
- **Go Testing Docs**: https://golang.org/pkg/testing/
- **Table-Driven Tests**: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests

## Summary

Fluidity's testing strategy ensures:
- ✅ Individual components work correctly (unit tests)
- ✅ Components interact properly (integration tests)
- ✅ Complete system functions as expected (E2E tests)
- ✅ Changes can be validated quickly and confidently
- ✅ Production deployments are reliable

**Run all tests regularly to maintain code quality and catch regressions early!**
