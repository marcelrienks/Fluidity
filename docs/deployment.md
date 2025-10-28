# Deployment Guide

This guide covers all supported deployment options for the Fluidity Tunnel Server and Agent: local development, Docker, and AWS Fargate (manual and CloudFormation). Use this as your single reference to get up and running in each environment.

---

## Prerequisites

- Go 1.21+ (for local builds)
- Docker Desktop (for container runs)
- Make (for platform-specific Makefiles)
- OpenSSL (for certificate generation)
- Node.js 18+ (for WebSocket tests)
- AWS account + AWS CLI v2 (for cloud deployment)

Certs: Generate client/server certs with scripts in `scripts/` (see Quick Start below).

---

## Docker Build Process

Fluidity uses a **simplified single-stage Docker build** that compiles Go binaries locally and copies them into lightweight Alpine containers (~44MB each).

For comprehensive Docker documentation including networking modes, troubleshooting, and advanced configurations, see **[docs/docker.md](docker.md)**.

### Why This Approach?

1. **Corporate Firewall Bypass**: Multi-stage Docker builds often fail in corporate environments where firewalls block Docker Hub or intercept HTTPS traffic. Building locally avoids these issues.
2. **Faster Builds**: ~2 seconds vs. 10+ seconds for multi-stage builds with Go module downloads.
3. **Platform Independence**: Works consistently across Windows, macOS, and Linux development environments.

### How It Works

```
1. Local Compilation → Static Linux binary (GOOS=linux GOARCH=amd64 CGO_ENABLED=0)
2. Docker COPY       → Binary copied into alpine/curl:latest container
3. Container Runtime → Minimal Alpine image with curl (~44MB total)
```

### Build Commands

All platform Makefiles (`Makefile.win`, `Makefile.macos`, `Makefile.linux`) ensure the Linux binary is built before the Docker image:

```bash
# Windows
make -f Makefile.win docker-build-server  # Builds Linux binary first, then Docker image
make -f Makefile.win docker-build-agent   # Same for agent

# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux
make docker-build-server
make docker-build-agent
```

### Dockerfile Structure

Both `deployments/server/Dockerfile` and `deployments/agent/Dockerfile` follow this simple pattern:

```dockerfile
FROM alpine/curl:latest
WORKDIR /app
COPY build/fluidity-server .  # Pre-built on host
RUN mkdir -p ./config ./certs
COPY configs/server.yaml ./config/
EXPOSE 8443
CMD ["./fluidity-server", "--config", "./config/server.yaml"]
```

**Key Benefits**:
- No Go installation in container
- No network calls during Docker build (all dependencies resolved locally)
- Reproducible builds across environments
- Small image size with essential utilities (curl for health checks)

---

## Option A — Local (Binaries)

Best for: development and quick iteration.

1) Generate certificates

```powershell
# Windows
./scripts/generate-certs.ps1
```

```bash
# macOS/Linux
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

2) Run server and agent in separate terminals

```powershell
# Windows
make -f Makefile.win run-server-local
make -f Makefile.win run-agent-local
```

```bash
# macOS
make -f Makefile.macos run-server-local
make -f Makefile.macos run-agent-local

# Linux
make -f Makefile.linux run-server-local
make -f Makefile.linux run-agent-local
```

3) Configure your browser to use HTTP/HTTPS proxy at `127.0.0.1:8080`

4) Test with curl

```powershell
# Windows (add --ssl-no-revoke to skip certificate revocation checks)
curl.exe -x http://127.0.0.1:8080 http://example.com -I
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

```bash
# macOS/Linux
curl -x http://127.0.0.1:8080 http://example.com -I
curl -x http://127.0.0.1:8080 https://example.com -I
```

---

## Option B — Docker (Local Containers)

**Best for**: Verifying container builds locally before cloud deployment.

**Note**: Docker containers work seamlessly on all platforms. The certificate generation scripts include `host.docker.internal` (for Windows/macOS Docker Desktop) in the server certificate's Subject Alternative Names, enabling local Docker testing out of the box.

For local development and quick iteration, **Option A (local binaries)** is still simpler as it avoids Docker overhead and starts instantly.

### Steps

1) Build images

```powershell
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent
```

```bash
# macOS/Linux
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
# or
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

2) Run containers (Windows PowerShell)

```powershell
# Server
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.windows-docker.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server

# Agent
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.windows-docker.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent
```

3) Run containers (macOS/Linux)

```bash
# Server
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.windows-docker.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server

# Agent
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.windows-docker.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent
```

**Config files used:**
- `server.windows-docker.yaml`: Binds to `0.0.0.0` (accessible from all network interfaces)
- `agent.windows-docker.yaml`: Connects to `host.docker.internal` (Docker Desktop hostname for host machine)

4) Test the tunnel

```powershell
# Windows - Test HTTP
curl.exe -x http://127.0.0.1:8080 http://example.com -I

# Windows - Test HTTPS
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

```bash
# macOS/Linux - Test HTTP
curl -x http://127.0.0.1:8080 http://example.com -I

# macOS/Linux - Test HTTPS
curl -x http://127.0.0.1:8080 https://example.com -I
```

You should see `HTTP/1.1 200 OK` responses, and both containers logging the traffic flow.

---

## Option C — AWS Fargate (ECS) — Manual

Best for: on-demand personal cloud use with minimal management.

**Note**: This option deploys just the server. For automated lifecycle management with Lambda control plane, see Option D.

Summary of steps (full details in `docs/fargate.md`):

1) Build, tag, and push the server image to ECR
2) Create CloudWatch Log Group, Security Group, ECS Cluster
3) Register a Fargate Task Definition (CPU=256, Memory=512, port 8443, awslogs)
4) Create an ECS Service with `desiredCount=0`, public subnets, `assignPublicIp=ENABLED`
5) Start on demand with `desiredCount=1`, wait ~60 seconds for cold start
6) Fetch the task public IP and run the Agent with `--server-ip <IP>`
7) Stop when done with `desiredCount=0`

Handy scripts (PowerShell and Bash) to start/stop and print the public IP are included in `docs/fargate.md`.

---

## Option D — AWS Fargate (ECS) — CloudFormation

Best for: repeatable, parameterized provisioning.

**Note**: This option deploys just the server infrastructure. For the complete Lambda control plane (Wake/Sleep/Kill), see Option E.

Use the template at `deployments/cloudformation/fargate.yaml`. It creates:
- ECS Cluster (Fargate), IAM execution role
- CloudWatch Log Group
- Security Group allowing inbound TCP on 8443 (or your port)
- ECS Task Definition + Service (desired count default 0)

1) Prepare `params.json` (example in `docs/fargate.md`):
- `ContainerImage` (ECR URI)
- `VpcId`, `PublicSubnets`
- Optional tuning: `DesiredCount`, `AllowedIngressCidr`, `Cpu`, `Memory`, `ContainerPort`

2) Deploy/Update

```powershell
$stackName = "fluidity-fargate"
aws cloudformation deploy `
  --template-file deployments/cloudformation/fargate.yaml `
  --stack-name $stackName `
  --parameter-overrides (Get-Content deployments/cloudformation/params.json | Out-String) `
  --capabilities CAPABILITY_NAMED_IAM
```

3) Start/Stop
- Update `DesiredCount` to 1 (start) or 0 (stop) via `aws cloudformation update-stack` (commands in `docs/fargate.md`)
- Or use `aws ecs update-service --desired-count ...`

4) Fetch Public IP
- Use the start script in `docs/fargate.md` to get the task's public IP and pass it to the Agent

---

## Option E — AWS Lambda Control Plane (Recommended for Production)

Best for: automated lifecycle management with cost optimization.

This option deploys the complete serverless control plane with Lambda functions for automated wake/sleep/kill operations, eliminating the need for manual ECS service management.

### Architecture Overview

```
Agent Startup → Wake Lambda → ECS DesiredCount=1 → Server Starts
                     ↓
               Agent Connects (retry for X seconds)

EventBridge (every 5 min) → Sleep Lambda → Check CloudWatch Metrics
                                                 ↓
                                          If idle → ECS DesiredCount=0

Agent Shutdown → Kill Lambda → ECS DesiredCount=0 (immediate)

EventBridge (daily 11 PM) → Kill Lambda → ECS DesiredCount=0
```

### Prerequisites

- Completed Option D (Fargate CloudFormation deployment)
- Python 3.11 installed (for Lambda functions)
- AWS SDK boto3 (automatically included in Lambda runtime)

### Components

1. **Wake Lambda**: Checks current ECS state, sets DesiredCount=1 if needed
2. **Sleep Lambda**: Queries CloudWatch metrics, scales down if idle
3. **Kill Lambda**: Immediately sets DesiredCount=0 (no validation)
4. **API Gateway**: HTTPS endpoints for Wake and Kill
5. **EventBridge Schedulers**: 
   - Periodic Sleep check (every X minutes)
   - Daily Kill at specific time

### Step 1: Deploy Lambda Infrastructure

```powershell
# Deploy Lambda control plane
aws cloudformation deploy `
  --template-file deployments/cloudformation/lambda.yaml `
  --stack-name fluidity-lambda `
  --parameter-overrides `
    ECSClusterName=fluidity `
    ECSServiceName=fluidity-server `
    IdleThresholdMinutes=15 `
    SleepCheckIntervalMinutes=5 `
    DailyKillTime="cron(0 23 * * ? *)" `
  --capabilities CAPABILITY_NAMED_IAM
```

### Step 2: Get API Gateway Endpoints

```powershell
# Get outputs
aws cloudformation describe-stacks `
  --stack-name fluidity-lambda `
  --query 'Stacks[0].Outputs'
```

Note the following outputs:
- `WakeAPIEndpoint`: HTTPS endpoint for wake
- `KillAPIEndpoint`: HTTPS endpoint for kill
- `APIKey`: API key for authentication

### Step 3: Update Agent Configuration

Add the Lambda control plane settings to your agent config:

```yaml
# configs/agent.local.yaml
server_ip: "<PUBLIC_IP>"  # Get from Fargate deployment
server_port: 8443
local_proxy_port: 8080
cert_file: "./certs/client.crt"
key_file: "./certs/client.key"
ca_cert_file: "./certs/ca.crt"
log_level: "info"

# Lambda control plane (NEW)
wake_api_endpoint: "https://xxxxxxxxxx.execute-api.us-east-1.amazonaws.com/prod/wake"
kill_api_endpoint: "https://xxxxxxxxxx.execute-api.us-east-1.amazonaws.com/prod/kill"
api_key: "your-api-key-here"
connection_timeout: "90s"           # Max time to wait after wake
connection_retry_interval: "5s"     # Retry interval
```

### Step 4: Update Server Configuration

Enable CloudWatch metrics emission:

```yaml
# configs/server.yaml (in Docker image or via env vars)
listen_addr: "0.0.0.0"
listen_port: 8443
cert_file: "/root/certs/server.crt"
key_file: "/root/certs/server.key"
ca_cert_file: "/root/certs/ca.crt"
log_level: "info"
max_connections: 100

# CloudWatch metrics (NEW)
emit_metrics: true
metrics_interval: "60s"  # Emit every 60 seconds
```

Rebuild and push the server Docker image:

```powershell
# Update configs/server.yaml with emit_metrics: true
make -f Makefile.win docker-build-server
docker tag fluidity-server:latest <ECR_URI>
docker push <ECR_URI>

# Update ECS task definition to use new image
aws ecs register-task-definition --cli-input-json file://updated-task-def.json
```

### Step 5: Test the Lifecycle

**Test Wake:**
```powershell
# Agent automatically calls wake on startup
./build/fluidity-agent --config ./configs/agent.local.yaml

# Monitor wake
aws logs tail /aws/lambda/fluidity-wake --follow
```

**Test Sleep (automatic):**
- Wait 15+ minutes with no traffic
- Sleep Lambda runs every 5 minutes
- Check logs:
  ```powershell
  aws logs tail /aws/lambda/fluidity-sleep --follow
  ```

**Test Kill (agent shutdown):**
```powershell
# Stop agent (Ctrl+C)
# Agent calls kill on shutdown

# Or trigger manually
curl -X POST "https://xxxxxxxxxx.execute-api.us-east-1.amazonaws.com/prod/kill" `
  -H "x-api-key: your-api-key-here"
```

**Test Daily Kill:**
- Scheduled automatically at configured time (default 11 PM UTC)
- Check EventBridge rules:
  ```powershell
  aws events list-rules --name-prefix fluidity
  ```

### Lambda Function Details

#### Wake Lambda
- Checks current ECS service state
- Sets `desiredCount=1` if service is stopped
- Returns idempotent response if already running
- Invoked by agent on startup via API Gateway

#### Sleep Lambda
- Queries CloudWatch for `ActiveConnections` and `LastActivityEpochSeconds` metrics
- Calculates idle duration based on last activity
- Sets `desiredCount=0` if idle threshold exceeded
- Invoked periodically by EventBridge scheduler

#### Kill Lambda
- Immediately sets `desiredCount=0` without validation
- Used for emergency shutdown or agent graceful termination
- Invoked by agent on shutdown via API Gateway

### IAM Permissions

The Lambda functions require:
- `ecs:DescribeServices`
- `ecs:UpdateService`
- `cloudwatch:GetMetricData` (Sleep Lambda only)

The server ECS task requires:
- `cloudwatch:PutMetricData`

### Cost Optimization

**Lambda costs**:
- Wake Lambda: ~1 invocation/day = negligible
- Sleep Lambda: ~288 invocations/day (every 5 min) = < $0.01/month
- Kill Lambda: ~2 invocations/day = negligible
- Total Lambda cost: < $0.05/month

**Combined costs** (Lambda + Fargate):
- 2 hours/day active: ~$0.50/month Fargate + $0.05 Lambda = **$0.55/month**
- 8 hours/day active: ~$3/month Fargate + $0.05 Lambda = **$3.05/month**
- Manual management (no Lambda): Same Fargate cost but requires manual start/stop

### Monitoring

**Lambda logs:**
```powershell
# Wake
aws logs tail /aws/lambda/fluidity-wake --follow

# Sleep
aws logs tail /aws/lambda/fluidity-sleep --follow

# Kill
aws logs tail /aws/lambda/fluidity-kill --follow
```

**CloudWatch metrics:**
```powershell
# View metrics
aws cloudwatch get-metric-statistics `
  --namespace Fluidity `
  --metric-name ActiveConnections `
  --dimensions Name=Service,Value=fluidity-server `
  --start-time 2025-11-01T00:00:00Z `
  --end-time 2025-11-01T23:59:59Z `
  --period 300 `
  --statistics Average
```

**ECS service status:**
```powershell
aws ecs describe-services `
  --cluster fluidity `
  --services fluidity-server `
  --query 'services[0].[desiredCount,runningCount,pendingCount]'
```

### Troubleshooting

**Agent can't wake server:**
- Check API Gateway endpoint URL
- Verify API key is correct
- Check Lambda execution logs
- Ensure Lambda has ECS permissions

**Sleep Lambda not scaling down:**
- Check CloudWatch metrics are being emitted (server config)
- Verify idle threshold is appropriate
- Check Lambda logs for metric query results

**Daily kill not working:**
- Verify EventBridge rule is enabled
- Check cron expression timezone (UTC)
- Review Lambda execution logs

---

## Security Tips

- Restrict Security Group ingress on 8443 to your current public IP (use `/32` CIDR)
- Keep mTLS enabled end-to-end; protect private keys and CA material
- Set CloudWatch Logs retention (e.g., 7–30 days)
- Prefer AWS Secrets Manager/SSM Parameter Store if you later externalize certs/configs

---

## Troubleshooting

### Local/Docker Testing

**Windows: `CRYPT_E_NO_REVOCATION_CHECK` error with curl**

If you see this error when testing HTTPS:
```
curl: (35) schannel: next InitializeSecurityContext failed: CRYPT_E_NO_REVOCATION_CHECK (0x80092012)
```

**Solution:** Add `--ssl-no-revoke` to your curl command:
```powershell
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

**Why:** Windows curl uses Schannel (native Windows SSL) which checks certificate revocation by default. This fails with self-signed certificates or when revocation servers are unreachable. The `--ssl-no-revoke` flag is safe for local testing.

---

**Docker: Port conflicts or connection issues**

If you encounter issues running Docker containers locally:

**Common Issues:**
- Port already in use: Another process is using port 8443 or 8080
- Cannot connect: Firewall blocking Docker network traffic

**Solutions:**

1. **Check port usage:**
```powershell
# Windows
netstat -ano | findstr :8443
netstat -ano | findstr :8080
```

2. **Stop conflicting processes or change ports:**
```powershell
# Edit configs/server.yaml or configs/agent.yaml to use different ports
```

3. **Verify Docker networking:**
```powershell
# Server should listen on 0.0.0.0 (all interfaces)
# Agent should connect to host.docker.internal (Windows/macOS)
```

**Note:** The certificate generation scripts now include `host.docker.internal` in the SAN list by default, so Windows/macOS Docker Desktop networking works out of the box.

---**Alternatives:**
- Use PowerShell: `Invoke-WebRequest -Uri https://example.com -Proxy http://127.0.0.1:8080 -Method Head`
- Use WSL: `wsl curl -x http://127.0.0.1:8080 https://example.com -I`
- Test with a browser (configure proxy to 127.0.0.1:8080)

### AWS Fargate

- Fargate task stuck in `PENDING`: check subnets are public, `assignPublicIp=ENABLED`, sufficient Fargate quota
- No logs: verify `awslogs` configuration and log group exists
- Connection refused: confirm SG allows port 8443 and container is listening
- TLS errors: confirm CA, server cert, key paths, and that Agent uses the correct `--server-ip`

---

## Costs (estimates)

- Fargate (0.25 vCPU, 0.5GB): ~$0.012/hour (≈ $0.50/month for 2h/day, ≈ $3/month for 8h/day)
- 24/7 Fargate: ≈ $9/month
- Additional minor charges: CloudWatch logs, ECR storage, data transfer

---

## References

- `docs/fargate.md` — Detailed Fargate and CloudFormation instructions
- `deployments/cloudformation/fargate.yaml` — CloudFormation template for ECS on Fargate
- `scripts/` — Certificate generation and test scripts
- `configs/` — Example configuration files
