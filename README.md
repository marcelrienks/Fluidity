# Fluidity

**Allowing http traffic to flow freely**

Provides a way for enabling HTTP traffic, tunnelling, and routing.
Or in layman's terms 'bypass corporate firewall blocking useful sites'

![Status](https://img.shields.io/badge/status-planning-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Project Overview

Fluidity is currently in the **planning phase** and aims to create a robust HTTP tunnel solution consisting of two main components:

- **Tunnel Server**: A Go-based server application deployed to a cloud service provider
- **Tunnel Agent**: A Go-based client agent running locally

This architecture enables HTTP traffic to bypass restrictive corporate firewalls by establishing a secure tunnel between the local agent and the cloud-hosted server.

## Intended Architecture

```
   [Local Network]          [Internet]          [Cloud Provider]
┌───────────────────┐      ┌──────────┐        ┌──────────────────┐
│  Docker Desktop   │      │          │        │  Tunnel Server   │
│  ┌─────────────┐  │      │ Firewall │        │   (Go Binary)    │
│  │Tunnel Agent ├──┼──────┤ Bypass   ├────────┤   Containerized  │
│  │ (Go Binary) │  │      │          │        │                  │
│  │Containerized│  │      │          │        │                  │
│  └─────────────┘  │      └──────────┘        └──────────────────┘
└───────────────────┘
```

## Key Features (Planned)

- **Go-based Implementation**: Both server and agent written in Go for performance and cross-platform compatibility
- **Containerized Deployment**: Docker containers for easy deployment and management
- **Cloud-hosted Server**: Deployed to major cloud service providers for reliability and global access
- **Local Agent**: Runs within Docker Desktop for easy local setup and management
- **HTTP Traffic Tunneling**: Secure routing of HTTP requests through the tunnel
- **Firewall Bypass**: Designed to work around restrictive corporate network policies

## Technology Stack

- **Language**: Go
- **Containerization**: Docker
- **Deployment**: Cloud service provider (TBD)
- **Local Runtime**: Docker Desktop

## Current Status

This project is currently in the **planning and design phase**. Implementation has not yet begun.

### Roadmap

1. **Architecture Design** - Define detailed system architecture and communication protocols
2. **Server Development** - Implement the Go-based tunnel server
3. **Agent Development** - Implement the Go-based tunnel agent
4. **Containerization** - Create Docker images for both components
5. **Cloud Deployment** - Deploy server to chosen cloud provider
6. **Testing & Validation** - Comprehensive testing of tunnel functionality
7. **Documentation** - Complete user guides and deployment instructions

## Prerequisites

Before building or running Fluidity, ensure you have the following installed:

### Go (>= 1.21)
- **Windows:** Either download and run the installer from [go.dev](https://go.dev/dl/), or install via Chocolatey:
  ```powershell
  choco install golang
  ```
  After installation, open a new PowerShell and verify:
  ```powershell
  go version
  ```
- **macOS:** Use Homebrew:
  ```bash
  brew install go
  ```
  or download from [go.dev](https://go.dev/dl/).
- **Linux:** Use your package manager (e.g., Ubuntu):
  ```bash
  sudo apt install golang-go
  ```
  or download from [go.dev](https://go.dev/dl/).

### Make
- **Windows:** Install via [Chocolatey](https://community.chocolatey.org/packages/make):
  ```powershell
  choco install make
  ```
- **macOS:** Use Homebrew:
  ```bash
  brew install make
  ```
- **Linux:** Use your package manager (e.g., Ubuntu):
  ```bash
  sudo apt install make
  ```

### Docker
- **Windows/macOS/Linux:** [Download Docker Desktop](https://www.docker.com/products/docker-desktop) and follow the installation instructions for your OS.

### OpenSSL
- **Windows:** It is recommended to install the `openssl.light` package for best compatibility:
  ```powershell
  choco install openssl.light
  ```
  If you have already installed `openssl` or `openssl.light`, you can verify with:
  ```powershell
  openssl version
  ```
- **macOS:** Use Homebrew:
  ```bash
  brew install openssl
  ```
- **Linux:** Use your package manager (e.g., Ubuntu):
  ```bash
  sudo apt install openssl
  ```

> **Note:** If you do not wish to use `make`, you can run the build commands manually as described in the Quick Start Guide.

## Using the Makefiles

Fluidity provides OS-specific Makefiles for building and running the project. Always use the Makefile that matches your operating system: `Makefile.win` (Windows), `Makefile.macos` (macOS), or `Makefile.linux` (Linux).

### Build Types Explained

**Native Builds** (`build`, `run-*-local`):
- Compiles Go binaries for your current OS (`.exe` on Windows, native binaries on macOS/Linux)
- No Docker required
- Best for local development and debugging with breakpoints in VS Code
- Example:
  ```powershell
  make -f Makefile.win build
  # Creates: build/fluidity-agent.exe and build/fluidity-server.exe
  ```

**Docker Images - Standard** (`docker-build-*`):
- Uses multi-stage Dockerfile with `golang:1.21-alpine` builder and `alpine:latest` runtime
- Includes system CA certificates for outbound HTTPS
- Suitable for production deployment
- Requires Docker Hub access to pull base images
- Image size: ~20-30MB

**Docker Images - Scratch** (`docker-build-*-scratch`):
- Cross-compiles static Linux binary (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`)
- Builds from `scratch` (no base OS, just the binary)
- No shell, no package manager, no CA certificates
- Useful when Docker Hub pulls are blocked
- Image size: ~14MB
- Note: Cannot make outbound HTTPS calls unless you mount CA certificates

### Quick Command Reference

**Windows:**
```powershell
# Native build and run (no Docker)
make -f Makefile.win build
make -f Makefile.win run-server-local  # Terminal 1
make -f Makefile.win run-agent-local   # Terminal 2

# Docker images (standard)
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# Docker images (scratch - for restricted networks)
make -f Makefile.win docker-build-server-scratch
make -f Makefile.win docker-build-agent-scratch
```

**macOS:**
```bash
# Native build and run (no Docker)
make -f Makefile.macos build
make -f Makefile.macos run-server-local  # Terminal 1
make -f Makefile.macos run-agent-local   # Terminal 2

# Docker images (standard)
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
```

**Linux:**
```bash
# Native build and run (no Docker)
make -f Makefile.linux build
make -f Makefile.linux run-server-local  # Terminal 1
make -f Makefile.linux run-agent-local   # Terminal 2

# Docker images (standard - auto-fallback to scratch if pulls fail)
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

> **Note:** The default `Makefile` is for reference only. Always use the OS-specific Makefile for your environment.

## Disclaimer

⚠️ **Important**: This tool is intended for legitimate use cases such as accessing necessary resources for work or personal use. Users are responsible for ensuring compliance with their organization's network policies and local laws. The developers are not responsible for any misuse of this software.

## Quick Start Guide (Local Development)

### 1. Generate Development Certificates

**Windows (PowerShell):**
```powershell
./scripts/generate-certs.ps1
```
**macOS/Linux:**
```bash
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

### 2. Build and Run Locally

You have two main options for running Fluidity locally:

#### Option A: Native Binaries (No Docker - Easiest for Debugging)

Build and run the server and agent as native executables on your machine:

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

This approach:
- Runs binaries directly on your OS (no containers)
- Uses local configs: `configs/server.local.yaml` and `configs/agent.local.yaml`
- Best for debugging with VS Code breakpoints
- Logs appear directly in your terminal

#### Option B: Docker Containers (Production-Like Environment)

Build Docker images and run them as containers:

**Step 1: Build Images**

Standard images (includes CA certificates, ~20-30MB):
```powershell
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent

# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux (auto-fallback to scratch if pulls fail)
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

Scratch images (minimal, no CA certs, ~14MB - use if Docker Hub pulls are blocked):
```powershell
# Windows
make -f Makefile.win docker-build-server-scratch
make -f Makefile.win docker-build-agent-scratch
```

**Step 2: Run Containers**

Windows (PowerShell):

```powershell
# Server (standard image)
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.local.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server `
  ./fluidity-server --config ./config/server.yaml

# Agent (standard image)
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.local.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent `
  ./fluidity-agent --config ./config/agent.yaml

# If you built scratch images, omit the binary path and pass only flags:
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.local.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server `
  --config /root/config/server.yaml
```

macOS/Linux (bash):

```bash
# Server (standard image)
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.local.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server \
  ./fluidity-server --config ./config/server.yaml

# Agent (standard image)
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.local.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent \
  ./fluidity-agent --config ./config/agent.yaml

# For scratch images, pass flags only
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.local.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server \
  --config /root/config/server.yaml
```

Tip:
- Scratch images don't include system CA certificates. If the server or agent needs to make outbound HTTPS requests, prefer the standard images (Alpine-based) or mount a CA bundle into the container.
- Standard images use `./fluidity-server` or `./fluidity-agent` as the entrypoint binary path
- Scratch images use `/fluidity-server` or `/fluidity-agent` (absolute path, no shell)

### 3. Configure Your Browser
Set your browser's HTTP proxy to `localhost:8080`.

### 4. Test Traffic
Browse to any HTTP website. You should see logs in both the agent and server terminals showing requests and responses flowing through the tunnel.

---

- The agent connects to the server at `127.0.0.1:8443`.
- Both agent and server run locally for easy debugging.
- Logs will show traffic passing from browser → agent → server → website → back to browser.

For advanced configuration, see the `configs/agent.local.yaml` and `configs/server.local.yaml` files.
