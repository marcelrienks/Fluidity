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

## Documentation Guide

### 📚 Complete Documentation Index

All documentation has been organized in the `docs/` folder for easy reference.

#### Getting Started

| Document | Summary | Best For |
|----------|---------|----------|
| **[Architecture](docs/architecture.md)** | System design, component overview, protocol details, mTLS security model, deployment architectures. Includes threat model and performance considerations. | Understanding how Fluidity works internally |
| **[Development Guide](docs/development.md)** | Local development workflow, build commands, architecture deep dive, code patterns, debugging tips. Comprehensive guide for developers from setup to testing. | Setting up development environment and contributing |
| **[Product Requirements](docs/product.md)** | Feature specification, user stories, success metrics, functional requirements for Phase 1-3. | Project scope and planning reference |

#### Deployment & Operations

| Document | Summary | Best For |
|----------|---------|----------|
| **[Deployment Guide](docs/deployment.md)** | Quick reference for all 5 deployment options (local, Docker, Fargate manual, CloudFormation, Lambda control plane) with cost comparison and troubleshooting. | Choosing and setting up your deployment |
| **[Docker Guide](docs/docker.md)** | Containerization approach, build process, networking for Docker Desktop, image sizes (~44MB), and troubleshooting. Explains why single-stage builds work in corporate environments. | Building and testing containerized deployment |
| **[AWS Fargate](docs/fargate.md)** | Step-by-step manual Fargate deployment: ECR setup, task definition, service creation, public IP retrieval. Cost ~$0.50-3/month. | Deploying server to AWS cloud manually |
| **[Infrastructure as Code](docs/infrastructure.md)** | CloudFormation templates for Fargate and Lambda stacks, parameterized deployment, drift detection, stack protection, monitoring dashboards, cost analysis, and Docker image management. | Automated repeatable infrastructure deployment |
| **[Lambda Functions](docs/lambda.md)** | Control plane architecture: Wake (start server), Sleep (auto-scale on idle), Kill (shutdown). API Gateway endpoints, EventBridge schedulers, IAM roles, cost optimization. | Automated lifecycle management with cost savings |
| **[Operational Runbook](docs/runbook.md)** | Daily operations procedures, manual lifecycle control, monitoring and alerting, troubleshooting guide, incident response, maintenance tasks. | Running Fluidity in production |
| **[Certificate Management](docs/certificate-management.md)** | Certificate generation and management, AWS Secrets Manager integration, local file management, security best practices. | Setting up and rotating TLS certificates |

#### Testing & Development

| Document | Summary | Best For |
|----------|---------|----------|
| **[Testing Guide](docs/testing.md)** | Three-tier testing strategy: unit tests (17), integration tests (30+), E2E tests (6). Coverage targets, CI/CD examples, debugging tips, performance profiling, and integration test organization. | Writing tests and validating code quality |

#### Planning & Status

| Document | Summary | Best For |
|----------|---------|----------|
| **[Development Plan](docs/plan.md)** | Project roadmap by phase: Phase 1 complete (core tunneling), Phase 2 in-progress (Lambda control plane), Phase 3 planned (CI/CD, hardening). Feature checklist and implementation status. | Tracking project progress and roadmap |

---

### Quick Lookup by Task

- **Just want to tunnel locally?** → [Quick Start](#quick-start) + [Deployment Guide](docs/deployment.md#option-a-local-development)
- **Setting up for development?** → [Development Guide](docs/development.md)
- **Testing containerization?** → [Docker Guide](docs/docker.md) + [Deployment Guide](docs/deployment.md#option-b-docker)
- **Deploying to AWS manually?** → [AWS Fargate](docs/fargate.md)
- **Setting up production infrastructure?** → [Infrastructure as Code](docs/infrastructure.md) + [Lambda Functions](docs/lambda.md)
- **Understanding the system?** → [Architecture](docs/architecture.md)
- **Running in production?** → [Operational Runbook](docs/runbook.md)
- **Writing tests?** → [Testing Guide](docs/testing.md)
- **Understanding requirements?** → [Product Requirements](docs/product.md)
- **Setting up certificates?** → [Certificate Management](docs/certificate-management.md)
- **Viewing project progress?** → [Development Plan](docs/plan.md)

---

## Disclaimer

⚠️ Users are responsible for compliance with organizational policies and local laws.

## License

Custom - See repository for details
