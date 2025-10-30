# Deployment Guide

Quick reference for all Fluidity deployment options.

---

## Prerequisites

- **Go 1.21+** (for local builds)
- **Docker Desktop** (for containers)
- **OpenSSL** (for certificates)
- **Node.js 18+** (for WebSocket tests)
- **AWS CLI v2** (for cloud deployment)

**Generate certificates first** (required for all options):

```bash
# macOS/Linux
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

```powershell
# Windows
./scripts/generate-certs.ps1
```

---

## Deployment Options

### Option A: Local Development (Recommended for Development)

Run binaries directly on your machine.

**Start server and agent:**

```bash
# macOS/Linux
make -f Makefile.macos run-server-local  # or Makefile.linux
make -f Makefile.macos run-agent-local
```

```powershell
# Windows
make -f Makefile.win run-server-local
make -f Makefile.win run-agent-local
```

**Configure browser proxy:** `127.0.0.1:8080`

**Test:**

```bash
# macOS/Linux
curl -x http://127.0.0.1:8080 http://example.com -I
curl -x http://127.0.0.1:8080 https://example.com -I
```

```powershell
# Windows
curl.exe -x http://127.0.0.1:8080 http://example.com -I
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

**Why Option A:**
- Instant startup (no Docker overhead)
- Easy debugging
- Best for rapid iteration

---

### Option B: Docker (Local Containers)

Verify containerized deployment locally before cloud.

**Build images:**

```bash
# macOS/Linux
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
```

```powershell
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent
```

**Run containers:**

```bash
# macOS/Linux - Server
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.docker.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server

# Agent
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.docker.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent
```

```powershell
# Windows - Server
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.docker.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server

# Agent
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.docker.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent
```

**Note:** Configs use `host.docker.internal` for agent-to-server communication. See **[Docker Guide](docker.md)** for networking details and troubleshooting.

---

### Option C: AWS Fargate (Manual Management)

Deploy server to cloud, manage lifecycle manually.

**Summary:**
1. Build and push server image to ECR
2. Create ECS cluster, task definition, and service
3. Start with `desiredCount=1`, get public IP
4. Run local agent with `--server-ip <IP>`
5. Stop with `desiredCount=0` when done

**Full instructions:** See **[Fargate Guide](fargate.md)**

**Costs:** ~$0.012/hour (~$0.50/month for 2h/day, ~$3/month for 8h/day)

---

### Option D: AWS Fargate (CloudFormation)

Repeatable infrastructure deployment with parameterized template.

**Deploy:**

```powershell
aws cloudformation deploy \
  --template-file deployments/cloudformation/fargate.yaml \
  --stack-name fluidity-fargate \
  --parameter-overrides file://deployments/cloudformation/params.json \
  --capabilities CAPABILITY_NAMED_IAM
```

**Manage:**

```powershell
# Start
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 1

# Stop
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 0
```

**Full instructions:** See **[Fargate Guide](fargate.md)**

---

### Option E: Lambda Control Plane (Recommended for Production)

Automated lifecycle with cost optimization.

**Architecture:**
- **Wake Lambda**: Agent calls on startup → ECS DesiredCount=1
- **Sleep Lambda**: EventBridge scheduler → Check CloudWatch metrics → Scale down if idle
- **Kill Lambda**: Agent calls on shutdown OR daily scheduled shutdown

**Prerequisites:**
- Option D deployed (Fargate infrastructure)
- Python 3.11 (included in Lambda runtime)

**Deploy:**

```powershell
aws cloudformation deploy \
  --template-file deployments/cloudformation/lambda.yaml \
  --stack-name fluidity-lambda \
  --parameter-overrides \
    ECSClusterName=fluidity \
    ECSServiceName=fluidity-server \
    IdleThresholdMinutes=15 \
    SleepCheckIntervalMinutes=5 \
    DailyKillTime="cron(0 23 * * ? *)" \
  --capabilities CAPABILITY_NAMED_IAM
```

**Configure agent:**

```yaml
# configs/agent.local.yaml
wake_api_endpoint: "https://xxx.execute-api.us-east-1.amazonaws.com/prod/wake"
kill_api_endpoint: "https://xxx.execute-api.us-east-1.amazonaws.com/prod/kill"
api_key: "your-api-key"
connection_timeout: "90s"
connection_retry_interval: "5s"
```

**Enable server metrics:**

```yaml
# configs/server.yaml (rebuild Docker image after changes)
emit_metrics: true
metrics_interval: "60s"
```

**Costs:** Lambda < $0.05/month + Fargate usage

**Full instructions:** See **[Lambda Functions Guide](lambda.md)**

---

## Quick Start by Use Case

| Use Case | Recommended Option | Why |
|----------|-------------------|-----|
| **Local development** | A: Local binaries | Fastest iteration, no overhead |
| **Testing containerization** | B: Docker | Verify images before cloud deployment |
| **Personal cloud use (manual)** | C: Fargate manual | Simple on-demand cloud access |
| **Repeatable infrastructure** | D: Fargate CloudFormation | Parameterized, version-controlled |
| **Production with cost optimization** | E: Lambda control plane | Automated lifecycle, minimal cost |

---

## Common Issues

### Windows: Certificate Revocation Error

**Error:** `CRYPT_E_NO_REVOCATION_CHECK`

**Solution:** Add `--ssl-no-revoke` flag:
```powershell
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

### Docker: Port Conflicts

**Solution:** Check port usage:
```powershell
# Windows
netstat -ano | findstr :8443
netstat -ano | findstr :8080
```

Change ports in `configs/server.yaml` or `configs/agent.yaml` if needed.

### Fargate: Task Stuck in PENDING

**Common causes:**
- Subnets not public
- `assignPublicIp` not enabled
- Insufficient Fargate quota

**Solution:** See **[Fargate Guide](fargate.md)** troubleshooting section.

---

## Security Best Practices

1. **Restrict Security Group:** Limit port 8443 ingress to your IP (`/32` CIDR)
2. **Protect certificates:** Secure private keys and CA material
3. **Enable logging retention:** Set CloudWatch Logs retention (7-30 days)
4. **Use secrets management:** Consider AWS Secrets Manager for production

---

## Related Documentation

- **[Docker Guide](docker.md)** - Container networking, build process, troubleshooting
- **[Fargate Guide](fargate.md)** - Detailed AWS ECS deployment steps
- **[Lambda Functions](lambda.md)** - Control plane architecture and configuration
- **[Architecture](architecture.md)** - System design and components
