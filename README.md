# Fluidity

**Secure HTTP/HTTPS tunneling to bypass restrictive firewalls**

![Status](https://img.shields.io/badge/status-phase_1_(alpha)-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Overview

Fluidity provides a secure tunnel solution to enable HTTP/HTTPS traffic through restrictive corporate firewalls using mutual TLS authentication.

**Components:**
- **Tunnel Server**: Cloud-hosted Go application that forwards requests
- **Tunnel Agent**: Local Go application acting as HTTP/HTTPS proxy

Both run in Docker containers (~44MB Alpine-based images).

## Key Features

- ✅ HTTP/HTTPS/WebSocket tunneling
- ✅ mTLS authentication with private CA
- ✅ Automatic reconnection with exponential backoff
- ✅ Cross-platform support (Windows/macOS/Linux)
- ✅ Comprehensive automated testing (75+ tests, ~77% coverage)
- 🚧 AWS Lambda control plane for automated lifecycle management

## Quick Start

### Prerequisites
- Go 1.21+, Docker Desktop, OpenSSL, Node.js 18+ (for testing)
- **Setup**: Run platform-specific script in `scripts/setup-prerequisites.*`

### 1. Generate Certificates
```bash
# Windows
.\scripts\generate-certs.ps1

# macOS/Linux
./scripts/generate-certs.sh
```

### 2. Build and Run

**Local Binaries:**
```bash
# Windows
make -f Makefile.win run-server-local  # Terminal 1
make -f Makefile.win run-agent-local   # Terminal 2

# macOS
make -f Makefile.macos run-server-local
make -f Makefile.macos run-agent-local

# Linux
make -f Makefile.linux run-server-local
make -f Makefile.linux run-agent-local
```

**Docker Containers:**
```bash
# Build
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# Run (see docs/deployment.md for full commands)
```

### 3. Configure Browser
Set HTTP/HTTPS proxy to `localhost:8080`

### 4. Test
```bash
curl -x http://127.0.0.1:8080 http://example.com -I
```

📖 **For detailed setup, see [docs/deployment.md](docs/deployment.md)**

## Testing

```bash
# Quick test
go test ./internal/... -cover

# Automated E2E
.\scripts\test-local.ps1    # Windows
./scripts/test-local.sh     # macOS/Linux
```

📖 **For complete testing guide, see [docs/testing.md](docs/testing.md)**

## Deployment Options

- **Local Development** - Run binaries directly (see [Quick Start](#quick-start))
- **Docker** - Containerized deployment (see [docs/docker.md](docs/docker.md))
- **AWS Fargate** - Cloud deployment with on-demand scaling (see [docs/fargate.md](docs/fargate.md))
- **Lambda Control Plane** - Automated lifecycle management (see [docs/deployment.md](docs/deployment.md) Option E)

## Documentation

- **[Architecture](docs/architecture.md)** - Technical design and Lambda control plane
- **[Deployment Guide](docs/deployment.md)** - All deployment options
- **[Testing Guide](docs/testing.md)** - Comprehensive testing documentation
- **[Product Requirements](docs/PRD.md)** - Feature requirements and specifications
- **[Project Plan](docs/plan.md)** - Development roadmap

## Disclaimer

⚠️ **Important**: This tool is intended for legitimate use cases. Users are responsible for ensuring compliance with their organization's policies and local laws.

## License

Custom license - see repository for details
