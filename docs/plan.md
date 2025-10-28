# Fluidity Project Plan

This document outlines all outstanding work required for Phase 1, organized by phases and actionable steps. It replaces `PHASE1.md`.

---

## Current Status Summary

### ✅ Completed Core Features
- HTTP/HTTPS tunneling with CONNECT method support
- Mutual TLS (mTLS) authentication between agent and server
- Private Certificate Authority infrastructure
- Basic configuration management (CLI, config files, environment variables)
- Concurrent request handling with goroutines
- Structured logging with privacy protections
- Connection recovery and reconnection logic
- Docker containerization for both agent and server
- WebSocket tunneling support (bidirectional communication)
- Automated testing scripts (Docker and local binary tests)

### 🚧 In Progress / Outstanding
- **Deployment**: Cloud deployment and production hardening
- **Monitoring**: Health checks and observability endpoints
- **Testing**: Expanded integration test coverage
- **Performance**: Connection pooling optimization

---

## Phase 1: Core Infrastructure (Outstanding Work)

### Protocol Support ✅ COMPLETE
- [x] Implement HTTPS/CONNECT tunneling (Oct 2025 - implemented in tunnel client/server)
- [x] Implement WebSocket support (Oct 2025 - full bidirectional WebSocket tunneling implemented)

### Security ✅ COMPLETE
- [x] mTLS certificate generation and integration (Oct 2025 - certificates exist in certs/)
- [x] Secure TLS verification (Oct 2025 - no InsecureSkipVerify found in codebase)

### Error Handling & Reliability ✅ COMPLETE
- [x] Improve error handling and recovery throughout agent and server (Oct 2025 - enhanced error messages, cleanup, timeouts)
- [x] Implement circuit breaker pattern for external requests (Oct 2025 - integrated in tunnel server)
- [x] Enhance retry logic for connection attempts (Oct 2025 - reconnection logic implemented)
- [x] Enhance retry logic for request forwarding (Oct 2025 - exponential backoff retry with configurable attempts)

### Testing ✅ COMPLETE
- [x] Unit tests for circuit breaker (Oct 2025 - 7 tests, 100% coverage)
- [x] Unit tests for retry mechanism (Oct 2025 - 10 tests, 100% coverage)
- [x] Integration tests for tunnel connections (Oct 2025 - 8 tests covering all scenarios)
- [x] Integration tests for HTTP proxy (Oct 2025 - 7 tests covering proxy functionality)
- [x] Integration tests for circuit breaker (Oct 2025 - 6 tests for integration scenarios)
- [x] Integration tests for WebSocket (Oct 2025 - 9 tests for WebSocket tunneling)
- [x] E2E tests for local binaries (Oct 2025 - test-local.ps1/.sh with HTTP/HTTPS/WebSocket)
- [x] E2E tests for Docker containers (Oct 2025 - test-docker.ps1/.sh with all protocols)
- [x] Comprehensive testing documentation (Oct 2025 - docs/testing.md with all test types)
- [x] Test utilities and helpers (Oct 2025 - internal/integration/testutil.go)
- [ ] Performance and load testing (future enhancement)

### Documentation ✅ MOSTLY COMPLETE
- [x] Comprehensive README with quick start guides (Oct 2025)
- [x] Architecture design document (Oct 2025 - docs/architecture.md)
- [x] Product requirements document (Oct 2025 - docs/PRD.md)
- [x] Testing documentation (Oct 2025 - docs/testing.md with unit/integration/E2E guides)
- [x] Error handling documentation (Oct 2025 - docs/error-handling-improvements.md)
- [x] Integration test documentation (Oct 2025 - internal/integration/README.md)
- [ ] Cloud deployment guide
- [ ] Troubleshooting and FAQ expansion

### Cloud Deployment 🚧 NOT STARTED
- [ ] Deploy Tunnel Server to a cloud provider (initial deployment)
- [ ] Configure cloud networking and firewall rules
- [ ] Set up monitoring and alerting
- [ ] Document cloud deployment procedures

### Performance & Monitoring 🚧 IN PROGRESS
- [x] Implement concurrent request handling (Oct 2025 - goroutines and channels in place)
- [ ] Optimize connection pooling
- [ ] Add health checks and monitoring endpoints
- [ ] Implement metrics collection

---


## Phase 2: Cloud Deployment

### Deployment & Scaling
- [ ] Deploy Tunnel Server to chosen cloud provider (AWS, Azure, GCP, etc.)
- [ ] Implement container orchestration (Docker Compose, Kubernetes, or cloud container services)
- [ ] **Lambda Control Plane (NEW)**: Deploy Wake/Sleep/Kill Lambda functions
- [ ] **API Gateway (NEW)**: Set up HTTPS endpoints for lifecycle management
- [ ] **EventBridge Schedulers (NEW)**: Configure periodic and daily triggers
- [ ] Set up CI/CD pipeline for automated builds and deployments
- [ ] Implement scaling and monitoring for cloud deployment

### Configuration & Networking
- [ ] Handle dynamic IP address changes for cloud deployments
- [ ] Secure cloud networking and firewall rules
- [ ] **Agent Lifecycle Integration (NEW)**: Add wake/kill API calls on startup/shutdown
- [ ] **Server Metrics Integration (NEW)**: Emit CloudWatch metrics for activity tracking

---

## Phase 2.5: Lambda Control Plane Implementation (NEW)

### Lambda Functions
- [ ] Create `deployments/lambda/wake/` with Python implementation
- [ ] Create `deployments/lambda/sleep/` with CloudWatch metrics query logic
- [ ] Create `deployments/lambda/kill/` with immediate shutdown logic
- [ ] Create IAM roles for each Lambda (least-privilege policies)
- [ ] Add environment variable configuration (cluster/service names, thresholds)
- [ ] Implement error handling and logging in all Lambdas

### API Gateway Setup
- [ ] Create REST API in AWS API Gateway
- [ ] Configure `/wake` endpoint → Wake Lambda integration
- [ ] Configure `/kill` endpoint → Kill Lambda integration
- [ ] Set up API key authentication
- [ ] Configure throttling and rate limits (10 req/min per endpoint)
- [ ] Add CloudWatch logging for API Gateway
- [ ] Test endpoints with curl/Postman

### EventBridge Configuration
- [ ] Create Sleep Lambda schedule rule (rate: every X minutes, configurable)
- [ ] Create Kill Lambda schedule rule (cron: daily at specified time)
- [ ] Set up retry policies for Lambda invocations
- [ ] Add CloudWatch Events logging
- [ ] Test schedule triggers

### Agent Lifecycle Integration
- [ ] Create `internal/agent/lifecycle/` package
- [ ] Implement Wake API client with retry logic
- [ ] Implement Kill API client
- [ ] Add configuration for API endpoints and API key
- [ ] Add connection retry with configurable timeout (default 90s)
- [ ] Add connection retry interval (default 5s)
- [ ] Integrate wake call in agent startup flow
- [ ] Integrate kill call in agent shutdown flow
- [ ] Add logging for lifecycle events

### Server Metrics Integration
- [ ] Create `internal/server/metrics/` package
- [ ] Implement CloudWatch PutMetricData client
- [ ] Add atomic counters for active connections
- [ ] Add atomic timestamp for last activity
- [ ] Emit `ActiveConnections` metric (gauge)
- [ ] Emit `LastActivityEpochSeconds` metric (timestamp)
- [ ] Add configurable emission interval (default 60s)
- [ ] Add enable/disable flag in server config
- [ ] Update server to track connection lifecycle
- [ ] Update server to update last activity on each request

### CloudFormation Templates
- [ ] Create `deployments/cloudformation/lambda.yaml`
  - [ ] Lambda function definitions (Wake, Sleep, Kill)
  - [ ] IAM roles and policies
  - [ ] API Gateway REST API
  - [ ] EventBridge schedule rules
  - [ ] Outputs for API endpoints and keys
- [ ] Update `deployments/cloudformation/fargate.yaml`
  - [ ] Add CloudWatch PutMetricData permission to task role
  - [ ] Add environment variables for metrics config
- [ ] Create `deployments/cloudformation/params.lambda.example.json`
- [ ] Document deployment process in README

### Testing
- [ ] Unit tests for lifecycle client (`internal/agent/lifecycle`)
- [ ] Unit tests for metrics emitter (`internal/server/metrics`)
- [ ] Integration test: Wake Lambda → ECS update
- [ ] Integration test: Sleep Lambda → CloudWatch query → ECS update
- [ ] Integration test: Kill Lambda → ECS update
- [ ] End-to-end test: Agent startup → wake → connect → shutdown → kill
- [ ] End-to-end test: Idle detection → automatic sleep
- [ ] Load test: Multiple wake/kill calls (debounce validation)

### Documentation
- [ ] Update `docs/architecture.md` with Lambda control plane design ✅
- [ ] Update `docs/deployment.md` with Lambda deployment guide ✅
- [ ] Update `docs/plan.md` with implementation tasks ✅
- [ ] Update `docs/fargate.md` with Lambda integration details
- [ ] Update `README.md` with Lambda control plane overview
- [ ] Create `docs/lambda-control-plane.md` with detailed guide
- [ ] Add troubleshooting section for Lambda/API Gateway issues
- [ ] Document cost breakdown (Fargate + Lambda + API Gateway)

---

## Phase 3: Enhanced Features

### Security & Certificates
- [x] Implement basic mTLS authentication (COMPLETE - working in current code)
- [ ] Implement advanced certificate management (monitoring, alerts)
- [x] Private Certificate Authority (CA) infrastructure (COMPLETE - certs/ directory)
- [ ] Implement certificate renewal and rotation automation

### Configuration & Usability
- [x] Basic configuration options (COMPLETE - CLI, config files, environment variables)
- [x] Server IP configuration with CLI override (COMPLETE - implemented in agent)
- [ ] Configuration hot-reload capability
- [ ] Enhanced configuration validation

### Monitoring & Logging
- [x] Basic structured logging (COMPLETE - using logrus with contextual fields)
- [ ] Add monitoring endpoints and health checks
- [ ] Enhanced logging capabilities (metrics, performance tracking)

---

## Phase 4: Testing & Documentation

### Testing
- [ ] Comprehensive unit, integration, and performance tests
- [ ] End-to-end test scenarios with real servers
- [ ] Security audit and vulnerability assessment

### Documentation
- [ ] Complete user documentation and guides
- [ ] Finalize troubleshooting and FAQ sections
- [ ] Document cloud deployment and scaling procedures

---

---

## References
- [Architecture Design](architecture.md)
- [Product Requirements Document (PRD)](PRD.md)

For future phases and a full requirements list, see the PRD and architecture documents.
