# Fluidity Project Plan

This document outlines the development roadmap, organized by completion status and priority.

---

## ✅ Phase 1: Core Infrastructure (COMPLETE)

### Tunneling & Protocol Support
- [x] HTTP/HTTPS tunneling with CONNECT method
- [x] WebSocket bidirectional tunneling
- [x] Concurrent request handling with goroutines

### Security
- [x] Mutual TLS (mTLS) authentication
- [x] Private Certificate Authority (CA)
- [x] Secure TLS verification (no InsecureSkipVerify)

### Reliability
- [x] Circuit breaker pattern for external requests
- [x] Exponential backoff retry logic
- [x] Connection recovery and reconnection
- [x] Graceful shutdown with context cancellation

### Configuration
- [x] YAML configuration files
- [x] CLI parameter handling
- [x] Environment variable support
- [x] Server IP configuration with override

### Containerization
- [x] Docker images for agent and server (~44MB Alpine)
- [x] Multi-platform builds (Windows/macOS/Linux)
- [x] Certificate volume mounting

### Testing
- [x] Unit tests (circuit breaker, retry, protocol, config) - 49 tests
- [x] Integration tests (tunnel, proxy, WebSocket) - 26 tests
- [x] E2E automated test scripts (local and Docker)
- [x] Test utilities and helpers
- [x] Testing documentation

### Documentation
- [x] README with quick start
- [x] Architecture design document
- [x] Product requirements document (PRD)
- [x] Testing guide
- [x] Deployment guide (local, Docker, Fargate)

---

## 🚧 Phase 2: AWS Lambda Control Plane (IN PROGRESS)

### Lambda Functions
- [x] Wake Lambda (Go) - check state, set DesiredCount=1
- [x] Sleep Lambda (Go) - query CloudWatch, scale down if idle
- [x] Kill Lambda (Go) - immediate shutdown
- [x] IAM roles with least-privilege policies
- [x] Error handling and structured logging
- [x] Lambda versioning and aliases
- [x] Timeout configuration (30s Wake/Kill, 60s Sleep)
- [x] Reserved concurrency limits

### API Gateway
- [x] REST API with `/wake` and `/kill` endpoints
- [x] API key authentication
- [x] Throttling and rate limits
- [x] CloudWatch execution logging
- [x] Access logging
- [x] Request/response validation
- [x] Usage plans and quotas

### EventBridge Schedulers
- [x] Sleep Lambda schedule (configurable rate, default 5 min)
- [x] Kill Lambda schedule (configurable cron, default 11 PM UTC)
- [x] Retry policies for failed invocations

### Agent Integration
- [x] Create `internal/core/agent/lifecycle/` package
- [x] Wake API client with exponential backoff retry
- [x] Kill API client
- [x] Configuration (endpoints, API key from env var)
- [x] Connection retry logic (90s timeout, 5s interval)
- [x] Startup: wake → retry connection
- [x] Shutdown: kill gracefully
- [x] Circuit breaker for API calls
- [x] Fallback: work without Lambda control plane

### Server Integration
- [x] Create `internal/core/server/metrics/` package (config.go, metrics.go, metrics_test.go)
- [x] CloudWatch PutMetricData client with AWS SDK v2
- [x] Track active connections (atomic.Int64 counter)
- [x] Track last activity timestamp (atomic.Int64 Unix epoch)
- [x] Emit `ActiveConnections` metric (60s configurable interval)
- [x] Emit `LastActivityEpochSeconds` metric
- [x] Metric batching (automatic via AWS SDK)
- [x] Error handling (graceful degradation without CloudWatch)
- [x] Integrated into server lifecycle (Start/Stop, connections, activity)
- [x] Comprehensive unit tests (6 tests, all passing)

### Infrastructure as Code
- [x] `deployments/cloudformation/lambda.yaml`
  - Lambda functions, IAM roles, API Gateway, EventBridge
- [x] Update `fargate.yaml` with CloudWatch permissions (PutMetricData for metrics emission)
- [x] Parameter files (dev/prod environments) - `params-dev.json`, `params-prod.json`
- [x] Stack policies to prevent deletion - `stack-policy.json`
- [x] Deployment and management scripts - `deploy-fluidity.ps1`, `deploy-fluidity.sh`
- [x] Drift detection (integrated in status action)
- [x] Comprehensive IaC documentation - `INFRASTRUCTURE_AS_CODE.md`

### Monitoring & Observability
- [x] CloudWatch alarms (Lambda errors, ECS stuck, no metrics)
- [x] SNS email notifications for alarms
- [x] CloudWatch dashboard

### Testing & Validation
- [x] Unit tests: lifecycle client, metrics emitter
- [x] Integration tests: Wake/Sleep/Kill → ECS updates
- [x] E2E test: agent startup → wake → connect → shutdown → kill
- [x] E2E test: idle detection → automatic sleep
- [x] Load test: concurrent wake/kill calls
- [x] Test failure scenarios (timeouts, API errors, network failures)

### Documentation
- [x] Update architecture.md with Lambda design
- [x] Update deployment.md with Lambda guide
- [x] Update plan.md with tasks
- [x] Update fargate.md with Lambda integration
- [x] Update README.md with Lambda overview
- [x] Operational runbook (manual wake/sleep/kill, troubleshooting)

---

## 🚀 Phase 3: Production Readiness (PERSONAL USE OPTIMIZED)

### Prerequisites ✅ COMPLETED
- [x] Store certificates in AWS Secrets Manager
  - Created `internal/shared/secretsmanager/secrets.go` package
  - Supports `SaveCertificatesToSecrets()` utility function
  - Scripts: `scripts/save-certs-to-secrets.sh` and `.ps1`
- [x] Update server code to fetch certs from Secrets Manager on startup
  - Added `use_secrets_manager` and `secrets_manager_name` config options
  - Server tries Secrets Manager first, falls back to local files
  - 30-second timeout with graceful degradation
- [x] Update agent code to fetch certs from Secrets Manager on startup
  - Added `use_secrets_manager` and `secrets_manager_name` config options
  - Agent tries Secrets Manager first, falls back to local files
  - Works with lifecycle management
- [x] Add `/health` endpoint to server
  - HTTP endpoint on port 8080: `GET /health`
  - Returns active connections, uptime, max connections, utilization %
  - Integrated with server startup/shutdown
- [x] Add `/health` endpoint to agent
  - HTTP endpoint on proxy port: `GET /health`
  - Returns connection status, uptime, proxy port, server address
  - Separate from tunnel traffic

### AWS Deployment
- [ ] Deploy Fargate stack to AWS using CloudFormation
  - Requires: Server code updated to use Secrets Manager
  - Reference: `docs/infrastructure.md` and `docs/fargate.md`
- [ ] Deploy Lambda stack to AWS using CloudFormation
  - Reference: `docs/lambda.md` and `docs/infrastructure.md`
- [ ] End-to-end testing (wake → connect → metrics → kill)
  - Verify server starts and fetches certs from Secrets Manager
  - Verify agent connects with health check
  - Check metrics flow to CloudWatch
- [ ] Configure SNS email alerts
  - Subscribe email to alarm topic
  - Test alarm notifications

---

## Related Documentation

- **[Infrastructure as Code](infrastructure.md)** - CloudFormation deployment details
- **[Fargate Deployment](fargate.md)** - Cloud setup and Lambda integration
- **[Lambda Functions](lambda.md)** - Control plane functions and configuration
- **[Operational Runbook](operational-runbook.md)** - Day-to-day operations and troubleshooting

---

## Related Documentation

- **[Architecture](architecture.md)** - System design and components
- **[Product Requirements](product.md)** - PRD and functional requirements  
- **[Deployment Guide](deployment.md)** - All deployment options
- **[Testing Guide](testing.md)** - Testing strategy and execution
