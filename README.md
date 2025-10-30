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

## Architecture

Fluidity uses a **client-server architecture** with mTLS authentication for secure tunneling through restrictive firewalls.

**Key Components:**
- **Agent** (local proxy): Accepts HTTP/HTTPS requests on port 8080, forwards to server via WebSocket tunnel
- **Server** (cloud-based): Receives tunneled requests, performs HTTP calls, returns responses
- **Protocol**: Custom WebSocket-based with request/response IDs, connection pooling, auto-reconnection
- **Security**: Mutual TLS with private CA certificates, no plaintext credentials

**→ Full details:** [Architecture Documentation](docs/architecture.md)

## Deployment

### Development

Local development and testing options for building and contributing to Fluidity.

#### **🏠 Local**
Run both server and agent on your local machine.

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

**Best for:** Development, testing, debugging  
**Cost:** Free

#### **🐳 Docker**
Build and run containerized images locally with Docker Desktop.

**Commands:**
```bash
# Build images
make -f Makefile.<platform> build-server
make -f Makefile.<platform> build-agent

# Run containers
make -f Makefile.<platform> run-server
make -f Makefile.<platform> run-agent
```

**Details:** Alpine Linux base, ~44MB per image, includes TLS certificates  
**Best for:** Testing containerization before cloud deployment  
**Cost:** Free

**Project Structure:**
- `cmd/` - Main entry points (server, agent, lambdas)
- `internal/core/` - Server and agent business logic
- `internal/shared/` - Reusable utilities (protocol, retry, circuit breaker, logging)
- `internal/lambdas/` - Control plane functions (wake, sleep, kill)

**Testing:** 75+ tests with ~77% coverage (unit, integration, E2E)

**→ Full details:** [Development Guide](docs/development.md) | [Docker Guide](docs/docker.md)

### Production (Recommended: CloudFormation)
Deploy to AWS using **Infrastructure as Code** for automated, repeatable, cost-effective infrastructure.

**Why CloudFormation:**
- ✅ Single command deployment
- ✅ Repeatable and version-controlled
- ✅ Integrated monitoring and dashboards
- ✅ Cost-optimized (~$0.11-0.21/month with Lambda control plane)

**Quick Deploy:**
```bash
./scripts/deploy-fluidity.sh fargate deploy  # Deploy server infrastructure
./scripts/deploy-fluidity.sh lambda deploy   # Deploy control plane
```

#### **💻 Agent (Local)**
The agent runs locally on your machine and connects to the cloud-hosted server.

**Setup:**
```bash
# 1. Generate certificates (if not already done)
./scripts/manage-certs.sh

# 2. Configure agent
# Edit configs/agent.yaml with Fargate server public IP

# 3. Run agent
make -f Makefile.<platform> run-agent-local
```

**Configuration:**
- Connects to Fargate server via WebSocket over mTLS
- Proxy port: 8080 (configurable)
- Auto-reconnection with exponential backoff

#### **☁️ Server (Fargate)**

Serverless container platform running the Fluidity server without managing EC2 instances.

**What's deployed:**
- ECS cluster with Fargate launch type
- Task definition (0.25 vCPU, 512 MB memory)
- VPC, subnets, security groups
- CloudWatch logs and monitoring
- Public IP for agent connectivity

**CloudFormation:**
```bash
# Deploy via script
./scripts/deploy-fluidity.sh fargate deploy

# Or use template directly
aws cloudformation create-stack \
  --stack-name fluidity-fargate \
  --template-body file://deployments/cloudformation/fargate.yaml \
  --parameters file://deployments/cloudformation/params.json
```

**Manual Deployment:**
```bash
# 1. Push image to ECR
make -f Makefile.<platform> push-server

# 2. Create task definition (AWS Console or CLI)
# 3. Start service
aws ecs update-service --cluster fluidity --service server --desired-count 1

# 4. Get public IP
aws ecs describe-tasks --cluster fluidity --tasks <task-arn> | grep "publicIp"
```

**Cost:** $0.50-3/month (24/7) or $0.10-0.20/month (on-demand with Lambda)

**→ Details:** [Infrastructure Documentation](docs/infrastructure.md) | [AWS Fargate Guide](docs/fargate.md)

#### **⚡ Control Plane (Lambda + API Gateway)**

Automated lifecycle management to minimize costs with on-demand server operation.

**Components:**
- **Wake Function** - Starts Fargate server (API Gateway endpoint)
- **Sleep Function** - Auto-scales to 0 after idle (EventBridge scheduler, every 5 min)
- **Kill Function** - Immediate shutdown (API Gateway endpoint)
- **API Gateway** - HTTP endpoints for Wake/Kill functions
- **EventBridge** - Scheduled Sleep automation

**CloudFormation:**
```bash
# Deploy via script
./scripts/deploy-fluidity.sh lambda deploy

# Or use template directly
aws cloudformation create-stack \
  --stack-name fluidity-lambda \
  --template-body file://deployments/cloudformation/lambda.yaml \
  --capabilities CAPABILITY_IAM
```

**API Usage:**
```bash
# Wake server (start on-demand)
curl -X POST https://<api-id>.execute-api.us-east-1.amazonaws.com/prod/wake

# Kill server (immediate shutdown)
curl -X POST https://<api-id>.execute-api.us-east-1.amazonaws.com/prod/kill
```

**Cost Optimization:**
- Fargate: $0.50-3/month (24/7) → $0.10-0.20/month (on-demand)
- Lambda: ~$0.01/month (1000 invocations)
- **Total: ~$0.11-0.21/month** (90% savings for occasional use)

**→ Details:** [Infrastructure Documentation](docs/infrastructure.md) | [Lambda Functions Guide](docs/lambda.md)

#### **🔐 Certificates**

mTLS certificates for secure authentication between agent and server.

**Generate Certificates:**
```bash
./scripts/manage-certs.sh              # Local files only
./scripts/manage-certs.sh --upload     # Upload to AWS Secrets Manager
```

**For Production (AWS):**
1. Generate certificates with `--upload` flag
2. CloudFormation automatically configures Fargate to use Secrets Manager
3. Agent uses local certificate files

**Local Files (./certs/):**
- `ca.crt`, `ca.key` - CA certificate and key
- `server.crt`, `server.key` - Server certificate
- `client.crt`, `client.key` - Client certificate

**AWS Secrets Manager:**
- Secret name: `fluidity/certificates`
- Contains Base64-encoded `cert_pem`, `key_pem`, `ca_pem`

**Certificate Rotation:**
1. Generate new certificates with `--upload`
2. Restart Fargate server (task pulls latest from Secrets Manager)
3. Restart local agent with new client certificates

**Security:** Private CA, 2048-bit RSA, SHA-256

**→ Full details:** [Certificate Guide](docs/certificate.md)

**→ Full deployment details:** [Deployment Guide](docs/deployment.md)

## Operations

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

## Testing

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

## Product Requirements

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

## Development Roadmap

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

## Disclaimer

⚠️ Users are responsible for compliance with organizational policies and local laws.

## License

Custom - See repository for details
