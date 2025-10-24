# Fluidity

**Secure HTTP/HTTPS tunneling to bypass restrictive firewalls**

![Status](https://img.shields.io/badge/status-phase_1_(alpha)-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Overview

Fluidity provides a secure tunnel solution to enable HTTP/HTTPS traffic through restrictive corporate firewalls. It consists of:

- **Tunnel Server**: Cloud-hosted Go application that forwards requests
- **Tunnel Agent**: Local Go application that acts as an HTTP/HTTPS proxy

Both components use mutual TLS (mTLS) for authentication and run in Docker containers.

## Architecture

```
   [Local Network]          [Internet]          [Cloud Provider]
┌───────────────────┐      ┌──────────┐        ┌──────────────────┐
│  Docker Desktop   │      │          │        │  Tunnel Server   │
│  ┌─────────────┐  │      │ Firewall │        │   (Go Binary)    │
│  │Tunnel Agent ├──┼──────┤ Bypass   ├────────┤   Containerized  │
│  │ (Go Binary) │  │      │          │        │                  │
│  └─────────────┘  │      └──────────┘        └──────────────────┘
└───────────────────┘
```

## Key Features

- ✅ HTTP/HTTPS/WebSocket tunneling
- ✅ mTLS authentication with private CA
- ✅ Automatic reconnection with exponential backoff
- ✅ Docker containerization (~43MB Alpine-based images)
- ✅ Cross-platform support (Windows/macOS/Linux)
- ✅ Comprehensive automated testing (75+ tests, ~77% coverage)

For detailed requirements and roadmap, see [docs/PRD.md](docs/PRD.md) and [docs/plan.md](docs/plan.md).

## Prerequisites

### Required
- **Go** (>= 1.21)
- **Make**
- **Docker Desktop**
- **OpenSSL**
- **Node.js** (>= 18.x, for WebSocket testing)

### Automated Setup

**Windows (PowerShell as Administrator):**
```powershell
.\scripts\setup-prerequisites.ps1
```

**macOS/Linux:**
```bash
chmod +x scripts/setup-prerequisites.sh
./scripts/setup-prerequisites.sh
```

After installing Node.js, install npm packages:
```bash
npm install
```

## Quick Start

### 1. Generate Certificates

**Windows:**
```powershell
.\scripts\generate-certs.ps1
```

**macOS/Linux:**
```bash
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

### 2. Build and Run

#### Option A: Local Binaries (Best for Development)

**Windows:**
```powershell
# Terminal 1 - Server
make -f Makefile.win run-server-local

# Terminal 2 - Agent
make -f Makefile.win run-agent-local
```

**macOS:**
```bash
# Terminal 1 - Server
make -f Makefile.macos run-server-local

# Terminal 2 - Agent
make -f Makefile.macos run-agent-local
```

**Linux:**
```bash
# Terminal 1 - Server
make -f Makefile.linux run-server-local

# Terminal 2 - Agent
make -f Makefile.linux run-agent-local
```

#### Option B: Docker Containers

**Build images:**
```powershell
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

**Run containers (Windows):**
```powershell
# Server
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.local.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server `
  ./fluidity-server --config ./config/server.yaml

# Agent
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.local.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent `
  ./fluidity-agent --config ./config/agent.yaml
```

**Run containers (macOS/Linux):**
```bash
# Server
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.local.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server \
  ./fluidity-server --config ./config/server.yaml

# Agent
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.local.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent \
  ./fluidity-agent --config ./config/agent.yaml
```

### 3. Configure Browser Proxy

Set browser's HTTP and HTTPS proxy to `localhost:8080`.

**Chrome/Edge (Windows):**
1. Settings → Network & Internet → Proxy
2. Enable "Use a proxy server"
3. Address: `127.0.0.1`, Port: `8080`

**Firefox:**
1. Settings → Network Settings
2. Manual proxy configuration
3. HTTP Proxy: `127.0.0.1`, Port: `8080`
4. HTTPS Proxy: `127.0.0.1`, Port: `8080`

### 4. Test

**CLI test:**
```powershell
# Windows
curl.exe -x http://127.0.0.1:8080 http://example.com -I
curl.exe -x http://127.0.0.1:8080 https://google.com -I

# macOS/Linux
curl -x http://127.0.0.1:8080 http://example.com -I
curl -x http://127.0.0.1:8080 https://google.com -I
```

**Browser test:**
- Visit http://example.com or https://google.com
- Check logs in both terminals for request flow

## Testing

### Quick Test Commands

```bash
# Unit tests (~1 second)
go test ./internal/shared/... -v

# Integration tests (~40 seconds)
go test ./internal/integration/... -v -timeout 5m

# All tests with coverage
go test ./internal/... -cover -timeout 5m
```

### Automated E2E Tests

**Windows:**
```powershell
# Test local binaries
.\scripts\test-local.ps1

# Test Docker containers
.\scripts\test-docker.ps1

# Skip build step
.\scripts\test-local.ps1 -SkipBuild
.\scripts\test-docker.ps1 -SkipBuild
```

**macOS/Linux:**
```bash
# Test local binaries
./scripts/test-local.sh

# Test Docker containers
./scripts/test-docker.sh

# Skip build step
./scripts/test-local.sh --skip-build
./scripts/test-docker.sh --skip-build
```

### Test Coverage Summary

- **Total Tests:** 75 (all passing ✅)
- **Unit Tests:** 49 tests, 72-100% coverage per package
- **Integration Tests:** 26 tests, 68.6% coverage
- **E2E Tests:** 6 scenarios (HTTP/HTTPS/WebSocket)
- **Overall Coverage:** ~77%

📖 **For complete testing documentation, see [docs/testing.md](docs/testing.md)**

## Build Commands Reference

**Native builds (no Docker):**
```bash
# Windows
make -f Makefile.win build

# macOS
make -f Makefile.macos build

# Linux
make -f Makefile.linux build
```

**Docker images:**
```bash
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

## Documentation

- **[Architecture](docs/architecture.md)** - Technical design and implementation details
- **[Product Requirements](docs/PRD.md)** - Full feature requirements and specifications
- **[Project Plan](docs/plan.md)** - Development roadmap and milestones
- **[Testing Guide](docs/testing.md)** - Comprehensive testing documentation

## Disclaimer

⚠️ **Important**: This tool is intended for legitimate use cases. Users are responsible for ensuring compliance with their organization's policies and local laws. The developers are not responsible for any misuse of this software.

## License

Custom license - see repository for details
