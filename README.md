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
./scripts/generate-certs.sh  # macOS/Linux
.\scripts\generate-certs.ps1  # Windows

# 2. Run server and agent
make -f Makefile.<platform> run-server-local  # Terminal 1
make -f Makefile.<platform> run-agent-local   # Terminal 2

# 3. Set browser proxy to localhost:8080

# 4. Test
curl -x http://127.0.0.1:8080 http://example.com
```

📖 **Full setup**: [Deployment Guide](docs/deployment.md)

## Documentation

### Core Documentation
- **[Deployment Guide](docs/deployment.md)** - Local, Docker, AWS Fargate setup
- **[Architecture](docs/architecture.md)** - System design and components
- **[Testing](docs/testing.md)** - Unit, integration, and E2E tests

### Deployment-Specific
- **[Infrastructure as Code](docs/infrastructure.md)** - CloudFormation templates and automated deployment
- **[Docker](docs/docker.md)** - Container builds and networking
- **[AWS Fargate](docs/fargate.md)** - Cloud deployment guide
- **[Lambda Functions](docs/lambda.md)** - Automated lifecycle management

### Testing & Validation
- **[Testing Summary](docs/TESTING_SUMMARY.md)** - Unit tests, coverage, and validation results

### Project Planning
- **[Product Requirements](docs/product.md)** - Features and specifications
- **[Development Plan](docs/plan.md)** - Roadmap and status

## Disclaimer

⚠️ Users are responsible for compliance with organizational policies and local laws.

## License

Custom - See repository for details
