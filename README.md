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
- **Windows:** [Download Go](https://go.dev/dl/) and run the installer.
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


Fluidity provides OS-specific Makefiles for building and running the project. For most users, the commands are the same across Windows, macOS, and Linux:

**Windows/macOS/Linux:**
```sh
make -f Makefile.<os> build
make -f Makefile.<os> run-server-local
make -f Makefile.<os> run-agent-local
```
Replace `<os>` with `win`, `macos`, or `linux` as appropriate for your platform (e.g., `Makefile.win` for Windows).

> **Note:** The default `Makefile` is for reference and may not work natively on all platforms. Always use the OS-specific Makefile for your environment.

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


### 2. Build and Run Locally (Windows/macOS/Linux)
```sh
make build-local
make run-server-local
make run-agent-local
```

### 5. Configure Your Browser
Set your browser's HTTP proxy to `localhost:8080`.

### 6. Test Traffic
Browse to any HTTP website. You should see logs in both the agent and server terminals showing requests and responses flowing through the tunnel.

---

- The agent connects to the server at `127.0.0.1:8443`.
- Both agent and server run locally for easy debugging.
- Logs will show traffic passing from browser → agent → server → website → back to browser.

For advanced configuration, see the `configs/agent.local.yaml` and `configs/server.local.yaml` files.
