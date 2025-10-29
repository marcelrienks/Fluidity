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
- [ ] Update `fargate.yaml` with CloudWatch permissions
- [ ] Parameter files (dev/prod environments)
- [ ] Stack policies to prevent deletion
- [ ] Drift detection

### Monitoring & Observability
- [ ] CloudWatch dashboard (Lambda, API Gateway, ECS, Fluidity metrics)
- [ ] CloudWatch alarms (Lambda errors, API Gateway 5xx, ECS stuck, no metrics)
- [ ] SNS topic for alarm notifications
- [ ] AWS X-Ray tracing for Lambda functions
- [ ] Lambda Insights for enhanced metrics
- [ ] Cost anomaly detection

### Testing & Validation
- [ ] Unit tests: lifecycle client, metrics emitter
- [ ] Integration tests: Wake/Sleep/Kill → ECS updates
- [ ] E2E test: agent startup → wake → connect → shutdown → kill
- [ ] E2E test: idle detection → automatic sleep
- [ ] Load test: concurrent wake/kill calls
- [ ] Test failure scenarios (timeouts, API errors, network failures)

### Documentation
- [x] Update architecture.md with Lambda design
- [x] Update deployment.md with Lambda guide
- [x] Update plan.md with tasks
- [ ] Update fargate.md with Lambda integration
- [ ] Update README.md with Lambda overview
- [ ] Operational runbook (manual wake/sleep/kill, troubleshooting)
- [ ] Cost optimization guide
- [ ] Edge case documentation (race conditions, stuck ECS, quotas)
- [ ] Architecture Decision Records (ADRs)

---

## 📋 Phase 3: Production Readiness (PLANNED)

### AWS Deployment
- [ ] Deploy to AWS ECS Fargate (CloudFormation)
- [ ] Configure VPC, subnets, security groups
- [ ] Set up ECR for Docker images
- [ ] Configure CloudWatch Logs

### CI/CD Pipeline
- [ ] GitHub Actions or AWS CodePipeline
- [ ] Automated builds on commit
- [ ] Automated testing (unit, integration, E2E)
- [ ] Automated deployment to dev/staging/prod

### Security Hardening
- [ ] CloudWatch Logs encryption (KMS)
- [ ] Secrets Manager for API keys and certificates
- [ ] WAF integration for API Gateway
- [ ] Certificate renewal and rotation automation
- [ ] Security audit and vulnerability assessment

### Performance Optimization
- [ ] Connection pooling optimization
- [ ] Load testing and benchmarking
- [ ] Identify and fix bottlenecks

### Monitoring & Health
- [ ] Health check endpoints (agent, server)
- [ ] Metrics collection and dashboards
- [ ] Distributed tracing
- [ ] Log aggregation and analysis

---

## Related Documentation

- **[Architecture](architecture.md)** - System design and components
- **[Product Requirements](product.md)** - PRD and functional requirements  
- **[Deployment Guide](deployment.md)** - All deployment options
- **[Testing Guide](testing.md)** - Testing strategy and execution
