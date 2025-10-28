# Docker Build and Deployment Guide

This document explains Fluidity's Docker build process, networking considerations, and best practices for containerized deployment.

---

## Table of Contents

- [Build Process](#build-process)
- [Image Architecture](#image-architecture)
- [Local Testing Limitations](#local-testing-limitations)
- [Production Deployment](#production-deployment)
- [Networking Modes](#networking-modes)
- [Troubleshooting](#troubleshooting)

---

## Build Process

### Overview

Fluidity uses a **simplified single-stage Docker build** that compiles Go binaries locally and copies them into minimal Alpine containers.

```
Host Machine (Any OS)    →    Docker Container (Linux)
─────────────────────         ────────────────────────
Go source code                Alpine Linux base
  ↓                             ↓
Static Linux binary          COPY binary
(GOOS=linux)                 Add curl utility
                             ~44MB total image
```

### Why This Approach?

#### 1. Corporate Firewall Bypass

**Problem**: Multi-stage Docker builds download Go modules and base images during build. Corporate firewalls often:
- Block Docker Hub (docker.io) with 403 Forbidden errors
- Intercept HTTPS traffic with their own certificates
- Prevent access to Go module proxies (proxy.golang.org)

**Solution**: Build everything locally before Docker starts:
- Go modules downloaded on host (using corporate proxy settings)
- Binary compiled with static linking (no dependencies)
- Docker only needs to COPY files (no network calls)

#### 2. Faster Builds

- **Multi-stage**: ~10+ seconds (download base image, Go modules, compile)
- **Single-stage**: ~2 seconds (copy pre-built binary to Alpine)

#### 3. Platform Independence

Works identically on:
- Windows (PowerShell)
- macOS (Bash/Zsh)
- Linux (Bash)

All build to the same Linux binary using cross-compilation.

### Build Commands

All Makefiles ensure the Linux binary is built before the Docker image:

```powershell
# Windows
make -f Makefile.win docker-build-server  # Depends on build-linux-server
make -f Makefile.win docker-build-agent   # Depends on build-linux-agent
```

```bash
# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux
make docker-build-server
make docker-build-agent
```

### Build Targets

**Example from Makefile.win:**
```makefile
build-linux-server:
	@echo Building Linux binary for server...
	$env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'; go build -o build/fluidity-server ./cmd/server

docker-build-server: build-linux-server
	@echo Building Docker image for server...
	docker build -f deployments/server/Dockerfile -t fluidity-server .
```

**Key flags:**
- `GOOS=linux`: Target Linux OS
- `GOARCH=amd64`: Target x86-64 architecture
- `CGO_ENABLED=0`: Static linking (no C dependencies)

---

## Image Architecture

### Base Image: alpine/curl:latest

We use `alpine/curl:latest` (~8MB) instead of plain `alpine` because:
- Includes curl for health checks and testing
- Still minimal (Alpine Linux is ~5MB, curl adds ~3MB)
- Official Alpine-based image with security updates

### Dockerfile Structure

**deployments/server/Dockerfile:**
```dockerfile
FROM alpine/curl:latest

WORKDIR /app

# Copy pre-built Linux binary
COPY build/fluidity-server .

# Create directories for volume mounts
RUN mkdir -p ./config ./certs

# Copy default config (can be overridden with volume mount)
COPY configs/server.yaml ./config/

# Expose tunnel port
EXPOSE 8443

# Run the server
CMD ["./fluidity-server", "--config", "./config/server.yaml"]
```

**deployments/agent/Dockerfile** follows the same pattern with:
- `build/fluidity-agent`
- `configs/agent.yaml`
- Exposes port 8080

### Image Sizes

```
REPOSITORY         TAG       SIZE
fluidity-server    latest    43.8MB
fluidity-agent     latest    43.9MB
```

Breakdown:
- Alpine base: ~5MB
- curl utility: ~3MB
- Go binary: ~35MB (includes HTTP/HTTPS/WebSocket/TLS stack)

---

## Local Testing Limitations

### Docker Desktop Networking

**Challenge**: When running both server and agent as Docker containers on the same machine, they need to communicate with each other.

**Windows/macOS Docker Desktop**:
- Containers run in a lightweight VM (Hyper-V or WSL2)
- To reach the host machine from a container, use `host.docker.internal`
- This is a special DNS name that resolves to the host's IP

**Good News**: The certificate generation scripts (`scripts/generate-certs.ps1` and `scripts/generate-certs.sh`) now include `host.docker.internal` in the server certificate's Subject Alternative Names (SAN) **by default**. This means Docker containers work out of the box on Windows and macOS!

**Certificate SANs (included by default)**:
```
DNS.1 = fluidity-server
DNS.2 = localhost
DNS.3 = host.docker.internal  ✅ (Added for Docker Desktop support)
IP.1 = 127.0.0.1
IP.2 = ::1
```

**Verify your certificates include this:**
```powershell
# Windows
openssl x509 -in .\certs\server.crt -noout -text | Select-String -Pattern "DNS:"

# macOS/Linux
openssl x509 -in ./certs/server.crt -noout -text | grep DNS:
```

**If you regenerated certificates before this update**, simply run the generation script again:
```powershell
# Windows
.\scripts\generate-certs.ps1

# macOS/Linux
./scripts/generate-certs.sh
```

### Recommended Testing Approach

#### Use Windows Docker Configs (Now Works Out of the Box!)

With the updated certificates, you can now test Docker containers locally on Windows/macOS without any issues:

```powershell
# Terminal 1 - Server (Windows)
docker run --rm -v "${PWD}/certs:/root/certs:ro" -v "${PWD}/configs/server.windows-docker.yaml:/root/config/server.yaml:ro" -p 8443:8443 fluidity-server

# Terminal 2 - Agent (Windows)
docker run --rm -v "${PWD}/certs:/root/certs:ro" -v "${PWD}/configs/agent.windows-docker.yaml:/root/config/agent.yaml:ro" -p 8080:8080 fluidity-agent
```

```bash
# macOS (same approach)
docker run --rm -v "$(pwd)/certs:/root/certs:ro" -v "$(pwd)/configs/server.windows-docker.yaml:/root/config/server.yaml:ro" -p 8443:8443 fluidity-server

docker run --rm -v "$(pwd)/certs:/root/certs:ro" -v "$(pwd)/configs/agent.windows-docker.yaml:/root/config/agent.yaml:ro" -p 8080:8080 fluidity-agent
```

**What these configs do:**
- `server.windows-docker.yaml`: Binds to `0.0.0.0` (accessible from containers)
- `agent.windows-docker.yaml`: Connects to `host.docker.internal` (reaches host machine)

**Volume mounts:**
- `/root/certs` - TLS certificates (matches Dockerfile WORKDIR /root/)
- `/root/config` - Configuration files (relative paths in YAML resolve correctly)

#### Alternative: Use Local Binaries

For even faster iteration during development (instant startup, no container overhead):

```powershell
# Terminal 1
make -f Makefile.win run-server-local

# Terminal 2
make -f Makefile.win run-agent-local
```

**Benefits**:
- No Docker networking complexity
- Faster startup (~instant vs. container overhead)
- Easier debugging (direct logs, no container inspection)
- Lower resource usage

#### Alternative: Custom Docker Network

If you prefer containers to communicate directly (without reaching through the host):

```powershell
# Create network
docker network create fluidity-net

# Run server (named "fluidity-server")
docker run --rm --name fluidity-server --network fluidity-net -v "${PWD}/certs:/root/certs:ro" -v "${PWD}/configs/server.yaml:/root/config/server.yaml:ro" -p 8443:8443 fluidity-server

# Run agent (connects to "fluidity-server")
docker run --rm --network fluidity-net -v "${PWD}/certs:/root/certs:ro" -v "${PWD}/configs/agent.yaml:/root/config/agent.yaml:ro" -p 8080:8080 fluidity-agent
```

**Note**: This works because the certificate includes `fluidity-server` in the SAN list, and Docker DNS resolves the container name to the correct IP.

### Testing Docker Containers

Once both containers are running, test end-to-end functionality:

**Test HTTP:**
```powershell
# Windows
curl.exe -x http://127.0.0.1:8080 http://example.com -I

# macOS/Linux
curl -x http://127.0.0.1:8080 http://example.com -I
```

**Expected output:**
```
HTTP/1.1 200 OK
Content-Type: text/html
...
```

**Test HTTPS:**
```powershell
# Windows
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke

# macOS/Linux
curl -x http://127.0.0.1:8080 https://example.com -I
```

**Expected output:**
```
HTTP/1.1 200 Connection Established

HTTP/1.1 200 OK
Content-Type: text/html
...
```

**Check container logs:**
```powershell
# View server logs
docker logs <server-container-id>

# View agent logs
docker logs <agent-container-id>
```

You should see debug logs showing:
- Agent: `"Connected to tunnel server","addr":"host.docker.internal:8443"`
- Server: `"CONNECT open request","address":"example.com:443"`
- Both: Data transfer logs showing bytes sent/received

---

## Production Deployment

### AWS Fargate / ECS

In cloud environments, Docker networking is straightforward:

**Why it works**:
1. Containers use AWS VPC networking with private IPs
2. ECS service discovery provides DNS names
3. Certificates can include ECS service names in SAN list
4. No `host.docker.internal` needed

**Example**:
```yaml
# ECS Task Definition
ServerDNS: fluidity-server.local
Certificate SAN: fluidity-server.local
Agent connects to: fluidity-server.local:8443 ✅
```

**CloudFormation Template**: See `deployments/cloudformation/fargate.yaml` for complete setup.

### Configuration for CloudWatch Metrics

When deploying to AWS, the server can emit metrics to CloudWatch for monitoring and automated lifecycle management (see Lambda control plane in `docs/deployment.md` Option E).

**Server Configuration (`server.yaml`):**
```yaml
server:
  host: "0.0.0.0"
  port: 8443
  
tls:
  cert_file: "/certs/server.crt"
  key_file: "/certs/server.key"
  ca_file: "/certs/ca.crt"
  
metrics:
  emit_metrics: true              # Enable CloudWatch metrics
  metrics_interval: 60s           # Emit every 60 seconds
  namespace: "Fluidity"           # CloudWatch namespace
  service_name: "fluidity-server" # Service dimension
```

**Environment Variables (Alternative):**
```bash
# Can override config via environment variables
FLUIDITY_METRICS_ENABLED=true
FLUIDITY_METRICS_INTERVAL=60s
FLUIDITY_NAMESPACE=Fluidity
FLUIDITY_SERVICE_NAME=fluidity-server
```

**Metrics Emitted:**
- `ActiveConnections` (Gauge) - Current number of connected agents
- `LastActivityEpochSeconds` (Timestamp) - Unix timestamp of last tunnel activity

**IAM Permissions Required:**
- `cloudwatch:PutMetricData` on namespace `Fluidity`

**Viewing Metrics:**
```bash
# Check active connections
aws cloudwatch get-metric-statistics \
  --namespace Fluidity \
  --metric-name ActiveConnections \
  --dimensions Name=ServiceName,Value=fluidity-server \
  --statistics Maximum \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300
```

For automated lifecycle management using these metrics, see `docs/deployment.md` Option E (Lambda control plane).

### Best Practices

1. **Secrets Management**: Use AWS Secrets Manager or SSM Parameter Store for certificates and keys (not baked into image)
2. **Health Checks**: ECS can use curl to check `https://localhost:8443/health` (add health endpoint if needed)
3. **Logging**: CloudWatch Logs integration (already in CloudFormation template)
4. **Security Groups**: Restrict port 8443 to specific IP ranges
5. **Cost Optimization**: Set `DesiredCount=0` when not in use (~$0.012/hour when running), or use Lambda control plane for automatic idle shutdown
6. **Metrics**: Enable CloudWatch metrics for monitoring and automated lifecycle decisions

---

## Networking Modes

### Host Network Mode

```powershell
docker run --network host fluidity-server
```

**Use case**: Server runs with host's network stack (no port mapping needed)
**Limitation**: Not available on Windows Docker Desktop (Linux only)

### Bridge Network Mode (Default)

```powershell
docker run -p 8443:8443 fluidity-server
```

**Use case**: Isolated network with port forwarding
**Limitation**: Container-to-container communication requires special DNS names

### Custom Bridge Network

```powershell
docker network create fluidity-net
docker run --network fluidity-net --name fluidity-server fluidity-server
docker run --network fluidity-net fluidity-agent
```

**Use case**: Containers communicate by name with automatic DNS
**Benefit**: Works with existing certificates (if `fluidity-server` is in SAN)

---

## Troubleshooting

### "403 Forbidden" during Docker build

**Symptom**:
```
ERROR [internal] load metadata for docker.io/library/golang:1.21-alpine
> failed to solve with frontend dockerfile.v0: failed to create LLB definition: unexpected status from GET request to https://registry-1.docker.io/v2/library/golang/manifests/1.21-alpine: 403 Forbidden
```

**Cause**: Corporate firewall blocking Docker Hub

**Solution**: Use the simplified build process (already implemented):
```powershell
make -f Makefile.win docker-build-server  # Builds locally first
```

### "UNAUTHORIZED: authentication required"

**Symptom**:
```
denied: requested access to the resource is denied
unauthorized: authentication required
```

**Cause**: Trying to pull from private registry without authentication

**Solution**: Use `alpine/curl:latest` (public image) or authenticate:
```powershell
docker login
```

### TLS certificate verification failed

**Symptom**:
```
tls: failed to verify certificate: x509: certificate is valid for fluidity-server, localhost, not host.docker.internal
```

**Cause**: Your certificates were generated before `host.docker.internal` was added to the default SAN list.

**Solution**: Regenerate certificates with the updated script:
```powershell
# Windows
.\scripts\generate-certs.ps1

# macOS/Linux
./scripts/generate-certs.sh
```

**Verify the fix:**
```powershell
# Windows
openssl x509 -in .\certs\server.crt -noout -text | Select-String -Pattern "DNS:"

# macOS/Linux
openssl x509 -in ./certs/server.crt -noout -text | grep DNS:

# Should show: DNS:fluidity-server, DNS:localhost, DNS:host.docker.internal
```

### Container starts but immediately exits

**Debug**:
```powershell
# Check logs
docker logs <container-id>

# Run with interactive shell
docker run -it --entrypoint /bin/sh fluidity-server

# Inside container:
ls -la  # Check binary exists
./fluidity-server --help  # Test binary
cat ./config/server.yaml  # Check config
```

### Permission denied on binary

**Symptom**:
```
/bin/sh: ./fluidity-server: Permission denied
```

**Cause**: Binary not executable (rare with COPY, but possible)

**Solution**: Add to Dockerfile after COPY:
```dockerfile
RUN chmod +x ./fluidity-server
```

### Volume mount issues (Windows)

**Symptom**:
```
Error response from daemon: invalid mount config for type "bind": bind source path does not exist
```

**Solution**: Use `${PWD}` with forward slashes or absolute paths:
```powershell
docker run -v "${PWD}/certs:/app/certs:ro" fluidity-server
```

---

## Summary

**For Local Development**:
- Use local binaries (`make run-server-local`, `make run-agent-local`)
- Faster, simpler, no Docker networking complexity

**For Cloud Deployment**:
- Docker images are production-ready (~44MB, Alpine-based)
- Build process bypasses corporate firewalls
- Works seamlessly with AWS Fargate/ECS
- Use CloudFormation template for infrastructure-as-code

**Key Insight**: Docker's power is in production deployment portability, not local development convenience. Use the right tool for the right job.
