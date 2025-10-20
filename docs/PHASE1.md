# Phase 1 Implementation - Core Infrastructure

This document describes the Phase 1 implementation of Fluidity, which includes the core infrastructure and basic functionality.

## What's Implemented

### Core Components
- ✅ **Go Project Structure**: Complete project layout with proper package organization
- ✅ **Shared Protocol**: JSON-based request/response protocol for tunnel communication
- ✅ **Logging System**: Structured logging with component-based loggers
- ✅ **Configuration Management**: YAML configuration with CLI overrides and environment variables
- ✅ **TLS Infrastructure**: Basic mTLS configuration loading (certificates not yet generated)

### Tunnel Agent
- ✅ **HTTP Proxy Server**: Local HTTP proxy that intercepts browser requests
- ✅ **Tunnel Client**: mTLS client that connects to the tunnel server
- ✅ **Configuration**: Server IP configuration with CLI override support
- ✅ **Connection Management**: Automatic reconnection on connection loss
- ✅ **CLI Interface**: Command-line interface with configuration options

### Tunnel Server
- ✅ **mTLS Server**: Accepts authenticated connections from agents
- ✅ **HTTP Client**: Makes requests to target websites on behalf of agents
- ✅ **Concurrent Handling**: Processes multiple requests concurrently
- ✅ **Connection Limits**: Configurable maximum connections
- ✅ **CLI Interface**: Command-line interface with configuration options

### Build & Deployment
- ✅ **Docker Support**: Multi-stage Dockerfiles for both components
- ✅ **Build Automation**: Makefile with build, test, and Docker targets
- ✅ **Certificate Generation**: Scripts for generating development certificates

## Current Limitations

### Security
- ⚠️ **No Certificates**: Development certificates not yet generated
- ⚠️ **TLS Verification**: InsecureSkipVerify used for IP-based connections
- ⚠️ **No Authentication**: mTLS setup exists but certificates need to be generated

### Protocol Support
- ⚠️ **HTTP Only**: Only supports regular HTTP requests
- ⚠️ **No CONNECT**: HTTPS tunneling (CONNECT method) not implemented
- ⚠️ **No WebSocket**: WebSocket support not implemented

### Error Handling
- ⚠️ **Basic Error Handling**: Minimal error handling and recovery
- ⚠️ **No Circuit Breaker**: No protection against cascading failures
- ⚠️ **Limited Retry Logic**: Basic retry only for connection attempts

## Quick Start

### Prerequisites
- Go 1.21 or later
- Docker (for containerization)
- OpenSSL (for certificate generation)

### 1. Generate Development Certificates

On Windows:
```powershell
.\scripts\generate-certs.ps1
```

On Linux/macOS:
```bash
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

### 2. Build Applications

```bash
make build
```

### 3. Run Server

```bash
make run-server
```

Or manually:
```bash
go run cmd/server/main.go --config ./configs/server.yaml
```

### 4. Run Agent

First, update the agent configuration with your server IP:
```bash
go run cmd/agent/main.go --config ./configs/agent.yaml --server-ip YOUR_SERVER_IP
```

### 5. Test the Tunnel

Configure your browser to use `127.0.0.1:8080` as an HTTP proxy, then try browsing to any HTTP website.

## Configuration

### Agent Configuration (`configs/agent.yaml`)
```yaml
server_ip: ""  # Set this to your tunnel server IP
server_port: 8443
local_proxy_port: 8080
cert_file: "./certs/client.crt"
key_file: "./certs/client.key"
ca_cert_file: "./certs/ca.crt"
log_level: "info"
```

### Server Configuration (`configs/server.yaml`)
```yaml
listen_addr: "0.0.0.0"
listen_port: 8443
cert_file: "./certs/server.crt"
key_file: "./certs/server.key"
ca_cert_file: "./certs/ca.crt"
log_level: "info"
max_connections: 100
```

## Docker Usage

### Build Images
```bash
make docker-build
```

### Run Server Container
```bash
docker run -d \
  --name fluidity-server \
  -p 8443:8443 \
  -v $(pwd)/certs:/root/certs:ro \
  fluidity-server:latest
```

### Run Agent Container
```bash
docker run -d \
  --name fluidity-agent \
  -p 8080:8080 \
  -v $(pwd)/certs:/root/certs:ro \
  -e FLUIDITY_SERVER_IP=YOUR_SERVER_IP \
  fluidity-agent:latest
```

## Development

### Project Structure
```
fluidity/
├── cmd/                 # Main applications
│   ├── agent/          # Agent CLI
│   └── server/         # Server CLI
├── internal/           # Private packages
│   ├── agent/          # Agent-specific logic
│   ├── server/         # Server-specific logic
│   └── shared/         # Shared components
├── deployments/        # Docker configurations
├── configs/           # Configuration files
├── certs/             # Certificates (generated)
├── scripts/           # Build and utility scripts
└── docs/              # Documentation
```

### Building
```bash
# Build both components
make build

# Build for Windows
make build-windows

# Clean build artifacts
make clean

# Run tests
make test

# Format code
make fmt
```

## What's Next (Phase 1.5 - Before Phase 2)

### Security Testing & Validation
1. **EDR/Security Tool Testing**: Test functionality against endpoint detection tools
   - Analyze behavior with CrowdStrike, Carbon Black, Windows Defender, etc.
   - Identify potential security tool triggers or alerts
   - Document any detection patterns or behavioral analysis concerns
   
2. **Security Tool Mitigation**: Implement changes to avoid triggering security monitoring
   - Adjust network communication patterns if needed
   - Review and optimize process behavior and resource usage
   - Ensure legitimate use patterns don't trigger false positives
   
3. **Integration Testing**: Develop comprehensive integration test suite
   - End-to-end HTTP request/response testing
   - End-to-end HTTPS CONNECT tunneling testing
   - Connection recovery and error handling scenarios
   - Certificate validation and mTLS authentication flows
   - Multi-request concurrent handling tests
   - Performance and stress testing

### Phase 2 Goals (After Phase 1.5)

1. **Cloud Deployment**: Deploy to chosen cloud provider
2. **WebSocket Support**: Add WebSocket protocol support
3. **Enhanced Security**: Improve certificate management and validation
4. **Better Error Handling**: Add circuit breakers and improved retry logic
5. **Performance Optimization**: Connection pooling and request optimization

## Known Issues

1. **Certificate Generation**: Certificates must be generated manually before first run
2. **HTTPS Tunneling**: Browser HTTPS requests will fail (CONNECT not implemented)
3. **Error Recovery**: Limited error recovery and user feedback
4. **Platform Specific**: Certificate generation script requires OpenSSL

## Troubleshooting

### "Connection Refused"
- Ensure server is running and listening on the correct port
- Check firewall settings on server
- Verify server IP configuration in agent

### "Certificate Errors"
- Run certificate generation script first
- Ensure certificates are in the correct location
- Check certificate file permissions

### "Proxy Not Working"
- Verify browser proxy settings (127.0.0.1:8080)
- Check agent logs for connection status
- Try HTTP-only websites first (no HTTPS)

For more details, see the main project documentation and architecture documents.

---

## Progress Update — 2025-10-20

- Built and ran Docker scratch images for both server and agent
- Generated development mTLS certificates via `scripts/generate-certs.ps1`
- Started containers on a user-defined Docker network (`fluidity-net`)
- Agent successfully established a TLS connection (TLS 1.3) to the server using client certs
- Verified end-to-end HTTP proxying with:

  ```powershell
  curl.exe -x http://127.0.0.1:8080 http://example.com -I
  ```

- Current limitation remains: HTTPS tunneling (CONNECT) not implemented yet — test with HTTP sites