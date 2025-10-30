# Fluidity

**Secure HTTP/HTTPS tunneling solution with mTLS authentication**

![Status](https://img.shields.io/badge/status-Phase_2-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Overview

Fluidity tunnels HTTP/HTTPS/WebSocket traffic through restrictive firewalls using mutual TLS authentication between a local agent and cloud-hosted server.

**Stack**: Go, Docker, AWS ECS Fargate, Lambda  
**Size**: ~44MB Alpine containers  
**Security**: mTLS with private CA

## Features

✅ HTTP/HTTPS/WebSocket tunneling  
✅ mTLS authentication  
✅ Auto-reconnection  
✅ Cross-platform (Windows/macOS/Linux)  
✅ 75+ tests, ~77% coverage  
🚧 Lambda-based lifecycle automation

## Quick Start

```bash
# 1. Generate certificates
./scripts/manage-certs.sh  # macOS/Linux
.\scripts\manage-certs.ps1  # Windows

# 2. Run server and agent
make -f Makefile.<platform> run-server-local  # Terminal 1
make -f Makefile.<platform> run-agent-local   # Terminal 2

# 3. Set browser proxy to localhost:8080

# 4. Test
curl -x http://127.0.0.1:8080 http://example.com
```

---

## Documentation

### Architecture

Fluidity uses a **client-server architecture** with mTLS authentication for secure tunneling through restrictive firewalls.

**Key Components:**
- **Agent** (local proxy): Accepts HTTP/HTTPS requests on port 8080, forwards to server via WebSocket tunnel
- **Server** (cloud-based): Receives tunneled requests, performs HTTP calls, returns responses
- **Protocol**: Custom WebSocket-based with request/response IDs, connection pooling, auto-reconnection
- **Security**: Mutual TLS with private CA certificates, no plaintext credentials

**Deployment Options:**
- Local development (both processes on same machine)
- Docker containerized (~44MB Alpine images)
- AWS Fargate (cloud-hosted server, local agent)
- Lambda control plane (automated wake/sleep/kill lifecycle)

**→ Full details:** [Architecture Documentation](docs/architecture.md)

---

### Development

Set up your local development environment to build, test, and contribute to Fluidity.

**Quick Setup:**
```bash
# 1. Generate certificates
./scripts/manage-certs.sh  # macOS/Linux
.\scripts\manage-certs.ps1  # Windows

# 2. Run locally
make -f Makefile.<platform> run-server-local  # Terminal 1
make -f Makefile.<platform> run-agent-local   # Terminal 2

# 3. Test
make -f Makefile.<platform> test
```

**Project Structure:**
- `cmd/` - Main entry points (server, agent, lambdas)
- `internal/core/` - Server and agent business logic
- `internal/shared/` - Reusable utilities (protocol, retry, circuit breaker, logging)
- `internal/lambdas/` - Control plane functions (wake, sleep, kill)

**Testing:** 75+ tests with ~77% coverage (unit, integration, E2E)

**→ Full details:** [Development Guide](docs/development.md)

---

### Deployment

Deploy Fluidity using one of five options, from local testing to full cloud infrastructure.

**Option 1: Local** - Both server and agent on same machine (development/testing)  
**Option 2: Docker** - Containerized with Docker Desktop  
**Option 3: AWS Fargate (Manual)** - Cloud-hosted server, ~$0.50-3/month  
**Option 4: CloudFormation** - Automated IaC deployment with monitoring  
**Option 5: Lambda Control Plane** - Automated lifecycle (wake/sleep/kill)

**Cost Comparison:**
- Local: Free
- Docker: Free (local)
- Fargate: $0.50-3/month (24/7) or $0.10-0.20/month (on-demand with Lambda)
- Lambda: ~$0.01/month (1000 invocations)

**→ Full details:** [Deployment Guide](docs/deployment.md)

---

### Docker

Build and run Fluidity in containerized environments with Docker.

**Build Commands:**
```bash
make -f Makefile.<platform> build-server
make -f Makefile.<platform> build-agent
```

**Image Details:**
- Base: Alpine Linux (minimal attack surface)
- Size: ~44MB per image
- Security: Non-root user, TLS certificates included
- Networking: Host networking for Docker Desktop compatibility

**Push to ECR:**
```bash
make -f Makefile.<platform> push-server
make -f Makefile.<platform> push-agent
```

**→ Full details:** [Docker Guide](docs/docker.md)

---

### AWS Fargate

Deploy the server to AWS ECS Fargate for cloud-hosted tunneling.

**Quick Deploy:**
```bash
# 1. Push to ECR
make -f Makefile.<platform> push-server

# 2. Create task definition (AWS Console or CLI)
# 3. Start service
aws ecs update-service --cluster fluidity --service server --desired-count 1

# 4. Get public IP
aws ecs describe-tasks --cluster fluidity --tasks <task-arn> | grep "publicIp"
```

**Configuration:**
- CPU: 256 (0.25 vCPU)
- Memory: 512 MB
- Networking: Public subnet with auto-assign public IP
- Cost: ~$0.50-3/month

**→ Full details:** [AWS Fargate Guide](docs/fargate.md)

---

### Infrastructure as Code

Automate infrastructure deployment using CloudFormation templates.

**Deploy with Script:**
```bash
# Deploy Fargate stack
./scripts/deploy-fluidity.sh fargate deploy

# Deploy Lambda control plane
./scripts/deploy-fluidity.sh lambda deploy
```

**Templates Included:**
- `fargate.yaml` - ECS cluster, task definitions, services, networking, monitoring
- `lambda.yaml` - Wake/Sleep/Kill functions, API Gateway, EventBridge schedulers

**Features:**
- Parameterized configuration (custom VPC, subnets, security groups)
- Stack protection against accidental deletion
- CloudWatch dashboards and alarms
- Drift detection and cost analysis

**→ Full details:** [Infrastructure Documentation](docs/infrastructure.md)

---

### Lambda Control Plane

Automate server lifecycle management to minimize costs with on-demand infrastructure.

**Functions:**
- **Wake** - Starts Fargate server (via API Gateway or scheduled)
- **Sleep** - Auto-scales to 0 after idle timeout (EventBridge every 5 min)
- **Kill** - Immediate shutdown (manual via API Gateway)

**Cost Optimization:**
- Fargate: $0.50-3/month (24/7) → $0.10-0.20/month (on-demand)
- Lambda: ~$0.01/month (1000 invocations)
- Total savings: ~90% for occasional use

**Endpoints:**
```bash
# Wake server
curl -X POST https://<api-id>.execute-api.us-east-1.amazonaws.com/prod/wake

# Kill server
curl -X POST https://<api-id>.execute-api.us-east-1.amazonaws.com/prod/kill
```

**→ Full details:** [Lambda Functions Guide](docs/lambda.md)

---

### Operations

Daily operations, monitoring, troubleshooting, and maintenance procedures for production environments.

**Key Tasks:**
- Manual lifecycle control (start/stop server)
- Monitoring: CloudWatch dashboards, metrics, logs, alarms
- Certificate rotation (quarterly recommended)
- Troubleshooting: Connection failures, performance issues, certificate problems

**Health Checks:**
```bash
# Check server status
aws ecs describe-services --cluster fluidity --services server

# View logs
aws logs tail /ecs/fluidity-server --follow
```

**→ Full details:** [Operational Runbook](docs/runbook.md)

---

### Certificate Management

Generate and manage mTLS certificates for Fluidity authentication.

**Generate Certificates:**
```bash
./scripts/manage-certs.sh  # macOS/Linux
.\scripts\manage-certs.ps1  # Windows
```

**Upload to AWS Secrets Manager:**
```bash
./scripts/manage-certs.sh --upload
```

**Certificate Rotation:**
1. Generate new certificates
2. Upload to Secrets Manager
3. Restart server and agent
4. Verify connectivity

**Security:** Private CA, 2048-bit RSA, 365-day validity, SHA-256

**→ Full details:** [Certificate Management Guide](docs/certificate-management.md)

---

### Testing

Three-tier testing strategy ensuring code quality and reliability.

**Test Tiers:**
- **Unit Tests** (17): Individual component testing, mock dependencies
- **Integration Tests** (30+): Multi-component workflows, real dependencies
- **E2E Tests** (6): Full system validation, client → agent → server → target

**Coverage:** ~77% overall (target: 80%)

**Run Tests:**
```bash
# All tests
make -f Makefile.<platform> test

# With coverage
make -f Makefile.<platform> coverage

# Specific package
go test -v ./internal/core/agent/...
```

**→ Full details:** [Testing Guide](docs/testing.md)

---

### Product Requirements

Feature specifications, user stories, and success metrics for Fluidity.

**Core Features (Phase 1 ✅):**
- HTTP/HTTPS/WebSocket tunneling
- mTLS authentication
- Auto-reconnection with backoff
- Cross-platform support

**Lambda Control Plane (Phase 2 🚧):**
- Wake/Sleep/Kill automation
- Cost optimization (on-demand)

**Production Hardening (Phase 3 📋):**
- CI/CD pipeline
- Enhanced monitoring
- Rate limiting and DDoS protection

**→ Full details:** [Product Requirements](docs/product.md)

---

### Development Roadmap

Project status and implementation roadmap by phase.

**Phase 1 (Complete ✅):**
- Core tunneling functionality
- Docker containerization
- Manual Fargate deployment
- 75+ tests, ~77% coverage

**Phase 2 (In Progress 🚧):**
- Lambda control plane (Wake/Sleep/Kill)
- CloudFormation automation
- Cost optimization

**Phase 3 (Planned 📋):**
- CI/CD with GitHub Actions
- Enhanced security (rate limiting, DDoS)
- Production monitoring improvements

**→ Full details:** [Development Plan](docs/plan.md)

---

## Disclaimer

⚠️ Users are responsible for compliance with organizational policies and local laws.

## License

Custom - See repository for details
