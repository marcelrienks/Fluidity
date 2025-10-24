# Fluidity

**Allowing http traffic to flow freely**

Provides a way for enabling HTTP traffic, tunnelling, and routing.
Or in layman's terms 'bypass corporate firewall blocking useful sites'

![Status](https://img.shields.io/badge/status-phase_1_(alpha)-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Project Overview

Fluidity is in **Phase 1 (core infrastructure, alpha)** and provides a robust HTTP tunnel solution consisting of two main components:

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

## Key Features

- **Go-based Implementation**: Both Tunnel Server (cloud) and Tunnel Agent (local) written in Go for performance and cross-platform compatibility
- **Containerized Deployment**: Docker containers for easy deployment and management
- **Cloud-hosted Tunnel Server**: Deployed to major cloud service providers for reliability and global access
- **Local Tunnel Agent**: Runs within Docker Desktop for easy local setup and management
- **HTTP/HTTPS Traffic Tunneling**: Secure routing of both HTTP and HTTPS requests through the tunnel via CONNECT
- **mTLS Authentication**: Mutual TLS authentication with private CA for secure, authenticated connections
- **Firewall Bypass**: Designed to work around restrictive corporate network policies
- **Planned Protocol Support**: HTTPS/CONNECT tunneling and WebSocket support (see [PRD](docs/PRD.md) for requirements)

> **Note:** HTTPS/CONNECT and WebSocket tunneling are requirements per the [Product Requirements Document](docs/PRD.md), but are not yet implemented. These are planned for future phases.

## Technology Stack

- **Language**: Go
- **Containerization**: Docker
- **Deployment**: Cloud service provider (TBD)
- **Local Runtime**: Docker Desktop

## Current Status

- Agent runs a local HTTP/HTTPS proxy on port 8080
- Agent connects to server over mTLS using dev certificates (generated with the provided scripts)
- Server accepts client-authenticated TLS and forwards HTTP/HTTPS requests to target sites
- End-to-end HTTP and HTTPS browsing via the agent proxy is verified and working
- HTTPS CONNECT tunneling fully implemented for secure website access

## Project Planning & Architecture

For all outstanding work, phase steps, and actionable items, see the [Project Plan](docs/plan.md).

For technical architecture details, see the [Architecture Design](docs/architecture.md).

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

**Docker Images** (`docker-build-*`):
- Uses multi-stage Dockerfile with `golang:1.21-alpine` builder
- Runtime base: **alpine/curl:latest** - includes CA certificates and debugging tools
- Supports outbound HTTPS requests (required for tunnel functionality)
- Image size: ~43MB (20.5MB base + 22-23MB binary)
- Includes shell for debugging and troubleshooting

### Quick Command Reference

**Windows:**
```powershell
# Native build and run (no Docker)
make -f Makefile.win build
make -f Makefile.win run-server-local  # Terminal 1
make -f Makefile.win run-agent-local   # Terminal 2

# Docker images
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent
```

**macOS:**
```bash
# Native build and run (no Docker)
make -f Makefile.macos build
make -f Makefile.macos run-server-local  # Terminal 1
make -f Makefile.macos run-agent-local   # Terminal 2

# Docker images
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
```

**Linux:**
```bash
# Native build and run (no Docker)
make -f Makefile.linux build
make -f Makefile.linux run-server-local  # Terminal 1
make -f Makefile.linux run-agent-local   # Terminal 2

# Docker images
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```


## Current Progress (Oct 24, 2025)

Validated end-to-end HTTP and HTTPS tunneling via Docker containers with dev mTLS certs:

- Built and ran server and agent containers on a user-defined Docker network
- Agent established a TLS connection to the server (TLS 1.3) using client certs
- Successfully proxied HTTP requests via `curl -x http://127.0.0.1:8080 http://example.com -I`
- ✅ **HTTPS CONNECT tunneling fully implemented and working**
- Successfully proxied HTTPS requests via `curl -x http://127.0.0.1:8080 https://example.com -I`
- Full web browsing support for both HTTP and HTTPS websites

## Disclaimer

⚠️ **Important**: This tool is intended for legitimate use cases such as accessing necessary resources for work or personal use. Users are responsible for ensuring compliance with their organization's network policies and local laws. The developers are not responsible for any misuse of this software.

---

For a full list of requirements and planned features, see the [Product Requirements Document (PRD)](docs/PRD.md).

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

**Step 2: Run Containers**

Windows (PowerShell):

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

macOS/Linux (bash):

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

### 3. Configure Your Browser
Set your browser's HTTP **and HTTPS** proxy to `localhost:8080`. Both protocols are fully supported.

### 4. Test Traffic
Browse to any HTTP or HTTPS website. You should see logs in both the agent and server terminals showing requests and responses flowing through the tunnel.
For a quick CLI check on Windows:

```powershell
# Test HTTP
curl.exe -x http://127.0.0.1:8080 http://example.com -I

# Test HTTPS
curl.exe -x http://127.0.0.1:8080 https://example.com -I
```

## Simple Browser Test (Windows)

Both HTTP and HTTPS sites are fully supported. Try any of these:

- https://google.com
- https://github.com
- http://example.com
- http://neverssl.com

### Chrome/Edge (use system proxy)

1. Open Windows Settings → Network & Internet → Proxy
2. Turn on "Use a proxy server"
3. Address: `127.0.0.1`  Port: `8080`
4. Apply/Save
5. Open Chrome/Edge and browse to any HTTP or HTTPS site (e.g., `https://google.com`)
6. Watch logs (optional):

```powershell
docker logs --tail 50 -f fluidity-agent
docker logs --tail 50 -f fluidity-server
```

To revert: turn off “Use a proxy server” in Windows proxy settings.

### Firefox (manual proxy)

1. Firefox → Settings → General → Network Settings → Settings…
2. Select "Manual proxy configuration"
3. HTTP Proxy: `127.0.0.1`  Port: `8080`
4. HTTPS Proxy: `127.0.0.1`  Port: `8080`
5. OR check "Use this proxy server for all protocols"
6. OK, then browse to any HTTP or HTTPS site

To revert: set "No proxy" or "Use system proxy settings".

---

- The agent connects to the server at `127.0.0.1:8443`.
- Both agent and server run locally for easy debugging.
- Logs will show traffic passing from browser → agent → server → website → back to browser.
- Full support for both HTTP and HTTPS websites via CONNECT tunneling.

For advanced configuration, see the `configs/agent.local.yaml` and `configs/server.local.yaml` files.

## Automated End-to-End Testing

Automated test scripts are available to quickly validate both Docker containers and local binaries.

### Testing Docker Containers

**Windows (PowerShell):**

```powershell
# Run full test (build + test)
.\scripts\test-docker.ps1

# Skip build and use existing binaries
.\scripts\test-docker.ps1 -SkipBuild

# Keep containers running for manual inspection
.\scripts\test-docker.ps1 -SkipBuild -KeepContainers

# Test with custom URL
.\scripts\test-docker.ps1 -SkipBuild -TestUrl "http://httpbin.org/get"
```

**macOS/Linux (Bash):**

```bash
# Run full test (build + test)
./scripts/test-docker.sh

# Skip build and use existing binaries
./scripts/test-docker.sh --skip-build

# Keep containers running for manual inspection
./scripts/test-docker.sh --skip-build --keep-containers

# Test with custom URL
./scripts/test-docker.sh --skip-build --test-url "http://httpbin.org/get"
```

### Testing Local Binaries (Native Execution)

Test native binaries running directly on your host (no Docker required):

**Windows (PowerShell):**

```powershell
# Run full test (build + test)
.\scripts\test-local.ps1

# Skip build and use existing binaries
.\scripts\test-local.ps1 -SkipBuild

# Test with custom URL
.\scripts\test-local.ps1 -SkipBuild -TestUrl "http://httpbin.org/get"

# Use custom ports
.\scripts\test-local.ps1 -SkipBuild -ProxyPort 8081 -ServerPort 8444
```

**macOS/Linux (Bash):**

```bash
# Run full test (build + test)
./scripts/test-local.sh

# Skip build and use existing binaries
./scripts/test-local.sh --skip-build

# Test with custom URL
./scripts/test-local.sh --skip-build --test-url "http://httpbin.org/get"

# Use custom ports
./scripts/test-local.sh --skip-build --proxy-port 8081 --server-port 8444
```

The local binary test:
- Builds native executables for your OS (Windows/macOS/Linux)
- Verifies all required files exist (binaries, configs, certificates)
- Checks that ports are available
- Starts server and agent processes
- Tests HTTP tunneling through the proxy
- Automatically cleans up processes on exit
- Saves logs to `logs/test-server.out` and `logs/test-agent.out`

### Docker Test Workflow

The Docker test script performs a complete 9-step workflow:

1. **Build Binaries** - Compiles Linux binaries (optional with `-SkipBuild`/`--skip-build`)
2. **Build Docker Images** - Creates test images using alpine/curl base (~43MB each)
3. **Setup Network** - Creates Docker network `fluidity-test-net` for container communication
4. **Cleanup** - Removes any existing containers named `fluidity-server` and `fluidity-agent`
5. **Start Server** - Launches server container with TLS configuration
6. **Start Agent** - Launches agent container with proxy port 8081
7. **Wait** - Allows 3 seconds for tunnel initialization
8. **Test HTTP Tunnel** - Verifies HTTP proxy functionality with httpbin.org
9. **Additional Tests** - Tests api.github.com and example.com for regression

**Output:** Color-coded status messages showing success/failure at each step, with container logs on failure.

**Test Port:** Uses port 8081 to avoid conflicts with existing containers on port 8080.

### Test Script Parameters

**Docker Test (test-docker.ps1 / test-docker.sh):**

Windows PowerShell:
- `-SkipBuild` - Skip binary compilation, use existing binaries in `build/` directory
- `-KeepContainers` - Don't remove test containers after completion (useful for debugging)
- `-TestUrl "url"` - Override default test URL (default: `http://httpbin.org/get`)

macOS/Linux Bash:
- `--skip-build` - Skip binary compilation, use existing binaries in `build/` directory
- `--keep-containers` - Don't remove test containers after completion (useful for debugging)
- `--test-url "url"` - Override default test URL (default: `http://httpbin.org/get`)

**Local Binary Test (test-local.ps1 / test-local.sh):**

Windows PowerShell:
- `-SkipBuild` - Skip binary compilation, use existing binaries
- `-TestUrl "url"` - Override default test URL (default: `http://httpbin.org/get`)
- `-ProxyPort 8081` - Custom proxy port (default: 8080)
- `-ServerPort 8444` - Custom server port (default: 8443)

macOS/Linux Bash:
- `--skip-build` - Skip binary compilation, use existing binaries
- `--test-url "url"` - Override default test URL (default: `http://httpbin.org/get`)
- `--proxy-port 8081` - Custom proxy port (default: 8080)
- `--server-port 8444` - Custom server port (default: 8443)

### Expected Test Output

**Successful Docker Test:**
```
=== Fluidity End-to-End Docker Test ===

[Step 1] Skipping build
[Step 2] Building Docker images
  [OK] Images built
[Step 3] Setting up network
  [OK] Network ready
[Step 4] Cleaning old containers
  [OK] Cleaned
[Step 5] Starting server
  [OK] Server started
[Step 6] Starting agent
  [OK] Agent started
[Step 7] Waiting for initialization...
  [OK] Ready
[Step 8] Testing HTTP tunnel
  URL: http://httpbin.org/get
  [OK] Test passed (HTTP 200)
[Step 9] Additional tests
  GitHub API: HTTP 200
  example.com: HTTP 200

========================================
  ALL TESTS PASSED!
========================================

Cleaning up test containers...
  [OK] Cleanup complete
```

**Failed Test:**
If a test fails, the script will display:
- Error message indicating which step failed
- Server container logs (last 10 lines)
- Agent container logs (last 10 lines)
- Exit with code 1

**Successful Local Binary Test:**
```
=== Fluidity Local Binary Test ===

[Step 1] Skipping build
[Step 2] Verifying files
  [OK] All files verified
[Step 3] Checking ports
  [OK] Ports available
[Step 4] Starting server
  Command: C:\...\build\fluidity-server.exe --config C:\...\configs\server.local.yaml
  [OK] Server started (PID: 12345)
[Step 5] Starting agent
  Command: C:\...\build\fluidity-agent.exe --config C:\...\configs\agent.local.yaml
  [OK] Agent started (PID: 12346)
[Step 6] Waiting for initialization...
  [OK] Processes running
[Step 7] Testing HTTP tunnel
  URL: http://httpbin.org/get
  [OK] Test passed (HTTP 200)
[Step 8] Additional tests
  GitHub API: HTTP 200
  example.com: HTTP 200

========================================
  ALL TESTS PASSED!
========================================

Log files:
  Server: logs\test-server.out
  Agent:  logs\test-agent.out

Cleaning up processes...
  Stopping server...
  Stopping agent...
  [OK] Cleanup complete
```

**Failed Local Binary Test:**
If a local test fails, the script will display:
- Error message indicating which step failed
- Server stderr (last 10 lines from `logs/test-server.err`)
- Agent stderr (last 10 lines from `logs/test-agent.err`)
- Processes are automatically cleaned up
- Exit with code 1

### Manual Testing After Automated Test

**After Docker Test:**

If you use `-KeepContainers`/`--keep-containers`, the test containers remain running for manual inspection:

**View Container Logs:**
```bash
# Follow server logs
docker logs -f fluidity-server

# Follow agent logs
docker logs -f fluidity-agent
```

**Manual Proxy Test:**
```bash
# Test HTTP
curl -x http://127.0.0.1:8081 http://example.com

# Test HTTPS
curl -x http://127.0.0.1:8081 https://api.github.com

# Test with headers
curl -x http://127.0.0.1:8081 http://httpbin.org/get
```

**Inspect Containers:**
```bash
# Check container status
docker ps | grep fluidity

# Check network connectivity
docker exec fluidity-agent ping -c 3 fluidity-server

# Inspect TLS certificates
docker exec fluidity-server ls -la /root/certs/
```

**Cleanup When Done:**
```bash
docker rm -f fluidity-server fluidity-agent
```

**After Local Binary Test:**

Check the log files to see detailed output from the server and agent:

```bash
# View server logs
cat logs/test-server.out
tail -f logs/test-server.out  # Follow in real-time

# View agent logs
cat logs/test-agent.out
tail -f logs/test-agent.out

# View error logs
cat logs/test-server.err
cat logs/test-agent.err
```

The processes are automatically cleaned up when the test completes, but logs remain for inspection.

### Troubleshooting Test Failures

**Docker Test Issues:**

**Container Won't Start:**
- Check if certificates exist: `ls certs/` should show ca.crt, server.crt, server.key, client.crt, client.key
- Verify binaries exist: `ls build/` should show fluidity-server and fluidity-agent
- Check Docker is running: `docker ps`

**HTTP Test Returns 502 (Bad Gateway):**
- Agent can't reach server - check network: `docker network inspect fluidity-test-net`
- Check server logs: `docker logs fluidity-server --tail 20`
- Verify DNS resolution: `docker exec fluidity-agent nslookup fluidity-server`

**HTTP Test Returns 000 (No Response):**
- Tunnel not established - check agent logs: `docker logs fluidity-agent --tail 20`
- TLS handshake failing - verify certificate dates: `openssl x509 -in certs/server.crt -noout -dates`
- Port 8081 already in use - change port or stop conflicting service

**Build Failures:**
- Go not installed: `go version` should show Go 1.19+
- Missing dependencies: Run `go mod download` in project root
- Platform mismatch: Ensure `GOOS=linux GOARCH=amd64` for Docker builds

**Network/DNS Issues:**
- Verify test network exists: `docker network ls | grep fluidity-test-net`
- Check container names match config: Server must be named `fluidity-server` (matches `configs/agent.docker.yaml`)
- Recreate network: `docker network rm fluidity-test-net && docker network create fluidity-test-net`

**Local Binary Test Issues:**

**Binary Not Found:**
- Build the binaries: `make -f Makefile.win build` (or Makefile.macos/linux)
- Check build directory: `ls build/` should show fluidity-server[.exe] and fluidity-agent[.exe]
- Verify Go is installed: `go version`

**Port Already in Use:**
- Check what's using the port: `netstat -ano | findstr :8080` (Windows) or `lsof -i :8080` (macOS/Linux)
- Stop conflicting process or use custom ports: `-ProxyPort 8081 -ServerPort 8444`
- Kill existing fluidity processes: `taskkill /F /IM fluidity-*.exe` (Windows) or `pkill fluidity-` (macOS/Linux)

**Process Dies Immediately:**
- Check stderr logs: `cat logs/test-server.err` and `cat logs/test-agent.err`
- Verify configs exist: `ls configs/server.local.yaml configs/agent.local.yaml`
- Check certificates: `ls certs/` should show all required files
- Test binary manually: `./build/fluidity-server --help`

**TLS Handshake Failures:**
- Regenerate certificates: `./scripts/generate-certs.sh` or `.ps1`
- Check certificate validity: `openssl x509 -in certs/server.crt -noout -dates`
- Verify CA matches: Server and agent must use same CA
- Check file permissions: Certificates must be readable

**Proxy Connection Refused:**
- Agent not listening: Check `logs/test-agent.out` for startup messages
- Wrong port: Verify agent is listening on correct port (default 8080)
- Firewall blocking: Temporarily disable firewall to test
- Server not reachable: Check `logs/test-agent.err` for connection errors

**Build Failures:**
- Go not installed: `go version` should show Go 1.19+
- Missing dependencies: Run `go mod download` in project root
- Platform mismatch: Ensure you're using the correct Makefile for your OS

### Quick Manual Testing Reference

For quick manual validation without running the full automated test:

**1. Verify Certificates Are Generated:**
```bash
ls -la certs/
# Should show: ca.crt, ca.key, server.crt, server.key, client.crt, client.key
```

**2. Build and Check Binaries:**
```bash
# Linux/macOS
make -f Makefile.linux build  # or Makefile.macos
ls -lh build/
# Should show: fluidity-server, fluidity-agent (~20-25MB each)

# Windows
make -f Makefile.win build
dir build\
```

**3. Quick Docker Image Test:**
```bash
# Build images
docker build -t fluidity-server:test -f deployments/server/Dockerfile.local .
docker build -t fluidity-agent:test -f deployments/agent/Dockerfile.local .

# Check image sizes
docker images | grep fluidity
# Should show ~43MB each for alpine/curl-based images
```

**4. Quick Container Connectivity Test:**
```bash
# Create network
docker network create fluidity-net

# Start server
docker run -d --name fluidity-server --network fluidity-net \
  -v "$(pwd)/configs:/root/configs:ro" \
  -v "$(pwd)/certs:/root/certs:ro" \
  fluidity-server:test --config configs/server.docker.yaml

# Start agent
docker run -d --name fluidity-agent --network fluidity-net \
  -p 8080:8080 \
  -v "$(pwd)/configs:/root/configs:ro" \
  -v "$(pwd)/certs:/root/certs:ro" \
  fluidity-agent:test --config configs/agent.docker.yaml

# Wait and test
sleep 3
curl -x http://127.0.0.1:8080 http://example.com -I

# Cleanup
docker rm -f fluidity-server fluidity-agent
docker network rm fluidity-net
```

**5. Test Individual Components:**
```bash
# Test TLS certificates
openssl x509 -in certs/server.crt -noout -text | grep "Subject:"
openssl verify -CAfile certs/ca.crt certs/server.crt

# Test server binary directly
./build/fluidity-server --help

# Test agent binary directly
./build/fluidity-agent --help
```

**6. Test Proxy Without Tunnel (Local Development):**
```bash
# Run server locally (Terminal 1)
make -f Makefile.linux run-server-local  # or Makefile.macos/win

# Run agent locally (Terminal 2)
make -f Makefile.linux run-agent-local

# Test proxy (Terminal 3)
curl -x http://127.0.0.1:8080 http://httpbin.org/get
curl -x http://127.0.0.1:8080 https://www.google.com -I
```

### Testing Checklist

Before committing changes or deploying, verify:

**Prerequisites:**
- [ ] Certificates generated successfully (`./scripts/generate-certs.sh` or `.ps1`)
- [ ] Binaries compile for your platform (`make -f Makefile.* build`)
- [ ] Go dependencies downloaded (`go mod download`)

**Docker Tests:**
- [ ] Docker images build successfully (standard and scratch variants)
- [ ] Docker automated test passes (`./scripts/test-docker.sh` or `.ps1`)
- [ ] Containers start without errors
- [ ] Container logs show successful TLS handshake
- [ ] No error messages in container logs

**Local Binary Tests:**
- [ ] Local automated test passes (`./scripts/test-local.sh` or `.ps1`)
- [ ] Binaries start and run without crashes
- [ ] Server binds to port 8443 successfully
- [ ] Agent binds to port 8080 successfully
- [ ] Processes log to files correctly (`logs/test-*.out`)

**Functional Tests:**
- [ ] HTTP tunneling works (curl via proxy returns 200 OK)
- [ ] HTTPS tunneling works (CONNECT method establishes tunnel)
- [ ] Multiple concurrent requests handled correctly
- [ ] Proxy works with browser (Chrome/Firefox/Edge)
- [ ] Common websites load correctly (google.com, github.com, etc.)

**Cross-Platform Tests (if applicable):**
- [ ] Windows build and test passes
- [ ] macOS build and test passes
- [ ] Linux build and test passes

 

## License
 
