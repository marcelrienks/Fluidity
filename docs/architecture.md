# Architecture Design

**Status**: Phase 1 Complete | Phase 2 In Progress

---

## System Overview

Fluidity uses a client-server architecture with mTLS authentication and optional Lambda-based lifecycle management.

```
┌─────────────────┐                    ┌──────────────────────┐
│  Local Network  │                    │     AWS Cloud        │
│                 │                    │                      │
│ ┌─────────────┐ │                    │ ┌─────────────────┐  │
│ │   Browser   │ │                    │ │Target Websites  │  │
│ └──────┬──────┘ │                    │ └────────▲────────┘  │
│        │ Proxy  │                    │          │           │
│ ┌──────▼──────┐ │   mTLS Tunnel      │ ┌────────┴────────┐  │
│ │Tunnel Agent │ │◄──────────────────►│ │ Tunnel Server   │  │
│ │  (Go/Docker)│ │                    │ │(ECS Fargate/Go) │  │
│ │             │ │                    │ │                 │  │
│ │•Wake on start│ │                    │ │•CloudWatch      │  │
│ │•Kill on stop│ │                    │ │ metrics         │  │
│ └─────────────┘ │                    │ └─────────────────┘  │
│        │        │                    │          │           │
│        │ HTTPS  │                    │ ┌────────▼────────┐  │
│        └────────┼────────────────────┼►│ Lambda Control  │  │
│                 │                    │ │ Plane (Wake/    │  │
│                 │                    │ │ Sleep/Kill)     │  │
└─────────────────┘                    └──────────────────────┘
```

### Components

**Tunnel Agent** (Local)
- HTTP proxy server (port 8080)
- mTLS client to server
- Lifecycle integration (Wake/Kill Lambda calls)
- Automatic reconnection

**Tunnel Server** (Cloud)
- mTLS server (port 8443)
- HTTP client for target websites
- CloudWatch metrics emission
- Concurrent connection handling

**Lambda Control Plane** (AWS)
- **Wake**: Starts ECS service (DesiredCount=1)
- **Sleep**: Auto-scales down when idle
- **Kill**: Immediate shutdown
- See [Lambda Functions](lambda.md) for details

---

## Communication Protocol

### Protocol Envelope

JSON-based message framing over TLS 1.3:

```go
type Envelope struct {
    Type    string      `json:"type"`    // "request", "response", "connect", etc.
    Payload interface{} `json:"payload"`
}
```

### Message Types

**HTTP Tunneling**:
- `Request`: HTTP method, URL, headers, body
- `Response`: Status code, headers, body

**HTTPS CONNECT**:
- `ConnectRequest`: Host for HTTPS tunnel
- `ConnectAck`: Success/failure
- `ConnectData`: Bidirectional stream data

**WebSocket**:
- `WebSocketOpen`: Initiate WebSocket connection
- `WebSocketAck`: Connection confirmed
- `WebSocketMessage`: Frame data
- `WebSocketClose`: Close connection

---

## Security Architecture

### mTLS Implementation

**Certificate Chain**:
```
Private CA (Self-signed)
├── Server Certificate (TLS server auth)
└── Client Certificate (TLS client auth)
```

**Configuration**:
- TLS 1.3 minimum
- Mutual authentication required
- Certificate validation against private CA
- No InsecureSkipVerify

**Agent TLS Config**:
```go
&tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      caCertPool,
    MinVersion:   tls.VersionTLS13,
}
```

**Server TLS Config**:
```go
&tls.Config{
    Certificates: []tls.Certificate{serverCert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    caCertPool,
    MinVersion:   tls.VersionTLS13,
}
```

---

## Project Structure

```
fluidity/
├── cmd/
│   ├── core/
│   │   ├── agent/main.go          # Agent entry point
│   │   └── server/main.go         # Server entry point
│   └── lambdas/
│       ├── wake/main.go           # Wake Lambda
│       ├── sleep/main.go          # Sleep Lambda
│       └── kill/main.go           # Kill Lambda
├── internal/
│   ├── core/
│   │   ├── agent/                 # Agent logic
│   │   └── server/                # Server logic
│   ├── lambdas/                   # Lambda implementations
│   └── shared/                    # Shared libraries
│       ├── protocol/              # Protocol definitions
│       ├── tls/                   # mTLS utilities
│       ├── config/                # Configuration
│       ├── logging/               # Logging
│       ├── circuitbreaker/        # Circuit breaker pattern
│       └── retry/                 # Retry logic
├── deployments/
│   ├── agent/Dockerfile
│   ├── server/Dockerfile
│   └── cloudformation/
│       ├── fargate.yaml           # ECS infrastructure
│       └── lambda.yaml            # Lambda control plane
├── configs/                       # YAML configurations
├── certs/                         # TLS certificates
└── scripts/                       # Build & test scripts
```

---

## Agent Architecture

### Core Responsibilities

1. **HTTP Proxy Server**: Accept browser/app requests on localhost:8080
2. **Tunnel Client**: Forward requests via mTLS to server
3. **Lifecycle Management**: Call Wake Lambda on startup, Kill on shutdown
4. **Connection Recovery**: Retry with exponential backoff

### Key Components

**Proxy Server** (`internal/core/agent/proxy.go`):
- Handles HTTP and HTTPS CONNECT requests
- Converts HTTP requests to protocol envelopes
- Returns responses to local clients

**Tunnel Connection** (`internal/core/agent/agent.go`):
- Establishes mTLS connection to server
- Manages request/response correlation
- Handles reconnection logic

**Lifecycle Client** (planned):
- Calls Wake Lambda via API Gateway on startup
- Retries connection for configured duration (default 90s)
- Calls Kill Lambda on graceful shutdown

### Configuration

```yaml
# agent.yaml
server_ip: "3.24.56.78"
server_port: 8443
local_proxy_port: 8080
cert_file: "./certs/client.crt"
key_file: "./certs/client.key"
ca_cert_file: "./certs/ca.crt"
log_level: "info"
```

---

## Server Architecture

### Core Responsibilities

1. **mTLS Server**: Accept authenticated agent connections
2. **HTTP Client**: Make requests to target websites
3. **Response Relay**: Return website responses through tunnel
4. **Metrics Emission**: Publish CloudWatch metrics (when enabled)

### Key Components

**Tunnel Server** (`internal/core/server/server.go`):
- Accepts mTLS connections on port 8443
- Validates client certificates
- Handles concurrent requests via goroutines

**HTTP Client**:
- Connection pooling for target requests
- Circuit breaker for external failures
- Retry logic with exponential backoff

**Metrics Emitter** (planned):
- Tracks active connections (atomic counter)
- Records last activity timestamp
- Emits CloudWatch metrics every 60s

### Configuration

```yaml
# server.yaml
listen_addr: "0.0.0.0"
listen_port: 8443
cert_file: "/root/certs/server.crt"
key_file: "/root/certs/server.key"
ca_cert_file: "/root/certs/ca.crt"
log_level: "info"
max_connections: 100
emit_metrics: true          # For Lambda control plane
metrics_interval: "60s"
```

---

## Reliability Patterns

### Circuit Breaker

Protects against cascading failures when target websites are unavailable.

**States**: Closed → Open → Half-Open → Closed

**Configuration**:
- Failure threshold: 5 consecutive failures
- Timeout: 30 seconds
- Half-open test requests: 1

### Retry Logic

Exponential backoff for transient failures.

**Configuration**:
- Max attempts: 3
- Initial delay: 1s
- Backoff multiplier: 2x
- Max delay: 10s

### Connection Recovery

Agent automatically reconnects on connection loss:
- Retry interval: 5s
- Max duration: 90s (after wake call)
- Context-based cancellation

---

## Deployment Architecture

### Local Development

```
Host Machine
├── Server binary (localhost:8443)
├── Agent binary (localhost:8080)
└── Certificates (./certs/)
```

**Use case**: Development and testing

### Docker

```
├── fluidity-server container (~44MB)
│   ├── Alpine Linux + curl
│   ├── Go binary (static)
│   └── Volume-mounted certs
└── fluidity-agent container (~44MB)
    ├── Alpine Linux + curl
    ├── Go binary (static)
    └── Volume-mounted certs
```

**Use case**: Local testing of containerized deployment

### AWS Fargate

```
ECS Cluster
└── ECS Service (fluidity-server)
    └── Fargate Task (on-demand)
        ├── Public IP (dynamic)
        ├── Security Group (port 8443)
        └── CloudWatch Logs
```

**Use case**: Production cloud deployment  
**Details**: See [fargate.md](fargate.md)

### Lambda Control Plane

```
API Gateway
├── /wake → Wake Lambda → ECS DesiredCount=1
└── /kill → Kill Lambda → ECS DesiredCount=0

EventBridge
├── rate(5 min) → Sleep Lambda → Check metrics → Scale down if idle
└── cron(0 23 * * ? *) → Kill Lambda → Nightly shutdown
```

**Use case**: Automated lifecycle with cost optimization  
**Details**: See [lambda.md](lambda.md)

---

## Performance Considerations

### Concurrency

- **Agent**: Handles multiple browser connections concurrently
- **Server**: Goroutine per request, channel-based coordination
- **Connection pooling**: Reuse connections to target websites

### Resource Limits

**Agent**:
- Memory: ~20-50MB (idle), ~100MB (active)
- CPU: Minimal (<5% on modern hardware)

**Server (Fargate)**:
- CPU: 256 (0.25 vCPU)
- Memory: 512 MB
- Concurrent connections: 100 (configurable)

### Network Optimization

- TLS session resumption
- HTTP/2 for target requests (where supported)
- Efficient binary serialization (JSON)

---

## Monitoring & Observability

### Logging

**Structured logging** with logrus:
- Startup/shutdown events
- Connection state changes
- Endpoint routing (domain only, no full URLs)
- Error conditions

**No sensitive data**: No credentials, query parameters, or POST bodies

### Metrics (Server)

**CloudWatch Custom Metrics** (Namespace: `Fluidity`):
- `ActiveConnections`: Current connection count
- `LastActivityEpochSeconds`: Unix timestamp of last activity

**Used by**: Sleep Lambda for idle detection

### Health Checks

- Server: TCP port 8443 (Fargate health check)
- Agent: Successful tunnel connection

---

## Security Considerations

### Threat Model

**Protected Against**:
- Unauthorized server access (mTLS)
- Man-in-the-middle attacks (certificate validation)
- Eavesdropping (TLS 1.3 encryption)

**User Responsibility**:
- Certificate private key protection
- Firewall compliance
- Target website security

### Best Practices

1. **Certificate Management**:
   - 2-year validity
   - Secure storage of CA private key
   - Regular rotation

2. **Network Security**:
   - Restrict Security Group to known IPs
   - Use private subnets where possible
   - Enable VPC Flow Logs

3. **Access Control**:
   - API Gateway authentication (API keys)
   - IAM least-privilege for Lambdas
   - CloudWatch Logs encryption

4. **Operational Security**:
   - Monitor CloudWatch Logs for anomalies
   - Set up alarms for errors/throttling
   - Regular security updates

---

## Implementation Status

### ✅ Phase 1: Core Infrastructure (Complete)

- HTTP/HTTPS/WebSocket tunneling
- mTLS authentication with private CA
- Circuit breaker and retry patterns
- Docker containerization
- Comprehensive testing (75+ tests)
- Cross-platform support

### 🚧 Phase 2: Lambda Control Plane (In Progress)

- [x] Lambda functions implemented (Wake/Sleep/Kill)
- [ ] CloudFormation templates
- [ ] Agent lifecycle integration
- [ ] Server metrics emission
- [ ] End-to-end testing

### 📋 Phase 3: Production (Planned)

- CI/CD pipeline
- Security hardening
- Performance optimization
- Enhanced monitoring

---

## Related Documentation

- **[Deployment Guide](deployment.md)** - Setup instructions
- **[Lambda Functions](lambda.md)** - Control plane details
- **[Fargate Deployment](fargate.md)** - AWS ECS setup
- **[Docker Guide](docker.md)** - Container details
- **[Testing Guide](testing.md)** - Test strategy
