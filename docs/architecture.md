# Architecture Design Document
# Fluidity - HTTP Tunnel Solution

**Status:** Phase 1 Complete - Core Infrastructure Implemented

---

## 1. Architecture Overview

### 1.1 System Architecture
Fluidity consists of two main Go applications communicating over mTLS, with an AWS Lambda control plane for automated lifecycle management:

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                              FLUIDITY ARCHITECTURE                                   │
├──────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│  ┌───────────────────────┐                         ┌─────────────────────────────┐   │
│  │   LOCAL NETWORK       │                         │   AWS CLOUD                 │   │
│  │                       │                         │                             │   │
│  │  ┌─────────────────┐  │                         │  ┌─────────────────────┐    │   │
│  │  │ Local Browser/  │  │                         │  │ Target Websites     │    │   │
│  │  │  Application    │  │                         │  │   (Internet)        │    │   │
│  │  └────────┬────────┘  │                         │  └──────────▲──────────┘    │   │
│  │           │           │                         │             │               │   │
│  │      HTTP | Proxy     │                         │             │ HTTP(S)       │   │
│  │           ▼           │                         │             │               │   │
│  │  ┌─────────────────┐  │       mTLS Tunnel       │  ┌──────────┴──────────┐    │   │
│  │  │ Tunnel Agent    │  │◄───────────────────────►│  │ Tunnel Server       │    │   │
│  │  │   (Go App)      │  │                         │  │ ECS Fargate Task    │    │   │
│  │  │  in Docker      │  │                         │  │   (Go in Docker)    │    │   │
│  │  │                 │  │                         │  │                     │    │   │
│  │  │ • Wake on start │  │                         │  │ • CloudWatch metrics│    │   │
│  │  │ • Kill on stop  │  │                         │  │   ActiveConns       │    │   │
│  │  └────────┬────────┘  │                         │  │   LastActivity      │    │   │
│  └───────────┼───────────┘                         │  └──────────▲──────────┘    │   │
│              │                                     │             |               │   │
│              │ HTTPS (API GW)                      │  ┌───────────────────────┐  │   │
│              └─────────────────────────────────────┼─►│ Lambda Control Plane  │  │   │
│                                                    │  │                       │  │   │
│                                                    │  │ ┌──────────────────┐  │  │   │
│                                                    │  │ │ API Gateway      │  │  │   │
│                                                    │  │ │ /wake  /kill     │  │  │   │
│                                                    │  │ └────────┬─────────┘  │  │   │
│                                                    │  │          │            │  │   │
│                                                    │  │ ┌────────▼─────────┐  │  │   │
│                                                    │  │ │ Wake Lambda      │  │  │   │
│                                                    │  │ │ Check & set =1   │  │  │   │
│                                                    │  │ └──────────────────┘  │  │   │
│                                                    │  │ ┌──────────────────┐  │  │   │
│                                                    │  │ │ Sleep Lambda     │  │  │   │
│                                                    │  │ │ Check & set =0   │  │  │   │
│                                                    │  │ └──────────────────┘  │  │   │
│                                                    │  │ ┌──────────────────┐  │  │   │
│                                                    │  │ │ Kill Lambda      │  │  │   │
│                                                    │  │ │ Force set =0     │  │  │   │
│                                                    │  │ └──────────────────┘  │  │   │
│                                                    │  │          ▲            │  │   │
│                                                    │  │          │            │  │   │
│                                                    │  │  ┌───────┴────────┐   │  │   │
│                                                    │  │  │ EventBridge    │   │  │   │
│                                                    │  │  │ Schedulers     │   │  │   │
│                                                    │  │  └────────────────┘   │  │   │
│                                                    │  └───────────────────────┘  │   │
│                                                    └─────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

#### Tunnel Agent (Local)
- **HTTP Proxy Server**: Acts as local HTTP proxy for browser/applications
- **mTLS Client**: Establishes secure connection to tunnel server
- **Traffic Forwarding**: Forwards HTTP requests through tunnel
- **Configuration Management**: Handles server IP configuration and updates
- **Lifecycle Integration**: Calls Wake Lambda on startup, Kill Lambda on shutdown
- **Connection Retry**: Attempts connection for configurable duration after wake call

#### Tunnel Server (Cloud - ECS Fargate)
- **mTLS Server**: Accepts authenticated connections from agents
- **HTTP Client**: Makes requests to target websites on behalf of agent
- **Response Relay**: Returns website responses through tunnel
- **Connection Management**: Handles multiple concurrent requests
- **Activity Metrics**: Emits CloudWatch metrics for active connections and last activity timestamp

#### Lambda Control Plane (AWS)
- **Wake Lambda**: 
  - Checks current ECS running count
  - Sets desired count to 1 if currently 0
  - Returns service status
  - Triggered by: Agent startup (via API Gateway)
  
- **Sleep Lambda**: 
  - Queries CloudWatch metrics for server activity
  - Sets desired count to 0 if idle beyond threshold
  - Implements cooldown to prevent thrashing
  - Triggered by: EventBridge schedule (every X minutes, configurable)
  
- **Kill Lambda**: 
  - Immediately sets desired count to 0 (no validation)
  - Forces shutdown regardless of activity
  - Triggered by: Agent shutdown (via API Gateway), EventBridge daily schedule

#### API Gateway
- **HTTP Endpoints**: Provides HTTPS endpoints for Lambda invocation
- **Authentication**: Secures endpoints (API key, IAM, or Cognito)
- **Rate Limiting**: Prevents abuse and excessive wake/kill calls

#### EventBridge Scheduler
- **Periodic Sleep Check**: Invokes Sleep Lambda every X minutes (configurable, default 5)
- **Daily Kill Schedule**: Invokes Kill Lambda at specific time daily (configurable, e.g., 11 PM)

---

## 2. Lambda Control Plane Architecture

### 2.1 Overview

The Lambda control plane manages the ECS Fargate server lifecycle, enabling cost-effective on-demand operation while maintaining responsiveness and preventing unnecessary runtime.

### 2.2 Wake Lambda

**Purpose**: Start the ECS service when an agent needs to connect.

**Trigger**: Agent startup → API Gateway `/wake` → Wake Lambda

**Logic**:
```python
def wake_handler(event, context):
    # 1. Check current service state
    service = ecs.describe_services(cluster=CLUSTER, services=[SERVICE])
    desired = service['desiredCount']
    running = service['runningCount']
    
    # 2. If already running or starting, return current state
    if desired > 0:
        return {
            "status": "already_running" if running > 0 else "starting",
            "desiredCount": desired,
            "runningCount": running
        }
    
    # 3. Set desired count to 1
    ecs.update_service(cluster=CLUSTER, service=SERVICE, desiredCount=1)
    
    # 4. Return status
    return {
        "status": "waking",
        "desiredCount": 1,
        "estimatedStartTime": "60-90 seconds"
    }
```

**IAM Permissions**:
- `ecs:DescribeServices` (read current state)
- `ecs:UpdateService` (set desired count)

**API Gateway Endpoint**: `POST /wake`

**Response**:
```json
{
  "status": "waking|already_running|starting",
  "desiredCount": 1,
  "runningCount": 0,
  "estimatedStartTime": "60-90 seconds"
}
```

### 2.3 Sleep Lambda

**Purpose**: Automatically scale down the ECS service when idle to save costs.

**Trigger**: EventBridge schedule (every X minutes, configurable)

**Logic**:
```python
def sleep_handler(event, context):
    # 1. Get CloudWatch metrics for last X minutes
    end = datetime.now(timezone.utc)
    start = end - timedelta(minutes=IDLE_THRESHOLD_MINUTES)
    
    metrics = cloudwatch.get_metric_data(
        MetricDataQueries=[
            {
                "Id": "active_connections",
                "MetricStat": {
                    "Metric": {
                        "Namespace": "Fluidity",
                        "MetricName": "ActiveConnections",
                        "Dimensions": [{"Name": "Service", "Value": "fluidity-server"}]
                    },
                    "Period": 60,
                    "Stat": "Average"
                }
            },
            {
                "Id": "last_activity",
                "MetricStat": {
                    "Metric": {
                        "Namespace": "Fluidity",
                        "MetricName": "LastActivityEpochSeconds",
                        "Dimensions": [{"Name": "Service", "Value": "fluidity-server"}]
                    },
                    "Period": 60,
                    "Stat": "Maximum"
                }
            }
        ],
        StartTime=start,
        EndTime=end
    )
    
    # 2. Check if idle
    avg_active = metrics['active_connections']['Average']
    last_activity = metrics['last_activity']['Maximum']
    idle_duration = time.time() - last_activity
    
    is_idle = (avg_active <= 0 and idle_duration >= IDLE_THRESHOLD_MINUTES * 60)
    
    # 3. Check current service state
    service = ecs.describe_services(cluster=CLUSTER, services=[SERVICE])
    desired = service['desiredCount']
    running = service['runningCount']
    
    # 4. If idle and running, scale down
    if is_idle and desired > 0:
        ecs.update_service(cluster=CLUSTER, service=SERVICE, desiredCount=0)
        return {
            "action": "scaled_down",
            "reason": "idle",
            "idleDuration": idle_duration
        }
    
    # 5. Otherwise, no action
    return {
        "action": "no_change",
        "desired": desired,
        "running": running,
        "avgActive": avg_active,
        "idleDuration": idle_duration
    }
```

**IAM Permissions**:
- `ecs:DescribeServices`
- `ecs:UpdateService`
- `cloudwatch:GetMetricData`

**EventBridge Schedule**: `rate(5 minutes)` (configurable)

**Configuration**:
- `IDLE_THRESHOLD_MINUTES`: Time to wait before scaling down (default: 15)
- `CLUSTER`: ECS cluster name
- `SERVICE`: ECS service name

### 2.4 Kill Lambda

**Purpose**: Immediately terminate the ECS service without validation.

**Trigger**: 
- Agent shutdown → API Gateway `/kill` → Kill Lambda
- EventBridge daily schedule (e.g., 11 PM)

**Logic**:
```python
def kill_handler(event, context):
    # 1. Set desired count to 0 immediately (no checks)
    ecs.update_service(cluster=CLUSTER, service=SERVICE, desiredCount=0)
    
    # 2. Return confirmation
    return {
        "status": "killed",
        "desiredCount": 0,
        "message": "Service shutdown initiated"
    }
```

**IAM Permissions**:
- `ecs:UpdateService`

**API Gateway Endpoint**: `POST /kill`

**EventBridge Schedule**: `cron(0 23 * * ? *)` (11 PM UTC daily, configurable)

**Response**:
```json
{
  "status": "killed",
  "desiredCount": 0,
  "message": "Service shutdown initiated"
}
```

### 2.5 CloudWatch Metrics Flow

**Server Metrics Emission** (Go code):

```go
// Package: internal/server/metrics (new)
package metrics

import (
    "context"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type MetricsEmitter struct {
    client    *cloudwatch.Client
    namespace string
    service   string
}

func NewMetricsEmitter(cfg aws.Config) *MetricsEmitter {
    return &MetricsEmitter{
        client:    cloudwatch.NewFromConfig(cfg),
        namespace: "Fluidity",
        service:   "fluidity-server",
    }
}

func (m *MetricsEmitter) EmitActivity(ctx context.Context, activeConnections int) error {
    _, err := m.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
        Namespace: aws.String(m.namespace),
        MetricData: []types.MetricDatum{
            {
                MetricName: aws.String("ActiveConnections"),
                Timestamp:  aws.Time(time.Now()),
                Unit:       types.StandardUnitCount,
                Value:      aws.Float64(float64(activeConnections)),
                Dimensions: []types.Dimension{
                    {Name: aws.String("Service"), Value: aws.String(m.service)},
                },
            },
            {
                MetricName: aws.String("LastActivityEpochSeconds"),
                Timestamp:  aws.Time(time.Now()),
                Unit:       types.StandardUnitSeconds,
                Value:      aws.Float64(float64(time.Now().Unix())),
                Dimensions: []types.Dimension{
                    {Name: aws.String("Service"), Value: aws.String(m.service)},
                },
            },
        },
    })
    return err
}
```

**Integration Points**:
- Server emits metrics on every request/connection event
- Sleep Lambda queries these metrics every X minutes
- Metrics retained for 15 months (CloudWatch standard)

### 2.6 API Gateway Configuration

**Endpoints**:
- `POST /wake` → Wake Lambda
- `POST /kill` → Kill Lambda

**Authentication Options**:
1. **API Key** (simplest): Fixed key in Agent config
2. **IAM SigV4**: Agent uses AWS credentials
3. **Cognito/JWT**: For multi-user scenarios

**Throttling**:
- Per-endpoint rate limits (e.g., 10 requests/minute)
- Burst limits to prevent abuse

**CORS**: Not required (Agent is not a browser)

### 2.7 EventBridge Scheduler Configuration

**Sleep Check Schedule**:
```json
{
  "ScheduleExpression": "rate(5 minutes)",
  "Target": {
    "Arn": "arn:aws:lambda:region:account:function:fluidity-sleep",
    "Input": "{}"
  }
}
```

**Daily Kill Schedule**:
```json
{
  "ScheduleExpression": "cron(0 23 * * ? *)",
  "Target": {
    "Arn": "arn:aws:lambda:region:account:function:fluidity-kill",
    "Input": "{\"reason\": \"daily_scheduled_shutdown\"}"
  }
}
```

---

## 3. Go Application Structure

### 3.1 Project Layout
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
│   │   ├── lifecycle/   # NEW: Wake/Kill API integration
│   │   └── cli/         # CLI handling
│   ├── server/          # Server-specific logic
│   │   ├── tunnel/      # Tunnel server
│   │   ├── proxy/       # HTTP client for target requests
│   │   ├── metrics/     # NEW: CloudWatch metrics emission
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
│   ├── lambda/          # NEW: Lambda function code
│   │   ├── wake/
│   │   │   └── main.py
│   │   ├── sleep/
│   │   │   └── main.py
│   │   └── kill/
│   │       └── main.py
│   ├── cloudformation/  # CloudFormation templates
│   │   ├── fargate.yaml # ECS Fargate infrastructure
│   │   └── lambda.yaml  # NEW: Lambda control plane infrastructure
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

## 4. Detailed Component Design

### 4.1 Tunnel Agent Architecture

```go
// Package: internal/agent
package agent

import (
    "context"
    "crypto/tls"
    "net/http"
    "sync"
    "time"
    
    "fluidity/internal/shared/protocol"
    "fluidity/internal/shared/logging"
    "fluidity/internal/agent/lifecycle"
)

// Agent represents the tunnel agent
type Agent struct {
    config         *Config
    tlsConfig      *tls.Config
    proxyServer    *http.Server
    tunnelConn     *TunnelConnection
    lifecycleClient *lifecycle.Client
    logger         *logging.Logger
    ctx            context.Context
    cancel         context.CancelFunc
    wg             sync.WaitGroup
}

// Config holds agent configuration
type Config struct {
    ServerIP            string        `yaml:"server_ip"`
    ServerPort          int           `yaml:"server_port"`
    LocalProxyPort      int           `yaml:"local_proxy_port"`
    CertFile            string        `yaml:"cert_file"`
    KeyFile             string        `yaml:"key_file"`
    CACertFile          string        `yaml:"ca_cert_file"`
    LogLevel            string        `yaml:"log_level"`
    WakeAPIEndpoint     string        `yaml:"wake_api_endpoint"`      // NEW: API Gateway wake endpoint
    KillAPIEndpoint     string        `yaml:"kill_api_endpoint"`      // NEW: API Gateway kill endpoint
    APIKey              string        `yaml:"api_key"`                // NEW: API Gateway auth
    ConnectionTimeout   time.Duration `yaml:"connection_timeout"`      // NEW: Max time to wait after wake
    ConnectionRetryInterval time.Duration `yaml:"connection_retry_interval"` // NEW: Retry interval
}

// Start initiates the agent with lifecycle integration
func (a *Agent) Start() error {
    a.logger.Info("Starting Fluidity Agent with lifecycle management")
    
    // 1. Call Wake Lambda via API Gateway
    a.logger.Info("Calling Wake API to start server")
    wakeResp, err := a.lifecycleClient.Wake(a.ctx)
    if err != nil {
        return fmt.Errorf("failed to wake server: %w", err)
    }
    a.logger.Info("Wake API response", "status", wakeResp.Status, "desiredCount", wakeResp.DesiredCount)
    
    // 2. Wait and retry connection for configured duration
    a.logger.Info("Attempting tunnel connection", "timeout", a.config.ConnectionTimeout)
    connected := false
    deadline := time.Now().Add(a.config.ConnectionTimeout)
    
    for time.Now().Before(deadline) && !connected {
        err = a.tunnelConn.Connect()
        if err == nil {
            connected = true
            a.logger.Info("Tunnel connection established")
            break
        }
        
        a.logger.Debug("Connection attempt failed, retrying", "error", err, "nextRetry", a.config.ConnectionRetryInterval)
        time.Sleep(a.config.ConnectionRetryInterval)
    }
    
    if !connected {
        return fmt.Errorf("failed to connect to server within timeout %v", a.config.ConnectionTimeout)
    }
    
    // 3. Start proxy server
    a.logger.Info("Starting proxy server", "port", a.config.LocalProxyPort)
    a.wg.Add(1)
    go func() {
        defer a.wg.Done()
        if err := a.proxyServer.ListenAndServe(); err != http.ErrServerClosed {
            a.logger.Error("Proxy server error", err)
        }
    }()
    
    return nil
}

// Shutdown gracefully stops the agent and calls Kill Lambda
func (a *Agent) Shutdown() error {
    a.logger.Info("Shutting down agent")
    a.cancel()
    
    // 1. Stop proxy server
    if err := a.proxyServer.Shutdown(context.Background()); err != nil {
        a.logger.Error("Error shutting down proxy server", err)
    }
    
    // 2. Close tunnel connection
    if err := a.tunnelConn.Disconnect(); err != nil {
        a.logger.Error("Error closing tunnel connection", err)
    }
    
    // 3. Call Kill Lambda via API Gateway
    a.logger.Info("Calling Kill API to stop server")
    killResp, err := a.lifecycleClient.Kill(a.ctx)
    if err != nil {
        a.logger.Warn("Failed to call Kill API", "error", err)
        // Don't fail shutdown if Kill fails
    } else {
        a.logger.Info("Kill API response", "status", killResp.Status)
    }
    
    // 4. Wait for goroutines
    a.wg.Wait()
    a.logger.Info("Agent shutdown complete")
    
    return nil
}

// TunnelConnection manages the mTLS connection to server
type TunnelConnection struct {
    conn      *tls.Conn
    mu        sync.RWMutex
    connected bool
    requests  map[string]chan *protocol.Response
}
```

#### 4.1.1 Lifecycle Client Component (NEW)
```go
// Package: internal/agent/lifecycle
package lifecycle

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// Client handles Wake/Kill API calls
type Client struct {
    wakeEndpoint string
    killEndpoint string
    apiKey       string
    httpClient   *http.Client
}

// NewClient creates a new lifecycle API client
func NewClient(wakeEndpoint, killEndpoint, apiKey string) *Client {
    return &Client{
        wakeEndpoint: wakeEndpoint,
        killEndpoint: killEndpoint,
        apiKey:       apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// WakeResponse represents the Wake Lambda response
type WakeResponse struct {
    Status             string `json:"status"`
    DesiredCount       int    `json:"desiredCount"`
    RunningCount       int    `json:"runningCount"`
    EstimatedStartTime string `json:"estimatedStartTime,omitempty"`
}

// Wake calls the Wake Lambda via API Gateway
func (c *Client) Wake(ctx context.Context) (*WakeResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "POST", c.wakeEndpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create wake request: %w", err)
    }
    
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call wake API: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("wake API returned status %d", resp.StatusCode)
    }
    
    var wakeResp WakeResponse
    if err := json.NewDecoder(resp.Body).Decode(&wakeResp); err != nil {
        return nil, fmt.Errorf("failed to decode wake response: %w", err)
    }
    
    return &wakeResp, nil
}

// KillResponse represents the Kill Lambda response
type KillResponse struct {
    Status       string `json:"status"`
    DesiredCount int    `json:"desiredCount"`
    Message      string `json:"message"`
}

// Kill calls the Kill Lambda via API Gateway
func (c *Client) Kill(ctx context.Context) (*KillResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "POST", c.killEndpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create kill request: %w", err)
    }
    
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call kill API: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("kill API returned status %d", resp.StatusCode)
    }
    
    var killResp KillResponse
    if err := json.NewDecoder(resp.Body).Decode(&killResp); err != nil {
        return nil, fmt.Errorf("failed to decode kill response: %w", err)
    }
    
    return &killResp, nil
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

### 4.2 Tunnel Server Architecture

```go
// Package: internal/server
package server

import (
    "context"
    "crypto/tls"
    "net"
    "sync"
    "sync/atomic"
    "time"
    
    "fluidity/internal/shared/protocol"
    "fluidity/internal/shared/logging"
    "fluidity/internal/server/metrics"
    
    "github.com/aws/aws-sdk-go-v2/config"
)

// Server represents the tunnel server
type Server struct {
    config          *Config
    tlsConfig       *tls.Config
    listener        net.Listener
    httpClient      *http.Client
    metricsEmitter  *metrics.Emitter      // NEW: CloudWatch metrics
    activeConns     atomic.Int32          // NEW: Track active connections
    lastActivity    atomic.Int64          // NEW: Track last activity timestamp
    logger          *logging.Logger
    ctx             context.Context
    cancel          context.CancelFunc
    wg              sync.WaitGroup
}

// Config holds server configuration
type Config struct {
    ListenAddr       string `yaml:"listen_addr"`
    ListenPort       int    `yaml:"listen_port"`
    CertFile         string `yaml:"cert_file"`
    KeyFile          string `yaml:"key_file"`
    CACertFile       string `yaml:"ca_cert_file"`
    LogLevel         string `yaml:"log_level"`
    MaxConns         int    `yaml:"max_connections"`
    EmitMetrics      bool   `yaml:"emit_metrics"`          // NEW: Enable CloudWatch metrics
    MetricsInterval  time.Duration `yaml:"metrics_interval"` // NEW: How often to emit
}

// NewServer creates a new tunnel server with metrics support
func NewServer(cfg *Config) (*Server, error) {
    s := &Server{
        config: cfg,
        logger: logging.NewLogger("server"),
    }
    
    // Load AWS config and create metrics emitter if enabled
    if cfg.EmitMetrics {
        awsCfg, err := config.LoadDefaultConfig(context.Background())
        if err != nil {
            return nil, fmt.Errorf("failed to load AWS config: %w", err)
        }
        s.metricsEmitter = metrics.NewEmitter(awsCfg)
        s.logger.Info("CloudWatch metrics enabled", "interval", cfg.MetricsInterval)
    }
    
    return s, nil
}

// Start begins the server and metrics emission
func (s *Server) Start() error {
    // ... existing listener setup ...
    
    // Start metrics emission goroutine
    if s.config.EmitMetrics && s.metricsEmitter != nil {
        s.wg.Add(1)
        go s.emitMetricsPeriodically()
    }
    
    // Accept connections
    // ... existing code ...
}

// emitMetricsPeriodically sends CloudWatch metrics on a timer
func (s *Server) emitMetricsPeriodically() {
    defer s.wg.Done()
    ticker := time.NewTicker(s.config.MetricsInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            activeConns := int(s.activeConns.Load())
            if err := s.metricsEmitter.EmitActivity(s.ctx, activeConns); err != nil {
                s.logger.Warn("Failed to emit metrics", "error", err)
            } else {
                s.logger.Debug("Emitted metrics", "activeConnections", activeConns)
            }
            
        case <-s.ctx.Done():
            return
        }
    }
}

// handleConnection tracks connection lifecycle and updates metrics
func (s *Server) handleConnection(conn *tls.Conn) {
    defer conn.Close()
    
    // Increment active connections
    s.activeConns.Add(1)
    s.updateLastActivity()
    defer s.activeConns.Add(-1)
    
    // ... existing connection handling ...
}

// processRequest updates activity timestamp on each request
func (s *Server) processRequest(req *protocol.Request, encoder *json.Encoder) {
    s.updateLastActivity()
    // ... existing request processing ...
}

// updateLastActivity records current time as last activity
func (s *Server) updateLastActivity() {
    s.lastActivity.Store(time.Now().Unix())
}
```

#### 4.2.1 Server Metrics Emitter Component (NEW)
```go
// Package: internal/server/metrics
package metrics

import (
    "context"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Emitter sends CloudWatch metrics for server activity
type Emitter struct {
    client    *cloudwatch.Client
    namespace string
    service   string
}

// NewEmitter creates a new CloudWatch metrics emitter
func NewEmitter(cfg aws.Config) *Emitter {
    return &Emitter{
        client:    cloudwatch.NewFromConfig(cfg),
        namespace: "Fluidity",
        service:   "fluidity-server",
    }
}

// EmitActivity sends active connections and last activity timestamp
func (e *Emitter) EmitActivity(ctx context.Context, activeConnections int) error {
    now := time.Now()
    
    _, err := e.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
        Namespace: aws.String(e.namespace),
        MetricData: []types.MetricDatum{
            {
                MetricName: aws.String("ActiveConnections"),
                Timestamp:  aws.Time(now),
                Unit:       types.StandardUnitCount,
                Value:      aws.Float64(float64(activeConnections)),
                Dimensions: []types.Dimension{
                    {Name: aws.String("Service"), Value: aws.String(e.service)},
                },
            },
            {
                MetricName: aws.String("LastActivityEpochSeconds"),
                Timestamp:  aws.Time(now),
                Unit:       types.StandardUnitSeconds,
                Value:      aws.Float64(float64(now.Unix())),
                Dimensions: []types.Dimension{
                    {Name: aws.String("Service"), Value: aws.String(e.service)},
                },
            },
        },
    })
    
    return err
}
```

#### 4.2.2 Tunnel Server Component
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

// Envelope wraps all message types with type discrimination
type Envelope struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

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

// ConnectRequest initiates HTTPS CONNECT tunneling
type ConnectRequest struct {
    ID   string `json:"id"`
    Host string `json:"host"`
}

// ConnectAck acknowledges CONNECT establishment
type ConnectAck struct {
    ID      string `json:"id"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

// ConnectData carries bidirectional CONNECT stream data
type ConnectData struct {
    ID   string `json:"id"`
    Data []byte `json:"data"`
}

// WebSocketOpen requests WebSocket connection establishment
type WebSocketOpen struct {
    ID      string              `json:"id"`
    URL     string              `json:"url"`
    Headers map[string][]string `json:"headers"`
}

// WebSocketAck acknowledges WebSocket connection
type WebSocketAck struct {
    ID      string `json:"id"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

// WebSocketMessage carries WebSocket frame data
type WebSocketMessage struct {
    ID   string `json:"id"`
    Data []byte `json:"data"`
}

// WebSocketClose signals WebSocket connection closure
type WebSocketClose struct {
    ID    string `json:"id"`
    Code  int    `json:"code"`
    Error string `json:"error,omitempty"`
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

## 5. Implementation Strategy

### 5.1 Development Phases

#### Phase 1: Core Infrastructure ✅ COMPLETE
1. **Project Setup** ✅
   - Initialized Go modules and project structure
   - Set up platform-specific Makefiles (Windows/macOS/Linux)
   - Created Docker configurations with multi-stage builds

2. **Basic Protocol Implementation** ✅
   - Implemented tunnel protocol (Request/Response/CONNECT/WebSocket structs)
   - JSON serialization/deserialization with envelope pattern
   - Secure TLS connection handling

3. **Certificate Management** ✅
   - Certificate generation utilities (PowerShell and Bash scripts)
   - mTLS configuration loading
   - Certificate validation against private CA

4. **Agent Implementation** ✅
   - HTTP/HTTPS proxy server on port 8080
   - Tunnel client connection with mTLS
   - Request forwarding with CONNECT method support
   - WebSocket tunneling support

5. **Server Implementation** ✅
   - mTLS TCP server on port 8443
   - HTTP client for target requests
   - Response forwarding through tunnel
   - WebSocket connection handling

#### Phase 2: Security and mTLS ✅ COMPLETE
1. **mTLS Integration** ✅
   - Client certificate authentication
   - Server certificate validation
   - TLS 1.3 enforcement

2. **Configuration Management** ✅
   - YAML configuration files (server.yaml, agent.yaml)
   - CLI parameter handling with Cobra
   - Environment variable support
   - Server IP configuration with CLI override

3. **Error Handling and Logging** ✅
   - Structured logging implementation with logrus
   - Error propagation with context
   - Debug logging for troubleshooting
   - Privacy-focused minimal logging

#### Phase 3: Production Features ✅ MOSTLY COMPLETE
1. **Connection Management** ✅
   - Automatic reconnection with exponential backoff
   - Graceful shutdown with context cancellation
   - Connection state management

2. **Configuration Updates** ✅
   - Dynamic IP configuration
   - Persistent configuration updates
   - CLI override functionality

3. **Performance Optimization** 🚧
   - Concurrent request handling with goroutines
   - Channel-based request/response matching
   - Connection pooling (needs optimization)

#### Phase 4: Containerization and Deployment ✅ MOSTLY COMPLETE
1. **Docker Implementation** ✅
   - Multi-stage Docker builds
   - Alpine-based containers (~43MB)
   - Certificate volume mounting

2. **Deployment Automation** 🚧
   - Build scripts for all platforms
   - Automated testing scripts (test-docker.ps1/.sh, test-local.ps1/.sh)
   - Cloud provider deployment guides (pending)

3. **Testing and Documentation** ✅
   - Integration tests (HTTP, HTTPS, WebSocket)
   - Automated end-to-end testing
   - Comprehensive user documentation

#### Phase 5: Lambda Control Plane 🚧 NOT STARTED
1. **Lambda Functions** 🚧
   - Implement Wake Lambda (Python)
   - Implement Sleep Lambda with CloudWatch metrics query (Python)
   - Implement Kill Lambda (Python)
   - Create IAM roles with least-privilege policies

2. **API Gateway Integration** 🚧
   - Create REST API with `/wake` and `/kill` endpoints
   - Configure API key authentication
   - Set up throttling and rate limits
   - CORS configuration (if needed)

3. **EventBridge Schedulers** 🚧
   - Create periodic Sleep Lambda trigger (every X minutes)
   - Create daily Kill Lambda trigger (configurable time)
   - Configure retry policies

4. **Agent Lifecycle Integration** 🚧
   - Add lifecycle client package (`internal/agent/lifecycle`)
   - Implement wake API call on agent startup
   - Implement connection retry with configurable timeout
   - Implement kill API call on agent shutdown
   - Add configuration for API endpoints and credentials

5. **Server Metrics Integration** 🚧
   - Add metrics emitter package (`internal/server/metrics`)
   - Implement CloudWatch PutMetricData calls
   - Track active connections with atomic counters
   - Track last activity timestamp
   - Emit metrics on configurable interval

6. **CloudFormation Templates** 🚧
   - Create `deployments/cloudformation/lambda.yaml` for Lambda infrastructure
   - Update existing `fargate.yaml` with metrics IAM permissions
   - Add outputs for API Gateway endpoints
   - Document deployment process

7. **Testing** 🚧
   - Unit tests for lifecycle client
   - Unit tests for metrics emitter
   - Integration tests for Lambda functions
   - End-to-end test with full lifecycle (wake → connect → kill)

### 5.2 Key Implementation Considerations

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

## 6. Deployment Architecture

### 6.1 Docker Configuration

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

## 7. Monitoring and Observability

### 7.1 Logging Strategy
- **Structured Logging**: JSON format for easy parsing
- **Log Levels**: ERROR, WARN, INFO, DEBUG
- **Contextual Information**: Request IDs, client certificates, timestamps
- **Privacy Protection**: No sensitive data in logs

### 7.2 Metrics (Enhanced with CloudWatch)
- **Connection count and duration** (tracked in server)
- **Request/response latency** (future enhancement)
- **Error rates and types**
- **Certificate expiration monitoring**
- **CloudWatch Custom Metrics** (NEW):
  - `ActiveConnections`: Gauge of current connections
  - `LastActivityEpochSeconds`: Timestamp of last activity
  - Used by Sleep Lambda for idle detection

### 7.3 Health Checks
- HTTP health endpoints for both agent and server
- Certificate validity checks
- Connection status monitoring

---

## 8. Security Considerations

### 8.1 mTLS Implementation
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