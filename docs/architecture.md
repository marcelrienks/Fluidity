# Architecture Design Document
# Fluidity - HTTP Tunnel Solution

---

## 1. Architecture Overview

### 1.1 System Architecture
Fluidity consists of two main Go applications communicating over mTLS:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              FLUIDITY ARCHITECTURE                         │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ┌─────────────────────┐       mTLS Tunnel       ┌─────────────────────┐   │
│  │   LOCAL NETWORK     │◄───────────────────────►│   CLOUD PROVIDER    │   │
│  │                     │                         │                     │   │
│  │  ┌───────────────┐  │                         │  ┌───────────────┐  │   │
│  │  │ Tunnel Agent  │  │                         │  │ Tunnel Server │  │   │
│  │  │   (Go App)    │  │                         │  │   (Go App)    │  │   │
│  │  │  in Docker    │  │                         │  │  in Docker    │  │   │
│  │  └───────────────┘  │                         │  └───────────────┘  │   │
│  │          │          │                         │          │          │   │
│  │          │          │                         │          │          │   │
│  │  ┌───────────────┐  │                         │  ┌───────────────┐  │   │
│  │  │Local Browser/ │  │                         │  │Target Websites│  │   │
│  │  │ Application   │  │                         │  │   (Internet)  │  │   │
│  │  └───────────────┘  │                         │  └───────────────┘  │   │
│  └─────────────────────┘                         └─────────────────────┘   │
└────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

#### Tunnel Agent (Local)
- **HTTP Proxy Server**: Acts as local HTTP proxy for browser/applications
- **mTLS Client**: Establishes secure connection to tunnel server
- **Traffic Forwarding**: Forwards HTTP requests through tunnel
- **Configuration Management**: Handles server IP configuration and updates

#### Tunnel Server (Cloud)
- **mTLS Server**: Accepts authenticated connections from agents
- **HTTP Client**: Makes requests to target websites on behalf of agent
- **Response Relay**: Returns website responses through tunnel
- **Connection Management**: Handles multiple concurrent requests

---

## 2. Go Application Structure

### 2.1 Project Layout
```
fluidity/
├── cmd/
│   ├── agent/           # Agent CLI application
│   │   └── main.go
│   └── server/          # Server CLI application
│       └── main.go
├── internal/
│   ├── agent/           # Agent-specific logic
│   │   ├── proxy/       # HTTP proxy server
│   │   ├── tunnel/      # Tunnel client
│   │   ├── config/      # Configuration management
│   │   └── cli/         # CLI handling
│   ├── server/          # Server-specific logic
│   │   ├── tunnel/      # Tunnel server
│   │   ├── proxy/       # HTTP client for target requests
│   │   └── config/      # Configuration management
│   ├── shared/          # Shared components
│   │   ├── tls/         # mTLS certificate handling
│   │   ├── protocol/    # Tunnel protocol definition
│   │   ├── logging/     # Logging utilities
│   │   └── config/      # Common configuration
│   └── certs/           # Certificate management utilities
├── pkg/                 # Public packages (if needed)
├── deployments/
│   ├── agent/           # Agent Docker configuration
│   │   └── Dockerfile
│   ├── server/          # Server Docker configuration
│   │   └── Dockerfile
│   └── compose/         # Docker Compose files
├── configs/             # Configuration files
├── certs/               # Certificate storage
├── scripts/             # Build and deployment scripts
├── docs/                # Documentation
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 2.2 Key Go Packages and Dependencies

#### Core Dependencies
```go
// Core HTTP and networking
"net/http"
"net"
"context"
"crypto/tls"

// mTLS and certificates
"crypto/x509"
"crypto/rsa"
"crypto/rand"
"encoding/pem"

// Configuration and CLI
"github.com/spf13/cobra"     // CLI framework
"github.com/spf13/viper"     // Configuration management
"gopkg.in/yaml.v3"           // YAML configuration

// Logging
"github.com/sirupsen/logrus" // Structured logging
"go.uber.org/zap"            // High-performance logging (alternative)

// HTTP utilities
"github.com/gorilla/mux"     // HTTP router (if needed)

// Containerization
"github.com/docker/docker"   // Docker integration (if needed)
```

---

## 3. Detailed Component Design

### 3.1 Tunnel Agent Architecture

```go
// Package: internal/agent
package agent

import (
    "context"
    "crypto/tls"
    "net/http"
    "sync"
    
    "fluidity/internal/shared/protocol"
    "fluidity/internal/shared/logging"
)

// Agent represents the tunnel agent
type Agent struct {
    config     *Config
    tlsConfig  *tls.Config
    proxyServer *http.Server
    tunnelConn  *TunnelConnection
    logger     *logging.Logger
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

// Config holds agent configuration
type Config struct {
    ServerIP       string `yaml:"server_ip"`
    ServerPort     int    `yaml:"server_port"`
    LocalProxyPort int    `yaml:"local_proxy_port"`
    CertFile       string `yaml:"cert_file"`
    KeyFile        string `yaml:"key_file"`
    CACertFile     string `yaml:"ca_cert_file"`
    LogLevel       string `yaml:"log_level"`
}

// TunnelConnection manages the mTLS connection to server
type TunnelConnection struct {
    conn      *tls.Conn
    mu        sync.RWMutex
    connected bool
    requests  map[string]chan *protocol.Response
}
```

#### 3.1.1 HTTP Proxy Component
```go
// Package: internal/agent/proxy
package proxy

import (
    "net/http"
    "net/http/httputil"
    "net/url"
)

// ProxyServer handles local HTTP proxy requests
type ProxyServer struct {
    server     *http.Server
    tunnelConn *TunnelConnection
    logger     *logging.Logger
}

// NewProxyServer creates a new proxy server
func NewProxyServer(port int, tunnelConn *TunnelConnection) *ProxyServer {
    proxy := &ProxyServer{
        tunnelConn: tunnelConn,
        logger:     logging.NewLogger("proxy"),
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", proxy.handleRequest)
    
    proxy.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", port),
        Handler: mux,
    }
    
    return proxy
}

// handleRequest processes incoming HTTP requests
func (p *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
    // Convert HTTP request to tunnel protocol
    tunnelReq := &protocol.Request{
        ID:     generateRequestID(),
        Method: r.Method,
        URL:    r.URL.String(),
        Headers: convertHeaders(r.Header),
        Body:   readBody(r.Body),
    }
    
    // Send through tunnel and get response
    resp, err := p.tunnelConn.SendRequest(tunnelReq)
    if err != nil {
        p.logger.Error("Failed to send request through tunnel", err)
        http.Error(w, "Tunnel error", http.StatusBadGateway)
        return
    }
    
    // Write response back to client
    writeResponse(w, resp)
}
```

#### 3.1.2 Tunnel Client Component
```go
// Package: internal/agent/tunnel
package tunnel

import (
    "crypto/tls"
    "encoding/json"
    "net"
    "sync"
    
    "fluidity/internal/shared/protocol"
)

// Client manages the tunnel connection to server
type Client struct {
    config     *tls.Config
    serverAddr string
    conn       *tls.Conn
    mu         sync.RWMutex
    requests   map[string]chan *protocol.Response
    logger     *logging.Logger
}

// Connect establishes mTLS connection to server
func (c *Client) Connect() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    conn, err := tls.Dial("tcp", c.serverAddr, c.config)
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    
    c.conn = conn
    c.logger.Info("Connected to tunnel server", "addr", c.serverAddr)
    
    // Start response handler
    go c.handleResponses()
    
    return nil
}

// SendRequest sends request through tunnel and waits for response
func (c *Client) SendRequest(req *protocol.Request) (*protocol.Response, error) {
    c.mu.RLock()
    if c.conn == nil {
        c.mu.RUnlock()
        return nil, fmt.Errorf("not connected to server")
    }
    c.mu.RUnlock()
    
    // Create response channel
    respChan := make(chan *protocol.Response, 1)
    c.mu.Lock()
    c.requests[req.ID] = respChan
    c.mu.Unlock()
    
    // Send request
    encoder := json.NewEncoder(c.conn)
    if err := encoder.Encode(req); err != nil {
        delete(c.requests, req.ID)
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    
    // Wait for response
    select {
    case resp := <-respChan:
        return resp, nil
    case <-time.After(30 * time.Second):
        delete(c.requests, req.ID)
        return nil, fmt.Errorf("request timeout")
    }
}
```

### 3.2 Tunnel Server Architecture

```go
// Package: internal/server
package server

import (
    "context"
    "crypto/tls"
    "net"
    "sync"
    
    "fluidity/internal/shared/protocol"
    "fluidity/internal/shared/logging"
)

// Server represents the tunnel server
type Server struct {
    config     *Config
    tlsConfig  *tls.Config
    listener   net.Listener
    httpClient *http.Client
    logger     *logging.Logger
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

// Config holds server configuration
type Config struct {
    ListenAddr  string `yaml:"listen_addr"`
    ListenPort  int    `yaml:"listen_port"`
    CertFile    string `yaml:"cert_file"`
    KeyFile     string `yaml:"key_file"`
    CACertFile  string `yaml:"ca_cert_file"`
    LogLevel    string `yaml:"log_level"`
    MaxConns    int    `yaml:"max_connections"`
}
```

#### 3.2.1 Tunnel Server Component
```go
// Package: internal/server/tunnel
package tunnel

import (
    "crypto/tls"
    "encoding/json"
    "net"
    "net/http"
    
    "fluidity/internal/shared/protocol"
)

// Server handles mTLS connections from agents
type Server struct {
    listener   net.Listener
    httpClient *http.Client
    logger     *logging.Logger
}

// NewServer creates a new tunnel server
func NewServer(tlsConfig *tls.Config, addr string) *Server {
    listener, err := tls.Listen("tcp", addr, tlsConfig)
    if err != nil {
        panic(fmt.Sprintf("Failed to create listener: %v", err))
    }
    
    // HTTP client for making requests to target websites
    httpClient := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }
    
    return &Server{
        listener:   listener,
        httpClient: httpClient,
        logger:     logging.NewLogger("tunnel-server"),
    }
}

// Start begins accepting connections
func (s *Server) Start() error {
    s.logger.Info("Tunnel server starting", "addr", s.listener.Addr())
    
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            s.logger.Error("Failed to accept connection", err)
            continue
        }
        
        // Handle each connection in a goroutine
        go s.handleConnection(conn.(*tls.Conn))
    }
}

// handleConnection processes requests from a single agent
func (s *Server) handleConnection(conn *tls.Conn) {
    defer conn.Close()
    
    // Verify client certificate
    state := conn.ConnectionState()
    if len(state.PeerCertificates) == 0 {
        s.logger.Warn("Client connected without certificate")
        return
    }
    
    clientCert := state.PeerCertificates[0]
    s.logger.Info("Client connected", "cn", clientCert.Subject.CommonName)
    
    decoder := json.NewDecoder(conn)
    encoder := json.NewEncoder(conn)
    
    for {
        var req protocol.Request
        if err := decoder.Decode(&req); err != nil {
            s.logger.Error("Failed to decode request", err)
            break
        }
        
        // Process request and send response
        go s.processRequest(&req, encoder)
    }
}

// processRequest handles a single HTTP request
func (s *Server) processRequest(req *protocol.Request, encoder *json.Encoder) {
    s.logger.Debug("Processing request", "id", req.ID, "url", req.URL)
    
    // Create HTTP request
    httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
    if err != nil {
        s.sendErrorResponse(req.ID, err, encoder)
        return
    }
    
    // Set headers
    for name, values := range req.Headers {
        for _, value := range values {
            httpReq.Header.Add(name, value)
        }
    }
    
    // Make request
    httpResp, err := s.httpClient.Do(httpReq)
    if err != nil {
        s.sendErrorResponse(req.ID, err, encoder)
        return
    }
    defer httpResp.Body.Close()
    
    // Read response body
    body, err := io.ReadAll(httpResp.Body)
    if err != nil {
        s.sendErrorResponse(req.ID, err, encoder)
        return
    }
    
    // Send response back through tunnel
    resp := &protocol.Response{
        ID:         req.ID,
        StatusCode: httpResp.StatusCode,
        Headers:    convertHeaders(httpResp.Header),
        Body:       body,
    }
    
    if err := encoder.Encode(resp); err != nil {
        s.logger.Error("Failed to send response", err)
    }
}
```

### 3.3 Shared Components

#### 3.3.1 Protocol Definition
```go
// Package: internal/shared/protocol
package protocol

// Request represents an HTTP request through the tunnel
type Request struct {
    ID      string              `json:"id"`
    Method  string              `json:"method"`
    URL     string              `json:"url"`
    Headers map[string][]string `json:"headers"`
    Body    []byte              `json:"body,omitempty"`
}

// Response represents an HTTP response through the tunnel
type Response struct {
    ID         string              `json:"id"`
    StatusCode int                 `json:"status_code"`
    Headers    map[string][]string `json:"headers"`
    Body       []byte              `json:"body,omitempty"`
    Error      string              `json:"error,omitempty"`
}
```

#### 3.3.2 TLS Configuration
```go
// Package: internal/shared/tls
package tls

import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
)

// LoadClientTLSConfig loads client-side mTLS configuration
func LoadClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
    // Load client certificate
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load client certificate: %w", err)
    }
    
    // Load CA certificate
    caCert, err := ioutil.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load CA certificate: %w", err)
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS13,
    }, nil
}

// LoadServerTLSConfig loads server-side mTLS configuration
func LoadServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
    // Load server certificate
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load server certificate: %w", err)
    }
    
    // Load CA certificate for client verification
    caCert, err := ioutil.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load CA certificate: %w", err)
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert,
        ClientCAs:    caCertPool,
        MinVersion:   tls.VersionTLS13,
    }, nil
}
```

#### 3.3.3 Configuration Management
```go
// Package: internal/shared/config
package config

import (
    "os"
    "github.com/spf13/viper"
)

// LoadConfig loads configuration with CLI override support
func LoadConfig[T any](configFile string, overrides map[string]interface{}) (*T, error) {
    viper.SetConfigFile(configFile)
    viper.SetConfigType("yaml")
    
    // Set defaults
    setDefaults()
    
    // Read config file
    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    // Apply CLI overrides
    for key, value := range overrides {
        if value != nil {
            viper.Set(key, value)
        }
    }
    
    // Environment variable support
    viper.AutomaticEnv()
    viper.SetEnvPrefix("FLUIDITY")
    
    var config T
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    return &config, nil
}

// SaveConfig saves updated configuration
func SaveConfig(configFile string, config interface{}) error {
    viper.Set("config", config)
    return viper.WriteConfigAs(configFile)
}
```

---

## 4. Implementation Strategy

### 4.1 Development Phases

#### Phase 1: Core Infrastructure (Weeks 1-4)
1. **Project Setup**
   - Initialize Go modules and project structure
   - Set up basic Makefile and build scripts
   - Create Docker configurations

2. **Basic Protocol Implementation**
   - Implement tunnel protocol (Request/Response structs)
   - Basic JSON serialization/deserialization
   - Simple TCP connection handling

3. **Certificate Management**
   - Certificate generation utilities
   - mTLS configuration loading
   - Basic certificate validation

4. **Minimal Agent Implementation**
   - Simple HTTP proxy server
   - Basic tunnel client connection
   - Request forwarding (without mTLS initially)

5. **Minimal Server Implementation**
   - Basic TCP server
   - HTTP client for target requests
   - Response forwarding

#### Phase 2: Security and mTLS (Weeks 5-6)
1. **mTLS Integration**
   - Client certificate authentication
   - Server certificate validation
   - TLS 1.3 enforcement

2. **Configuration Management**
   - YAML configuration files
   - CLI parameter handling
   - Environment variable support

3. **Error Handling and Logging**
   - Structured logging implementation
   - Error propagation
   - Debug logging for troubleshooting

#### Phase 3: Production Features (Weeks 7-8)
1. **Connection Management**
   - Connection pooling
   - Automatic reconnection
   - Graceful shutdown

2. **Configuration Updates**
   - Dynamic IP configuration
   - Persistent configuration updates
   - CLI override functionality

3. **Performance Optimization**
   - Concurrent request handling
   - Connection reuse
   - Request/response caching (if needed)

#### Phase 4: Containerization and Deployment (Weeks 9-10)
1. **Docker Implementation**
   - Multi-stage Docker builds
   - Container optimization
   - Certificate embedding

2. **Deployment Automation**
   - Build scripts
   - Deployment scripts
   - Cloud provider deployment guides

3. **Testing and Documentation**
   - Unit tests
   - Integration tests
   - User documentation

### 4.2 Key Implementation Considerations

#### Concurrency Model
- Use goroutines for handling multiple concurrent requests
- Channel-based communication for request/response matching
- Context-based cancellation for graceful shutdown

#### Error Handling
- Wrap errors with context using `fmt.Errorf`
- Use structured logging for error tracking
- Implement circuit breaker pattern for external requests

#### Security
- Validate all certificates against private CA
- Sanitize URLs and headers
- Implement rate limiting to prevent abuse

#### Performance
- Use connection pooling for HTTP clients
- Implement request batching if needed
- Monitor memory usage and implement limits

---

## 5. Deployment Architecture

### 5.1 Docker Configuration

#### Agent Dockerfile
```dockerfile
# Agent Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o fluidity-agent ./cmd/agent

FROM alpine/curl:latest
WORKDIR /root/

COPY --from=builder /app/fluidity-agent .
COPY configs/agent.yaml ./config/
COPY certs/ ./certs/

EXPOSE 8080
CMD ["./fluidity-agent", "--config", "./config/agent.yaml"]
```

#### Server Dockerfile
```dockerfile
# Server Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o fluidity-server ./cmd/server

FROM alpine/curl:latest
WORKDIR /root/

COPY --from=builder /app/fluidity-server .
COPY configs/server.yaml ./config/
COPY certs/ ./certs/

EXPOSE 8443
CMD ["./fluidity-server", "--config", "./config/server.yaml"]
```

### 5.2 Cloud Deployment Strategy

#### Container Orchestration Options
1. **Simple Docker Deployment**: Single container on cloud VM
2. **Docker Compose**: Multi-container setup with volumes
3. **Kubernetes**: For advanced scaling and management
4. **Cloud Container Services**: AWS ECS, Azure Container Instances, GCP Cloud Run

#### Example Cloud Deployment (AWS ECS)
```yaml
# task-definition.json
{
  "family": "fluidity-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "fluidity-server",
      "image": "your-registry/fluidity-server:latest",
      "portMappings": [
        {
          "containerPort": 8443,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "FLUIDITY_LOG_LEVEL",
          "value": "info"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/fluidity/server",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

---

## 6. Monitoring and Observability

### 6.1 Logging Strategy
- **Structured Logging**: JSON format for easy parsing
- **Log Levels**: ERROR, WARN, INFO, DEBUG
- **Contextual Information**: Request IDs, client certificates, timestamps
- **Privacy Protection**: No sensitive data in logs

### 6.2 Metrics (Future Enhancement)
- Connection count and duration
- Request/response latency
- Error rates and types
- Certificate expiration monitoring

### 6.3 Health Checks
- HTTP health endpoints for both agent and server
- Certificate validity checks
- Connection status monitoring

---

## 7. Security Considerations

### 7.1 mTLS Implementation
- **Certificate Validation**: Strict validation against private CA
- **TLS Version**: Enforce TLS 1.3 minimum
- **Cipher Suites**: Use only strong cipher suites
- **Key Management**: Secure storage of private keys

### 7.2 Network Security
- **Firewall Rules**: Restrict server access to necessary ports
- **Rate Limiting**: Prevent abuse and DoS attacks
- **Input Validation**: Sanitize all incoming data
- **Output Filtering**: Prevent data leakage in logs

### 7.3 Operational Security
- **Regular Updates**: Keep dependencies updated
- **Security Scanning**: Regular vulnerability scans
- **Certificate Rotation**: Plan for certificate renewal
- **Audit Logging**: Track security-relevant events

---

This architecture document provides a comprehensive blueprint for implementing the Fluidity HTTP tunnel solution in Go, based on the requirements specified in the PRD. The design emphasizes security, performance, and maintainability while keeping the implementation suitable for personal use.