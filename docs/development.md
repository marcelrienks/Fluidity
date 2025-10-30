# Development Guide

This guide covers local development workflow, architecture, and common patterns for Fluidity developers.

## Project Overview

Fluidity is a Go-based HTTP tunnel solution that bypasses restrictive corporate firewalls. It consists of two main components:
- **Tunnel Server**: Cloud-hosted Go binary that accepts mTLS connections and forwards HTTP/HTTPS requests
- **Tunnel Agent**: Local Go binary that runs a proxy server and establishes secure tunnel to server

The architecture uses mTLS authentication with a private CA and supports both HTTP and HTTPS (via CONNECT) tunneling.

## Prerequisites

- Go >= 1.21
- Make
- Docker (for containerized testing)
- OpenSSL (for certificate generation)

## Quick Build Commands

### Windows (PowerShell)

```powershell
# Build native binaries (for local debugging)
make -f Makefile.win build

# Build Docker images (standard Alpine-based, ~43MB)
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# Build Docker images (scratch-based, for restricted networks)
make -f Makefile.win docker-build-server-scratch
make -f Makefile.win docker-build-agent-scratch
```

### macOS

```bash
make -f Makefile.macos build
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
```

### Linux

```bash
make -f Makefile.linux build
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

## Running Locally

### Windows

```powershell
# Terminal 1 - Server
make -f Makefile.win run-server-local

# Terminal 2 - Agent
make -f Makefile.win run-agent-local
```

### macOS/Linux

```bash
# Terminal 1
make -f Makefile.macos run-server-local  # or Makefile.linux

# Terminal 2
make -f Makefile.macos run-agent-local   # or Makefile.linux
```

## Testing

### Generate Development Certificates

**Required before first run:**

```powershell
# Windows
./scripts/manage-certs.ps1 -Command generate

# macOS/Linux
./scripts/manage-certs.sh
```

See [Certificate Management](certificate-management.md) for detailed options.

### Automated End-to-End Tests

```powershell
# Windows - Test Docker containers
./scripts/test-docker.ps1
./scripts/test-docker.ps1 -SkipBuild  # Use existing binaries

# Windows - Test local binaries
./scripts/test-local.ps1
./scripts/test-local.ps1 -SkipBuild

# macOS/Linux - Test Docker containers
./scripts/test-docker.sh
./scripts/test-docker.sh --skip-build

# macOS/Linux - Test local binaries
./scripts/test-local.sh
./scripts/test-local.sh --skip-build
```

### Manual Testing

```powershell
# Test HTTP tunneling
curl -x http://127.0.0.1:8080 http://example.com -I

# Test HTTPS tunneling (CONNECT method)
curl -x http://127.0.0.1:8080 https://example.com -I
```

**Note**: There are no unit tests (`*_test.go` files) in this codebase. Testing is done via automated integration test scripts and manual validation. See [Testing Guide](testing.md) for details on integration testing.

## Code Architecture

### Component Structure

```
cmd/
├── agent/main.go          # Agent entry point with CLI flags
└── server/main.go         # Server entry point with CLI flags

internal/
├── core/
│   ├── agent/
│   │   ├── config.go      # Agent configuration
│   │   ├── proxy.go       # HTTP proxy server (listens on port 8080)
│   │   └── lifecycle/     # Lifecycle management
│   ├── server/
│   │   ├── config.go      # Server configuration
│   │   ├── server.go      # TLS server + HTTP forwarder
│   │   └── metrics/       # Metrics collection
│   ├── lambdas/           # Lambda function handlers
│   │   ├── kill/
│   │   ├── sleep/
│   │   └── wake/
│   └── shared/
│       ├── circuitbreaker/  # Circuit breaker pattern
│       ├── config/          # Generic config loading (Viper)
│       ├── logger/          # Structured logging (logrus)
│       ├── logging/         # Logger utilities
│       ├── protocol/        # Wire protocol definitions
│       ├── retry/           # Retry logic
│       ├── secretsmanager/  # AWS Secrets Manager integration
│       └── tls/             # mTLS utilities

configs/                   # YAML configuration files
├── agent.local.yaml       # Agent config for native execution
├── server.local.yaml      # Server config for native execution
├── agent.docker.yaml      # Agent config for Docker
└── server.docker.yaml     # Server config for Docker
```

### Protocol and Communication Flow

**Wire Protocol** (`internal/shared/protocol/protocol.go`):
- All messages are JSON-encoded and wrapped in `Envelope` types
- Envelope types: `http_request`, `http_response`, `connect_open`, `connect_ack`, `connect_data`, `connect_close`
- HTTP requests use `Request`/`Response` structs
- HTTPS CONNECT uses TCP tunneling with `ConnectOpen`, `ConnectData`, `ConnectClose` messages

**HTTP Request Flow**:
1. Browser → Agent proxy server (`:8080`)
2. Agent creates `Request` with unique ID
3. Agent sends `Envelope{Type: "http_request", Payload: Request}` over mTLS connection
4. Server receives, forwards to target website via `http.Client`
5. Server sends `Envelope{Type: "http_response", Payload: Response}` back
6. Agent returns response to browser

**HTTPS CONNECT Flow**:
1. Browser sends `CONNECT host:port HTTP/1.1` to agent
2. Agent sends `connect_open` envelope to server
3. Server dials target, sends `connect_ack` 
4. Agent hijacks client connection, sends `200 Connection Established`
5. Bidirectional pumping: `connect_data` envelopes carry raw TCP bytes
6. Either side sends `connect_close` to tear down tunnel

### TLS/mTLS Configuration

- **Agent**: Uses `tls.LoadClientTLSConfig()` with client cert, key, and CA cert
- **Server**: Uses `tls.LoadServerTLSConfig()` with server cert, key, and CA cert
- **Critical**: Agent must set `ServerName` in TLS config for proper SNI
- Certificates are generated by `scripts/manage-certs.{sh,ps1}` using OpenSSL
- Development certs are stored in `certs/` directory (ca.crt, server.crt/key, client.crt/key)

### Configuration Management

- Uses Viper for YAML configuration loading
- Supports CLI flag overrides (e.g., `--server-ip`, `--log-level`)
- Config files have different variants: `.local.yaml` (native), `.docker.yaml` (containers)
- Agent can save updated config when server IP is provided via CLI

### Logging

- Uses logrus for structured logging
- Log levels: debug, info, warn, error
- Component-specific loggers: "agent", "proxy-server", "tunnel-client", "server", "tunnel-server"
- Privacy-conscious: logs only domain names, not full URLs or request bodies

## Build System Notes

### OS-Specific Makefiles

Always use the Makefile for your platform:
- `Makefile.win` - Windows (PowerShell)
- `Makefile.macos` - macOS (bash)
- `Makefile.linux` - Linux (bash)

The default `Makefile` is for reference only.

### Build Types

1. **Native builds** (`make -f Makefile.* build`):
   - Compiles for host OS (`.exe` on Windows)
   - Best for debugging with VS Code breakpoints
   - Uses local config files (`configs/*.local.yaml`)

2. **Docker standard images** (`docker-build-*`):
   - Multi-stage build with `golang:1.21-alpine` builder
   - Runtime: `alpine:latest` with CA certificates
   - Image size: ~43MB (21MB base + 22MB binary)
   - Best for production deployment

3. **Docker scratch images** (`docker-build-*-scratch`):
   - Uses `alpine/curl:latest` as base (includes CA certs and libc)
   - Static Linux binary (`CGO_ENABLED=0`)
   - Image size: ~43MB (20.5MB base + 22-23MB binary)
   - Use when Docker Hub pulls are blocked

## Development Workflow

### Starting from Scratch

1. Generate certificates:
   ```powershell
   .\scripts\manage-certs.ps1 -Command generate  # Windows
   ./scripts/manage-certs.sh                      # macOS/Linux
   ```

2. Build binaries:
   ```powershell
   make -f Makefile.win build  # or appropriate OS makefile
   ```

3. Run server and agent in separate terminals:
   ```powershell
   make -f Makefile.win run-server-local  # Terminal 1
   make -f Makefile.win run-agent-local   # Terminal 2
   ```

4. Test with curl or browser (configure proxy to `localhost:8080`)

### Making Code Changes

**When modifying agent proxy logic** (`internal/core/agent/proxy.go`):
- HTTP requests: `handleHTTPRequest()` method
- HTTPS CONNECT: `handleConnect()` method
- Request ID generation: `generateRequestID()`

**When modifying agent tunnel client** (`internal/core/agent/agent.go`):
- Connection management: `Connect()`, `Disconnect()`, `handleResponses()`
- Request/response handling: `SendRequest()`, channel-based response delivery
- CONNECT protocol: `ConnectOpen()`, `ConnectSend()`, `ConnectClose()`, `ConnectDataChannel()`

**When modifying server tunnel logic** (`internal/core/server/server.go`):
- Connection handling: `handleConnection()`, TLS handshake verification
- HTTP forwarding: `processRequest()`, `sendErrorResponse()`
- CONNECT protocol: `handleConnectOpen()`, `handleConnectData()`, `handleConnectClose()`

**When modifying protocol**:
- Update `internal/shared/protocol/protocol.go` with new message types
- Add corresponding Envelope type handlers in both agent and server
- Update both `agent.go` and `server.go` simultaneously

### Configuration Changes

- Native execution: Edit `configs/agent.local.yaml` or `configs/server.local.yaml`
- Docker execution: Edit `configs/agent.docker.yaml` or `configs/server.docker.yaml`
- Key differences: Docker configs use container hostnames (e.g., `server_ip: fluidity-server`)

### Testing Changes

1. **Local binary test** (fastest for iteration):
   ```powershell
   ./scripts/test-local.ps1
   ```

2. **Docker test** (validates containerization):
   ```powershell
   ./scripts/test-docker.ps1
   ```

3. **Manual browser test**:
   - Configure browser proxy to `127.0.0.1:8080`
   - Visit HTTP and HTTPS sites
   - Check logs in both terminals for traffic flow

## Debugging

### Native Binaries with VS Code

- Set breakpoints in `*.go` files
- Use VS Code Go debugger with native binaries
- Check `configs/*.local.yaml` for proper cert paths

### Docker Containers

- View logs: `docker logs -f fluidity-server` or `fluidity-agent`
- Keep containers running: `./scripts/test-docker.ps1 -KeepContainers`
- Exec into container: `docker exec -it fluidity-agent sh`

### TLS Issues

- Enable TLS debug logging (see comments in `cmd/*/main.go` about GODEBUG)
- Verify cert dates: `openssl x509 -in certs/server.crt -noout -dates`
- Check ServerName is set in client TLS config

### Connection Issues

- Agent can't connect: Check server is listening, firewall rules, cert paths
- Proxy returns 502: Tunnel not established, check agent logs
- Proxy returns 000: Agent not listening or server unreachable

## Important Patterns

### Always Use Envelopes for Wire Protocol

```go
// Sending
env := protocol.Envelope{Type: "http_request", Payload: req}
encoder.Encode(env)

// Receiving
var env protocol.Envelope
decoder.Decode(&env)
// Then type-assert and unmarshal env.Payload
```

### Request ID Tracking

Every HTTP request gets a unique ID (`generateRequestID()`) used to correlate requests and responses across the tunnel.

### Graceful Shutdown

Both agent and server use context cancellation and wait groups for clean shutdown on SIGINT/SIGTERM.

### Reconnection Logic

Agent automatically reconnects if tunnel connection drops (see `cmd/agent/main.go` connection management goroutine).

### CONNECT Tunneling

HTTPS uses HTTP CONNECT method with connection hijacking, then raw TCP pumping via `connect_data` messages.

## Dependencies

Key external packages:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/sirupsen/logrus` - Structured logging

Standard library usage:
- `crypto/tls` - mTLS connections
- `net/http` - HTTP proxy and client
- `encoding/json` - Wire protocol serialization

## Project Status

Currently in **Phase 2**: Core infrastructure is complete and functional. Both HTTP and HTTPS tunneling are fully working. Lambda control plane for lifecycle automation is in development. The system is ready for local development and Docker containerization, with production deployment pending.

See [Development Plan](plan.md) for detailed roadmap.

## Related Documentation

- [Testing Guide](testing.md) - Unit and integration testing
- [Architecture](architecture.md) - System design deep dive
- [Certificate Management](certificate-management.md) - Certificate generation and AWS Secrets Manager integration
- [Docker Deployment](docker.md) - Containerization details
- [Deployment Guide](deployment.md) - All deployment options
