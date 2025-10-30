# Deployment Guide

Complete deployment guide for all Fluidity deployment options.

---

## Prerequisites

Before deploying, ensure you have:

- **Go 1.21+** (for local builds)
- **Docker Desktop** (for containers)
- **OpenSSL** (for certificate generation)
- **AWS CLI v2** (for cloud deployment)
- **jq** (for JSON parsing in scripts)

---

## Certificate Generation (Required First Step)

**All deployment options require certificates to be generated first.**

```bash
# macOS/Linux
./scripts/manage-certs.sh

# Windows PowerShell
.\scripts\manage-certs.ps1
```

This creates certificates in `./certs/`:
- `ca.crt`, `ca.key` - Certificate Authority
- `server.crt`, `server.key` - Server certificate  
- `client.crt`, `client.key` - Client certificate

**Important:** Keep these files secure. The agent uses client certificates, and cloud deployments read server certificates for upload to AWS.

---

## Deployment Options

### Option A: Local Development (Recommended for Development)

Run server and agent binaries directly on your machine.

**1. Generate certificates** (if not done already):
```bash
./scripts/manage-certs.sh
```

**2. Start server** (Terminal 1):
```bash
# macOS
make -f Makefile.macos run-server-local

# Linux
make -f Makefile.linux run-server-local

# Windows
make -f Makefile.win run-server-local
```

**3. Start agent** (Terminal 2):
```bash
# macOS
make -f Makefile.macos run-agent-local

# Linux  
make -f Makefile.linux run-agent-local

# Windows
make -f Makefile.win run-agent-local
```

**4. Configure browser proxy:** `127.0.0.1:8080`

**5. Test:**
```bash
# macOS/Linux
curl -x http://127.0.0.1:8080 http://example.com -I
curl -x http://127.0.0.1:8080 https://example.com -I

# Windows
curl.exe -x http://127.0.0.1:8080 http://example.com -I
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke
```

**Why use this option:**
- Fastest iteration cycle
- No container overhead
- Easy debugging
- Best for development

**Cost:** Free

---

### Option B: Docker (Local Containers)

Test containerized deployment locally before cloud deployment.

**1. Generate certificates** (if not done already):
```bash
./scripts/manage-certs.sh
```

**2. Build images:**
```bash
# macOS
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent

# Linux
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent

# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent
```

**3. Run server:**
```bash
# macOS/Linux
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.docker.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server
```

```powershell
# Windows PowerShell
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.docker.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server
```

**4. Run agent** (new terminal):
```bash
# macOS/Linux
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.docker.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent
```

```powershell
# Windows PowerShell
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.docker.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent
```

**5. Test** (same as Option A)

**Why use this option:**
- Verify containers work before cloud deployment
- Test Docker configurations locally
- Validate image builds

**Cost:** Free

**See also:** [Docker Guide](docker.md) for detailed container documentation

---

### Option C: AWS Fargate with CloudFormation (Recommended for Production)

Deploy server to AWS using Infrastructure as Code with automated certificate management.

#### Step 1: Generate Certificates

```bash
# macOS/Linux
./scripts/manage-certs.sh

# Windows PowerShell
.\scripts\manage-certs.ps1
```

This creates certificates in `./certs/` directory. **These files are required** - the deployment script reads them and passes them to CloudFormation as parameters.

#### Step 2: Build and Push Server Image to ECR

**Create ECR repository** (one-time):
```bash
aws ecr create-repository \
  --repository-name fluidity-server \
  --region us-east-1
```

**Build and push:**
```bash
# macOS
make -f Makefile.macos push-server

# Linux
make -f Makefile.linux push-server

# Windows
make -f Makefile.win push-server
```

Or manually:
```bash
# Get ECR login
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Build and tag
docker build -f deployments/server/Dockerfile -t fluidity-server .
docker tag fluidity-server:latest <ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest

# Push
docker push <ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest
```

#### Step 3: Configure Parameters

Edit `deployments/cloudformation/params.json`:

```json
[
  {
    "ParameterKey": "ContainerImage",
    "ParameterValue": "<ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest"
  },
  {
    "ParameterKey": "VpcId",
    "ParameterValue": "vpc-xxxxx"
  },
  {
    "ParameterKey": "PublicSubnets",
    "ParameterValue": "subnet-xxxxx,subnet-yyyyy"
  },
  {
    "ParameterKey": "AllowedIngressCidr",
    "ParameterValue": "0.0.0.0/0"
  }
]
```

**Note:** Certificate parameters (`CertPem`, `KeyPem`, `CaPem`) are automatically added by the deployment script.

#### Step 4: Deploy CloudFormation Stack

**Using deployment script** (recommended):
```bash
./scripts/deploy-fluidity.sh -e prod -a deploy
```

The script will:
1. Check that certificates exist in `./certs/`
2. Base64-encode the certificates
3. Pass them as parameters to CloudFormation
4. Deploy the stack with Secrets Manager secret created

**Or deploy manually:**
```bash
aws cloudformation create-stack \
  --stack-name fluidity-fargate \
  --template-body file://deployments/cloudformation/fargate.yaml \
  --parameters file://deployments/cloudformation/params.json \
    ParameterKey=CertPem,ParameterValue=$(base64 -i ./certs/server.crt | tr -d '\n') \
    ParameterKey=KeyPem,ParameterValue=$(base64 -i ./certs/server.key | tr -d '\n') \
    ParameterKey=CaPem,ParameterValue=$(base64 -i ./certs/ca.crt | tr -d '\n') \
  --capabilities CAPABILITY_NAMED_IAM
```

#### Step 5: Start the Server

```bash
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 1
```

Wait ~60 seconds for task to start.

#### Step 6: Get Server Public IP

```bash
aws ecs list-tasks \
  --cluster fluidity \
  --service-name fluidity-server \
  --query 'taskArns[0]' \
  --output text | xargs -I {} \
aws ecs describe-tasks \
  --cluster fluidity \
  --tasks {} \
  --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' \
  --output text | xargs -I {} \
aws ec2 describe-network-interfaces \
  --network-interface-ids {} \
  --query 'NetworkInterfaces[0].Association.PublicIp' \
  --output text
```

Or check CloudFormation outputs:
```bash
aws cloudformation describe-stacks \
  --stack-name fluidity-fargate \
  --query 'Stacks[0].Outputs'
```

#### Step 7: Configure and Run Agent Locally

Edit `configs/agent.yaml`:
```yaml
server_host: "<SERVER_PUBLIC_IP>"
server_port: 8443
cert_file: "./certs/client.crt"
key_file: "./certs/client.key"
ca_file: "./certs/ca.crt"
```

Start agent:
```bash
make -f Makefile.<platform> run-agent-local
```

#### Step 8: Test

```bash
curl -x http://127.0.0.1:8080 http://example.com
```

#### Step 9: Stop Server (Optional)

```bash
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 0
```

**Why use this option:**
- Infrastructure as Code (repeatable, version-controlled)
- Secrets Manager secret managed with stack lifecycle
- Certificates passed securely as parameters
- Single source of truth for certificate generation (manage-certs.sh)
- Clean stack deletion removes all resources

**Cost:** ~$0.012/hour (~$9/month for 24/7, ~$0.50/month for occasional use)

**Certificate workflow:**
1. Local script generates certificates → `./certs/`
2. Deployment script reads `./certs/` and base64-encodes
3. CloudFormation creates Secrets Manager secret with certificate parameters
4. ECS task reads certificates from Secrets Manager at runtime
5. Stack deletion removes secret automatically

**See also:** [Infrastructure Guide](infrastructure.md) for CloudFormation details

---

### Option D: Lambda Control Plane (Cost Optimization Add-on)

Add automated lifecycle management to Option C for significant cost savings.

**Prerequisites:**
- Option C deployed (Fargate infrastructure)
- CloudWatch metrics enabled in server config

#### Step 1: Enable Server Metrics

Edit `configs/server.yaml`:
```yaml
emit_metrics: true
metrics_interval: "60s"
```

Rebuild and push server image:
```bash
make -f Makefile.<platform> push-server
```

Update ECS service to use new image:
```bash
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --force-new-deployment
```

#### Step 2: Deploy Lambda Control Plane

```bash
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

#### Step 3: Get API Endpoints

```bash
aws cloudformation describe-stacks \
  --stack-name fluidity-lambda \
  --query 'Stacks[0].Outputs'
```

Note the `WakeApiEndpoint` and `KillApiEndpoint` values.

#### Step 4: Configure Agent with Lifecycle Endpoints

Edit `configs/agent.yaml`:
```yaml
server_host: "<SERVER_PUBLIC_IP>"
server_port: 8443
wake_api_endpoint: "https://xxxxx.execute-api.us-east-1.amazonaws.com/prod/wake"
kill_api_endpoint: "https://xxxxx.execute-api.us-east-1.amazonaws.com/prod/kill"
connection_timeout: "90s"
connection_retry_interval: "5s"
```

#### Step 5: Test Automated Lifecycle

**Start agent:**
```bash
make -f Makefile.<platform> run-agent-local
```

Agent will:
1. Call Wake API → Server starts (DesiredCount=1)
2. Wait for server to be ready (~60s)
3. Connect and tunnel traffic
4. On shutdown, call Kill API → Server stops (DesiredCount=0)

**Sleep function** runs every 5 minutes:
- Checks CloudWatch metrics for idle connections
- If idle > 15 minutes → DesiredCount=0

**Daily kill** (optional):
- Scheduled shutdown at configured time
- Prevents runaway costs

**Why use this option:**
- **90% cost reduction** for occasional use
- Automated lifecycle (no manual start/stop)
- Pay only when actively using
- Ideal for personal development environments

**Cost:** Lambda ~$0.01/month + Fargate usage only when active = ~$0.11-0.21/month

**See also:** [Lambda Functions Guide](lambda.md) for architecture details

---

## Certificate Rotation

To rotate certificates (recommended every 6-12 months):

**1. Generate new certificates:**
```bash
./scripts/manage-certs.sh
```

**2. Redeploy CloudFormation stack:**
```bash
./scripts/deploy-fluidity.sh -e prod -a deploy
```

The deployment script will:
- Read new certificates from `./certs/`
- Update Secrets Manager secret via CloudFormation parameters
- CloudFormation updates the secret

**3. Restart Fargate server:**
```bash
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --force-new-deployment
```

Server pulls new certificates from Secrets Manager.

**4. Restart local agent:**

Agent uses new certificates from `./certs/` directory automatically on next start.

---

## Quick Reference

| Use Case | Option | Monthly Cost |
|----------|--------|--------------|
| Local development | A: Local binaries | Free |
| Test containers | B: Docker | Free |
| Cloud (always-on) | C: Fargate CloudFormation | ~$9 |
| Cloud (occasional) | C + D: Fargate + Lambda | ~$0.11-0.21 |

---

## Troubleshooting

### Windows: Certificate Revocation Error

**Error:** `CRYPT_E_NO_REVOCATION_CHECK`

**Solution:** Add `--ssl-no-revoke`:
```powershell
curl.exe -x http://127.0.0.1:8080 https://example.com --ssl-no-revoke
```

### Deployment Script: Certificates Not Found

**Error:** `Certificates not found in ./certs`

**Solution:** Run certificate generation first:
```bash
./scripts/manage-certs.sh
```

### Fargate: Task Stuck in PENDING

**Common causes:**
- Subnets not public
- `AssignPublicIp` not ENABLED
- Insufficient Fargate quota
- Invalid container image URI

**Solution:** Check CloudWatch logs and ECS task events

### Agent: Cannot Connect to Server

**Checklist:**
1. Is server running? (`desiredCount=1`)
2. Correct public IP in `agent.yaml`?
3. Security group allows your IP?
4. Certificates match (same CA)?

---

## Security Best Practices

1. **Restrict ingress:** Use your IP `/32` in `AllowedIngressCidr`
2. **Protect certificate files:** `.gitignore` already excludes `certs/*.key`
3. **Enable CloudWatch Logs retention:** Set 7-30 days
4. **Rotate certificates:** At least annually
5. **Use stack policies:** Prevent accidental deletions (included in deploy script)

---

## Related Documentation

- **[Certificate Guide](certificate.md)** - Certificate generation and management
- **[Docker Guide](docker.md)** - Container build and networking
- **[Fargate Guide](fargate.md)** - Detailed AWS ECS deployment
- **[Lambda Functions](lambda.md)** - Control plane architecture
- **[Infrastructure Guide](infrastructure.md)** - CloudFormation templates
- **[Architecture](architecture.md)** - System design overview
