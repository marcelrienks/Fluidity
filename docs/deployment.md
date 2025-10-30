# Deployment Guide# Deployment Guide



## PrerequisitesQuick reference for all Fluidity deployment options.



1. Generate certificates (required):---

   ```bash

   ./scripts/manage-certs.sh              # Linux/macOS## Prerequisites

   .\scripts\manage-certs.ps1             # Windows

   ```- **Go 1.21+** (for local builds)

- **Docker Desktop** (for containers)

2. Install dependencies:- **OpenSSL** (for certificates)

   - Go 1.21+ (for local builds)- **Node.js 18+** (for WebSocket tests)

   - Docker Desktop (for containers)- **AWS CLI v2** (for cloud deployment)

   - AWS CLI v2 (for cloud deployment)

**Generate certificates first** (required for all options):

## Deployment Options

```bash

### Local Development# macOS/Linux

chmod +x scripts/generate-certs.sh

**Start server and agent:**./scripts/generate-certs.sh

```bash```

make -f Makefile.<platform> run-server-local  # Terminal 1

make -f Makefile.<platform> run-agent-local   # Terminal 2```powershell

# platform: win, macos, or linux# Windows

```./scripts/generate-certs.ps1

```

**Test:**

```bash---

curl -x http://127.0.0.1:8080 http://example.com

```## Deployment Options



**Best for:** Rapid development and debugging### Option A: Local Development (Recommended for Development)



---Run binaries directly on your machine.



### Docker (Local)**Start server and agent:**



**Build:**```bash

```bash# macOS/Linux

make -f Makefile.<platform> docker-build-servermake -f Makefile.macos run-server-local  # or Makefile.linux

make -f Makefile.<platform> docker-build-agentmake -f Makefile.macos run-agent-local

``````



**Run:**```powershell

```bash# Windows

# Servermake -f Makefile.win run-server-local

docker run --rm \make -f Makefile.win run-agent-local

  -v "$(pwd)/certs:/root/certs:ro" \```

  -v "$(pwd)/configs/server.docker.yaml:/root/config/server.yaml:ro" \

  -p 8443:8443 \**Configure browser proxy:** `127.0.0.1:8080`

  fluidity-server

**Test:**

# Agent

docker run --rm \```bash

  -v "$(pwd)/certs:/root/certs:ro" \# macOS/Linux

  -v "$(pwd)/configs/agent.docker.yaml:/root/config/agent.yaml:ro" \curl -x http://127.0.0.1:8080 http://example.com -I

  -p 8080:8080 \curl -x http://127.0.0.1:8080 https://example.com -I

  fluidity-agent```

```

```powershell

**Windows PowerShell:** Use `${PWD}` instead of `$(pwd)` and backticks for line continuation.# Windows

curl.exe -x http://127.0.0.1:8080 http://example.com -I

**Best for:** Testing containerized deployment before cloudcurl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke

```

---

**Why Option A:**

### AWS Fargate (Manual)- Instant startup (no Docker overhead)

- Easy debugging

**Steps:**- Best for rapid iteration

1. Build and push to ECR

2. Create ECS cluster, task definition, service---

3. Start: `desiredCount=1`

4. Get public IP from task### Option B: Docker (Local Containers)

5. Run local agent with server IP

6. Stop: `desiredCount=0`Verify containerized deployment locally before cloud.



**Cost:** ~$0.012/hour**Build images:**



**Full guide:** [Fargate Guide](fargate.md)```bash

# macOS/Linux

---make -f Makefile.macos docker-build-server

make -f Makefile.macos docker-build-agent

### AWS CloudFormation```



**Deploy infrastructure:**```powershell

```bash# Windows

cd scriptsmake -f Makefile.win docker-build-server

./deploy-fluidity.sh -e prod -a deploy  # Linux/macOSmake -f Makefile.win docker-build-agent

.\deploy-fluidity.ps1 -Environment prod -Action deploy  # Windows```

```

**Run containers:**

**Start/stop:**

```bash```bash

aws ecs update-service \# macOS/Linux - Server

  --cluster fluidity-prod \docker run --rm \

  --service fluidity-server-prod \  -v "$(pwd)/certs:/root/certs:ro" \

  --desired-count 1  # or 0 to stop  -v "$(pwd)/configs/server.docker.yaml:/root/config/server.yaml:ro" \

```  -p 8443:8443 \

  fluidity-server

**Best for:** Repeatable, version-controlled infrastructure

# Agent

**Full guide:** [Infrastructure Guide](infrastructure.md)docker run --rm \

  -v "$(pwd)/certs:/root/certs:ro" \

---  -v "$(pwd)/configs/agent.docker.yaml:/root/config/agent.yaml:ro" \

  -p 8080:8080 \

### Lambda Control Plane (Recommended)  fluidity-agent

```

**Architecture:**

- **Wake**: Agent calls on startup → starts server```powershell

- **Sleep**: Auto-scales down when idle (EventBridge)# Windows - Server

- **Kill**: Agent calls on shutdown or daily scheduledocker run --rm `

  -v ${PWD}\certs:/root/certs:ro `

**Deploy:**  -v ${PWD}\configs\server.docker.yaml:/root/config/server.yaml:ro `

```bash  -p 8443:8443 `

aws cloudformation deploy \  fluidity-server

  --template-file deployments/cloudformation/lambda.yaml \

  --stack-name fluidity-lambda \# Agent

  --parameter-overrides \docker run --rm `

    ECSClusterName=fluidity \  -v ${PWD}\certs:/root/certs:ro `

    ECSServiceName=fluidity-server \  -v ${PWD}\configs\agent.docker.yaml:/root/config/agent.yaml:ro `

  --capabilities CAPABILITY_NAMED_IAM  -p 8080:8080 `

```  fluidity-agent

```

**Configure agent:**

```yaml**Note:** Configs use `host.docker.internal` for agent-to-server communication. See **[Docker Guide](docker.md)** for networking details and troubleshooting.

# configs/agent.yaml

wake_api_endpoint: "https://xxx.execute-api.region.amazonaws.com/prod/wake"---

kill_api_endpoint: "https://xxx.execute-api.region.amazonaws.com/prod/kill"

api_key: "your-api-key"### Option C: AWS Fargate (Manual Management)

connection_timeout: "90s"

```Deploy server to cloud, manage lifecycle manually.



**Enable server metrics:****Summary:**

```yaml1. Build and push server image to ECR

# configs/server.yaml (rebuild image after changes)2. Create ECS cluster, task definition, and service

emit_metrics: true3. Start with `desiredCount=1`, get public IP

metrics_interval: "60s"4. Run local agent with `--server-ip <IP>`

```5. Stop with `desiredCount=0` when done



**Cost:** Lambda <$0.05/month + Fargate usage**Full instructions:** See **[Fargate Guide](fargate.md)**



**Best for:** Production with automated lifecycle and cost optimization**Costs:** ~$0.012/hour (~$0.50/month for 2h/day, ~$3/month for 8h/day)



**Full guide:** [Lambda Functions](lambda.md)---



---### Option D: AWS Fargate (CloudFormation)



## Quick ReferenceRepeatable infrastructure deployment with parameterized template.



| Use Case | Option | Cost |**Deploy:**

|----------|--------|------|

| Development | Local binaries | Free |```powershell

| Testing containers | Docker | Free |aws cloudformation deploy \

| Cloud (manual) | Fargate | ~$0.50-3/month |  --template-file deployments/cloudformation/fargate.yaml \

| Production | Lambda + Fargate | ~$0.05-1/month |  --stack-name fluidity-fargate \

  --parameter-overrides file://deployments/cloudformation/params.json \

## Troubleshooting  --capabilities CAPABILITY_NAMED_IAM

```

**Windows certificate error:**

```powershell**Manage:**

curl.exe -x http://127.0.0.1:8080 https://example.com --ssl-no-revoke

``````powershell

# Start

**Port conflicts:**aws ecs update-service \

```bash  --cluster fluidity \

netstat -ano | findstr :8443  --service fluidity-server \

netstat -ano | findstr :8080  --desired-count 1

```

# Stop

**Fargate task stuck:** Check subnets are public and `assignPublicIp=ENABLED`aws ecs update-service \

  --cluster fluidity \

## Security  --service fluidity-server \

  --desired-count 0

1. Restrict Security Group to your IP (`/32`)```

2. Secure certificate private keys

3. Enable CloudWatch Logs retention**Full instructions:** See **[Fargate Guide](fargate.md)**

4. Use AWS Secrets Manager for production

---

## Related Documentation

### Option E: Lambda Control Plane (Recommended for Production)

- [Docker Guide](docker.md) - Container details

- [Fargate Guide](fargate.md) - AWS ECS deploymentAutomated lifecycle with cost optimization.

- [Infrastructure Guide](infrastructure.md) - CloudFormation

- [Lambda Functions](lambda.md) - Control plane**Architecture:**

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
