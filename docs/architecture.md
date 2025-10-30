# Architecture# Architecture Design



Fluidity uses a client-server architecture with mTLS authentication and Lambda-based lifecycle management.**Status**: Phase 1 Complete | Phase 2 In Progress



## System Overview---



```## System Overview

┌─────────────────┐                    ┌──────────────────────┐

│  Local Network  │                    │     AWS Cloud        │Fluidity uses a client-server architecture with mTLS authentication and optional Lambda-based lifecycle management.

│                 │                    │                      │

│ ┌─────────────┐ │                    │ ┌─────────────────┐  │```

│ │   Browser   │ │                    │ │Target Websites  │  │┌─────────────────┐                    ┌──────────────────────┐

│ └──────┬──────┘ │                    │ └────────▲────────┘  ││  Local Network  │                    │     AWS Cloud        │

│        │ Proxy  │                    │          │           ││                 │                    │                      │

│ ┌──────▼──────┐ │   mTLS Tunnel      │ ┌────────┴────────┐  ││ ┌─────────────┐ │                    │ ┌─────────────────┐  │

│ │Tunnel Agent │ │◄──────────────────►│ │ Tunnel Server   │  ││ │   Browser   │ │                    │ │Target Websites  │  │

│ │  (Go/Docker)│ │                    │ │(ECS Fargate/Go) │  ││ └──────┬──────┘ │                    │ └────────▲────────┘  │

│ └─────────────┘ │                    │ └─────────────────┘  ││        │ Proxy  │                    │          │           │

│        │ HTTPS  │                    │          │           ││ ┌──────▼──────┐ │   mTLS Tunnel      │ ┌────────┴────────┐  │

│        └────────┼────────────────────┼►│ Lambda Control  │  ││ │Tunnel Agent │ │◄──────────────────►│ │ Tunnel Server   │  │

│                 │                    │ │ (Wake/Sleep)    │  ││ │  (Go/Docker)│ │                    │ │(ECS Fargate/Go) │  │

└─────────────────┘                    └──────────────────────┘│ │             │ │                    │ │                 │  │

```│ │•Wake on start│ │                    │ │•CloudWatch      │  │

│ │•Kill on stop│ │                    │ │ metrics         │  │

## Components│ └─────────────┘ │                    │ └─────────────────┘  │

│        │        │                    │          │           │

### Tunnel Agent (Local)│        │ HTTPS  │                    │ ┌────────▼────────┐  │

- HTTP proxy server (port 8080)│        └────────┼────────────────────┼►│ Lambda Control  │  │

- mTLS client connecting to server│                 │                    │ │ Plane (Wake/    │  │

- Calls Wake Lambda on startup, Kill on shutdown│                 │                    │ │ Sleep/Kill)     │  │

- Auto-reconnects on connection loss└─────────────────┘                    └──────────────────────┘

```

### Tunnel Server (Cloud)

- mTLS server (port 8443)### Components

- Forwards requests to target websites

- Publishes CloudWatch metrics**Tunnel Agent** (Local)

- Handles concurrent connections- HTTP proxy server (port 8080)

- mTLS client to server

### Lambda Control Plane (AWS)- Lifecycle integration (Wake/Kill Lambda calls)

- **Wake**: Starts server (DesiredCount=1)- Automatic reconnection

- **Sleep**: Auto-scales down when idle

- **Kill**: Immediate shutdown**Tunnel Server** (Cloud)

- mTLS server (port 8443)

See [Lambda Functions](lambda.md) for details.- HTTP client for target websites

- CloudWatch metrics emission

## Protocol- Concurrent connection handling



JSON messages over TLS 1.3:**Lambda Control Plane** (AWS)

- **Wake**: Starts ECS service (DesiredCount=1)

```go- **Sleep**: Auto-scales down when idle

type Envelope struct {- **Kill**: Immediate shutdown

    Type    string      `json:"type"`- See [Lambda Functions](lambda.md) for details

    Payload interface{} `json:"payload"`

}---

```

## Communication Protocol

**Message Types**:

- `Request/Response` - HTTP tunneling### Protocol Envelope

- `ConnectRequest/ConnectAck/ConnectData` - HTTPS CONNECT

- `WebSocketOpen/WebSocketMessage/WebSocketClose` - WebSocketJSON-based message framing over TLS 1.3:



## Security```go

type Envelope struct {

### mTLS Configuration    Type    string      `json:"type"`    // "request", "response", "connect", etc.

    Payload interface{} `json:"payload"`

**Certificate Chain**:}

``````

Private CA (self-signed)

├── Server Certificate### Message Types

└── Client Certificate

```**HTTP Tunneling**:

- `Request`: HTTP method, URL, headers, body

**TLS Requirements**:- `Response`: Status code, headers, body

- TLS 1.3 minimum

- Mutual authentication required**HTTPS CONNECT**:

- Certificate validation against private CA- `ConnectRequest`: Host for HTTPS tunnel

- `ConnectAck`: Success/failure

**Agent Config**:- `ConnectData`: Bidirectional stream data

```go

&tls.Config{**WebSocket**:

    Certificates: []tls.Certificate{clientCert},- `WebSocketOpen`: Initiate WebSocket connection

    RootCAs:      caCertPool,- `WebSocketAck`: Connection confirmed

    MinVersion:   tls.VersionTLS13,- `WebSocketMessage`: Frame data

}- `WebSocketClose`: Close connection

```

---

**Server Config**:

```go## Security Architecture

&tls.Config{

    Certificates: []tls.Certificate{serverCert},### mTLS Implementation

    ClientAuth:   tls.RequireAndVerifyClientCert,

    ClientCAs:    caCertPool,**Certificate Chain**:

    MinVersion:   tls.VersionTLS13,```

}Private CA (Self-signed)

```├── Server Certificate (TLS server auth)

└── Client Certificate (TLS client auth)

## Project Structure```



```**Configuration**:

cmd/core/- TLS 1.3 minimum

├── agent/main.go          # Agent entry- Mutual authentication required

└── server/main.go         # Server entry- Certificate validation against private CA

- No InsecureSkipVerify

cmd/lambdas/

├── wake/main.go**Agent TLS Config**:

├── sleep/main.go```go

└── kill/main.go&tls.Config{

    Certificates: []tls.Certificate{clientCert},

internal/core/    RootCAs:      caCertPool,

├── agent/                 # Proxy + tunnel client    MinVersion:   tls.VersionTLS13,

└── server/                # mTLS server + HTTP forwarder}

```

internal/shared/

├── protocol/              # Message definitions**Server TLS Config**:

├── tls/                   # mTLS utilities```go

├── circuitbreaker/        # Failure protection&tls.Config{

└── retry/                 # Retry logic    Certificates: []tls.Certificate{serverCert},

    ClientAuth:   tls.RequireAndVerifyClientCert,

deployments/    ClientCAs:    caCertPool,

├── agent/Dockerfile    MinVersion:   tls.VersionTLS13,

├── server/Dockerfile}

└── cloudformation/        # Infrastructure as Code```

```

---

## Configuration

## Project Structure

### Agent (`agent.yaml`)

```yaml```

server_ip: "3.24.56.78"fluidity/

server_port: 8443├── cmd/

local_proxy_port: 8080│   ├── core/

cert_file: "./certs/client.crt"│   │   ├── agent/main.go          # Agent entry point

key_file: "./certs/client.key"│   │   └── server/main.go         # Server entry point

ca_cert_file: "./certs/ca.crt"│   └── lambdas/

log_level: "info"│       ├── wake/main.go           # Wake Lambda

```│       ├── sleep/main.go          # Sleep Lambda

│       └── kill/main.go           # Kill Lambda

### Server (`server.yaml`)├── internal/

```yaml│   ├── core/

listen_addr: "0.0.0.0"│   │   ├── agent/                 # Agent logic

listen_port: 8443│   │   └── server/                # Server logic

cert_file: "/root/certs/server.crt"│   ├── lambdas/                   # Lambda implementations

key_file: "/root/certs/server.key"│   └── shared/                    # Shared libraries

ca_cert_file: "/root/certs/ca.crt"│       ├── protocol/              # Protocol definitions

log_level: "info"│       ├── tls/                   # mTLS utilities

max_connections: 100│       ├── config/                # Configuration

emit_metrics: true│       ├── logging/               # Logging

```│       ├── circuitbreaker/        # Circuit breaker pattern

│       └── retry/                 # Retry logic

## Reliability Patterns├── deployments/

│   ├── agent/Dockerfile

### Circuit Breaker│   ├── server/Dockerfile

- Failure threshold: 5 consecutive failures│   └── cloudformation/

- Timeout: 30 seconds│       ├── fargate.yaml           # ECS infrastructure

- Protects against cascading failures│       └── lambda.yaml            # Lambda control plane

├── configs/                       # YAML configurations

### Retry Logic├── certs/                         # TLS certificates

- Max attempts: 3└── scripts/                       # Build & test scripts

- Exponential backoff: 1s, 2s, 4s```

- Max delay: 10s

---

### Auto-Reconnection

- Retry interval: 5s## Agent Architecture

- Max duration: 90s (after wake)

### Core Responsibilities

## Deployment Options

1. **HTTP Proxy Server**: Accept browser/app requests on localhost:8080

### Local Development2. **Tunnel Client**: Forward requests via mTLS to server

```3. **Lifecycle Management**: Call Wake Lambda on startup, Kill on shutdown

Host Machine4. **Connection Recovery**: Retry with exponential backoff

├── Server binary (localhost:8443)

├── Agent binary (localhost:8080)### Key Components

└── Certs (./certs/)

```**Proxy Server** (`internal/core/agent/proxy.go`):

- Handles HTTP and HTTPS CONNECT requests

### Docker- Converts HTTP requests to protocol envelopes

```- Returns responses to local clients

├── fluidity-server container (~44MB)

└── fluidity-agent container (~44MB)**Tunnel Connection** (`internal/core/agent/agent.go`):

```- Establishes mTLS connection to server

- Manages request/response correlation

### AWS Fargate- Handles reconnection logic

```

ECS Cluster → Service → Task (0.25 vCPU, 512MB)**Lifecycle Client** (planned):

├── Public IP (dynamic)- Calls Wake Lambda via API Gateway on startup

├── Security Group (port 8443)- Retries connection for configured duration (default 90s)

└── CloudWatch Logs- Calls Kill Lambda on graceful shutdown

```

### Configuration

See [Deployment Guide](deployment.md) for setup instructions.

```yaml

## Monitoring# agent.yaml

server_ip: "3.24.56.78"

### Logsserver_port: 8443

- Structured logging with logruslocal_proxy_port: 8080

- Connection events and errorscert_file: "./certs/client.crt"

- No sensitive data (no credentials, URLs, or POST bodies)key_file: "./certs/client.key"

ca_cert_file: "./certs/ca.crt"

### Metrics (CloudWatch)log_level: "info"

- `ActiveConnections`: Current connection count```

- `LastActivityEpochSeconds`: Unix timestamp

---

Used by Sleep Lambda for idle detection.

## Server Architecture

## Performance

### Core Responsibilities

### Resource Usage

**Agent**: ~20-50MB memory, <5% CPU  1. **mTLS Server**: Accept authenticated agent connections

**Server**: 256 CPU units, 512 MB memory, 100 concurrent connections2. **HTTP Client**: Make requests to target websites

3. **Response Relay**: Return website responses through tunnel

### Optimizations4. **Metrics Emission**: Publish CloudWatch metrics (when enabled)

- TLS session resumption

- HTTP/2 for target requests### Key Components

- Connection pooling

**Tunnel Server** (`internal/core/server/server.go`):

## Security Best Practices- Accepts mTLS connections on port 8443

- Validates client certificates

1. **Certificates**: 2-year validity, secure CA key storage, regular rotation- Handles concurrent requests via goroutines

2. **Network**: Restrict Security Group to known IPs, use private subnets

3. **Access**: API Gateway authentication, IAM least-privilege**HTTP Client**:

4. **Monitoring**: CloudWatch Logs, alarms for errors- Connection pooling for target requests

- Circuit breaker for external failures

## Related Documentation- Retry logic with exponential backoff



- [Deployment Guide](deployment.md) - Setup instructions**Metrics Emitter** (planned):

- [Lambda Functions](lambda.md) - Control plane- Tracks active connections (atomic counter)

- [Development Guide](development.md) - Local development- Records last activity timestamp

- [Testing Guide](testing.md) - Test strategy- Emits CloudWatch metrics every 60s


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
