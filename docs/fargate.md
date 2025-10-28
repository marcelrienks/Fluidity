# AWS Fargate Deployment Plan — Fluidity Server

This guide describes how to deploy the Fluidity Tunnel Server to AWS using ECS on Fargate. It focuses on simplicity, pay‑per‑use operation (start/stop on demand), and minimal AWS footprint.

---

## Goals

- Run the Fluidity server Docker image on AWS without managing EC2 instances
- Keep costs near-zero when not in use by setting desired count to 0
- Expose port 8443 over the Internet with tight security group rules
- Keep logs in CloudWatch
- Make daily usage simple via start/stop scripts that also output the public IP

---

## Prerequisites

- AWS account with permissions for ECR, ECS, EC2 (VPC/subnets/SG), IAM, CloudWatch Logs
- AWS CLI v2 configured (`aws configure`)
- Docker installed locally
- Private image for the server (built from this repo)
  - Recommended: use the existing Dockerfile in `deployments/server/Dockerfile`
  - Or Makefile target: `make -f Makefile.win docker-build-server` (or macOS/Linux Makefiles)
- Certificates and config ready (see `certs/` and `configs/server.*.yaml`)

Security note: For personal use you may bake certs/configs into the image for simplicity. Prefer using AWS Secrets Manager or SSM Parameter Store for sensitive materials if you plan to share or harden this.

---

## Architecture Overview

- ECS Cluster (Fargate launch type)
- Task Definition `fluidity-server` (0.25 vCPU, 512MB RAM)
- Service with desired count toggled between 0 and 1
- Networking: default VPC, public subnets, assign public IP
- Security group inbound TCP 8443 from your workstation’s public IP (restrictive), or 0.0.0.0/0 for testing only
- CloudWatch Logs for container stdout/stderr

Sequence:
1) Push image to ECR
2) Create IAM roles, log group, network (SG)
3) Register task definition
4) Create service (desired count 0)
5) Start on demand (desired count 1), fetch task public IP
6) Use that IP in the Agent `--server-ip` (or config)
7) Stop when done (desired count 0)

---

## Image Build and Push (ECR)

- Create an ECR repository (one‑time)
- Build the server image using existing Makefile or Dockerfile
- Tag and push the image to ECR

Example outline (replace REGION/ACCOUNT):

```powershell
# Create ECR repository (one-time)
aws ecr create-repository --repository-name fluidity-server

# Authenticate Docker to ECR
aws ecr get-login-password --region <REGION> | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com

# Build, tag, push
# Option A (Makefile):
# make -f Makefile.win docker-build-server
# docker tag fluidity-server:latest <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest

# Option B (Dockerfile only):
# docker build -t fluidity-server -f deployments/server/Dockerfile .
docker tag fluidity-server:latest <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest
docker push <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest
```

---

## IAM, Networking, and Logs

- Task execution role: use managed `ecsTaskExecutionRole` (grants pull from ECR, write logs)
- Task role: not required unless accessing other AWS services from the container
- Log group: create `/fluidity/server` in CloudWatch Logs (or let ECS create on first run)
- VPC & Subnets: use default VPC with at least one public subnet
- Security Group: inbound TCP 8443 from your current public IP (recommended), or 0.0.0.0/0 for testing

```powershell
# Create CloudWatch log group (optional)
aws logs create-log-group --log-group-name /fluidity/server

# Create a security group (in default VPC); note the VPC id if you have multiple VPCs
$defaultVpcId = (aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text)
$sgId = aws ec2 create-security-group --group-name fluidity-sg --description "Fluidity server SG" --vpc-id $defaultVpcId --query 'GroupId' --output text

# Allow inbound 8443 only from your current public IP (replace x.x.x.x/32)
aws ec2 authorize-security-group-ingress --group-id $sgId --protocol tcp --port 8443 --cidr x.x.x.x/32
```

---

## Task Definition (Fargate)

Minimal example (save as `task-definition.json` and register):

```json
{
  "family": "fluidity-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "server",
      "image": "<ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest",
      "portMappings": [
        { "containerPort": 8443, "hostPort": 8443, "protocol": "tcp" }
      ],
      "essential": true,
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/fluidity/server",
          "awslogs-region": "<REGION>",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "environment": [
        { "name": "FLUIDITY_LOG_LEVEL", "value": "info" }
      ]
    }
  ]
}
```

Register the task definition (example):

```powershell
aws ecs register-task-definition --cli-input-json file://task-definition.json
```

---

## ECS Cluster and Service

Create a cluster (one‑time), then a service referencing the task definition. Use public subnets and the security group created earlier, and enable `assignPublicIp`.

```powershell
# Create or reuse an ECS cluster
aws ecs create-cluster --cluster-name fluidity

# Get one or more public subnet ids from default VPC
$subnetIds = aws ec2 describe-subnets --filters Name=vpc-id,Values=$defaultVpcId Name=defaultForAz,Values=true --query 'Subnets[].SubnetId' --output text

# Create service with desired count 0 (off by default)
aws ecs create-service `
  --cluster fluidity `
  --service-name fluidity-server `
  --task-definition fluidity-server `
  --desired-count 0 `
  --launch-type FARGATE `
  --network-configuration "awsvpcConfiguration={subnets=[$subnetIds],securityGroups=[$sgId],assignPublicIp=ENABLED}"
```

---

## Start/Stop On Demand + Retrieve Public IP

Start when needed (desired count = 1), wait ~60 seconds for cold start, retrieve the task’s ENI public IP, then configure the Agent with `--server-ip`.

PowerShell (save as `scripts/start-cloud-server.ps1`):

```powershell
Write-Host "Starting Fluidity server in AWS..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1 | Out-Null

Write-Host "Waiting for server to start (60 seconds)..."
Start-Sleep -Seconds 60

$taskArn = aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text
if (-not $taskArn -or $taskArn -eq "None") { throw "No running task found" }

$eniId = aws ecs describe-tasks --cluster fluidity --tasks $taskArn `
  --query 'tasks[0].attachments[0].details[?name==`"networkInterfaceId"`].value' --output text

$publicIp = aws ec2 describe-network-interfaces --network-interface-ids $eniId `
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text

Write-Host "Server started. Public IP: $publicIp"
Write-Host "Tip: Run agent with: .\\build\\fluidity-agent --config .\\configs\\agent.local.yaml --server-ip $publicIp"
```

Stop (save as `scripts/stop-cloud-server.ps1`):

```powershell
Write-Host "Stopping Fluidity server..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0 | Out-Null
Write-Host "Server stopped. Billing paused."
```

Bash equivalents (optional):

```bash
# scripts/start-cloud-server.sh
#!/usr/bin/env bash
set -euo pipefail

echo "Starting Fluidity server in AWS..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1 >/dev/null

echo "Waiting for server to start (60 seconds)..."
sleep 60

TASK_ARN=$(aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text)
if [[ -z "$TASK_ARN" || "$TASK_ARN" == "None" ]]; then echo "No running task found"; exit 1; fi

ENI_ID=$(aws ecs describe-tasks --cluster fluidity --tasks "$TASK_ARN" \
  --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text)
PUBLIC_IP=$(aws ec2 describe-network-interfaces --network-interface-ids "$ENI_ID" \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "Server started. Public IP: $PUBLIC_IP"
echo "Run agent with: ./build/fluidity-agent --config ./configs/agent.local.yaml --server-ip $PUBLIC_IP"
```

```bash
# scripts/stop-cloud-server.sh
#!/usr/bin/env bash
set -euo pipefail

echo "Stopping Fluidity server..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0 >/dev/null
echo "Server stopped. Billing paused."
```

---

## Handling Dynamic IPs

By default a Fargate task gets a new public IP each time it starts. Options:
- Simple (recommended personal use): fetch the IP after start (scripts above) and pass `--server-ip <IP>` to the Agent; persist it in your agent config once verified.
- Network Load Balancer + static endpoint: Put the service behind an NLB with TCP listener 8443; adds cost/complexity but keeps a stable endpoint.
- Elastic IP via NAT/LB: Requires additional architecture; not necessary for personal on‑demand use.

---

## Security Recommendations

- Restrict Security Group ingress on 8443 to your current public IP
- Keep mTLS enforced (already in Fluidity)
- Rotate certificates periodically; keep private keys out of public repos
- If baking certs into the image, keep ECR private and least‑privilege access
- Set CloudWatch Logs retention (e.g., 7–30 days) and avoid logging sensitive data

---

## Lambda Control Plane Integration (Optional)

For automated lifecycle management and cost optimization, you can integrate the Fargate deployment with AWS Lambda functions that control the ECS service based on activity metrics. This is recommended for production deployments.

### Architecture Overview

The Lambda control plane provides three key capabilities:
1. **Wake Lambda** - Starts the ECS service when agents need connectivity
2. **Sleep Lambda** - Stops the service after idle period to minimize costs
3. **Kill Lambda** - Emergency shutdown endpoint

### Key Components

**CloudWatch Metrics:**
The server emits custom metrics to CloudWatch (namespace: `Fluidity`):
- `ActiveConnections` - Current number of connected agents
- `LastActivityEpochSeconds` - Unix timestamp of last activity

**Server Configuration:**
Enable metrics in `server.yaml`:
```yaml
server:
  host: "0.0.0.0"
  port: 8443
  
metrics:
  emit_metrics: true          # Enable CloudWatch metrics
  metrics_interval: 60s       # Emit every 60 seconds
  namespace: "Fluidity"       # CloudWatch namespace
  service_name: "fluidity-server"
```

**Agent Configuration:**
Configure lifecycle integration in `agent.yaml`:
```yaml
agent:
  proxy_port: 8080
  server_ip: "<task-public-ip>"
  server_port: 8443
  
lifecycle:
  wake_endpoint: "https://<api-gateway-url>/wake"
  kill_endpoint: "https://<api-gateway-url>/kill"
  api_key: "<api-gateway-key>"
  wake_timeout: 90s           # Retry connection for 90 seconds
  wake_retry_interval: 5s     # Check every 5 seconds
```

**EventBridge Schedulers:**
- **Sleep Schedule**: `rate(5 minutes)` - Periodically checks if service is idle
- **Kill Schedule**: `cron(0 23 * * ? *)` - Daily shutdown at 11 PM UTC

### Deployment

1. **Deploy Fargate infrastructure first** (this guide or `fargate.yaml`)

2. **Deploy Lambda control plane** (see `docs/deployment.md` Option E):
```bash
# Create lambda.yaml CloudFormation stack
aws cloudformation deploy \
  --template-file deployments/cloudformation/lambda.yaml \
  --stack-name fluidity-lambda \
  --parameter-overrides \
      EcsClusterName=fluidity-cluster \
      EcsServiceName=fluidity-server \
      IdleThresholdMinutes=5 \
  --capabilities CAPABILITY_IAM
```

3. **Update server configuration** with CloudWatch metrics enabled

4. **Update agent configuration** with wake/kill endpoints from API Gateway

### Monitoring

**View CloudWatch Metrics:**
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

# Check last activity timestamp
aws cloudwatch get-metric-statistics \
  --namespace Fluidity \
  --metric-name LastActivityEpochSeconds \
  --dimensions Name=ServiceName,Value=fluidity-server \
  --statistics Maximum \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300
```

**Test Lambda Functions:**
```bash
# Manually trigger wake
curl -X POST https://<api-gateway-url>/wake \
  -H "x-api-key: <your-api-key>"

# Check ECS service status
aws ecs describe-services \
  --cluster fluidity-cluster \
  --services fluidity-server \
  --query 'services[0].desiredCount'

# Manually trigger kill
curl -X POST https://<api-gateway-url>/kill \
  -H "x-api-key: <your-api-key>"
```

### Cost Impact

With Lambda control plane automation:
- **Idle periods**: $0/hour (service stopped)
- **Active periods**: $0.012/hour (Fargate running)
- **Lambda costs**: ~$0.20-$0.50/month (minimal invocations)
- **Total**: $0.55-$3.05/month vs $3-$9/month for manual management

For complete Lambda deployment guide, see `docs/deployment.md` Option E.

---

## Costs (rough order of magnitude)

Fargate (0.25 vCPU, 0.5 GB): ~ $0.012/hour
- 2 hours/day, 20 days/month ≈ $0.50/month
- 8 hours/day, 20 days/month ≈ $3.00/month
Always‑on 24/7: ≈ $9/month

CloudWatch, ECR storage, data transfer may add small additional costs.

**With Lambda Control Plane:**
- Lambda invocations: ~$0.20-$0.50/month
- API Gateway: ~$0.03/month (10,000 requests free tier)
- Total: $0.55-$3.05/month (automatic idle shutdown)

---

## Troubleshooting

- Task stuck in PENDING: check subnets are public, `assignPublicIp=ENABLED`, SG allows 8443, and your account has Fargate quotas in the region
- Can’t connect: confirm SG ingress from your current IP; verify server is listening on 8443 inside the container
- No logs: confirm `awslogs` log driver options and log group exist
- TLS errors: verify CA and server certificates match; server uses the correct cert files

---

## Deploy with CloudFormation

You can provision the full Fargate setup via a single CloudFormation stack: see `deployments/cloudformation/fargate.yaml`.

### Creates
- ECS Cluster (Fargate)
- IAM execution role for tasks (`AmazonECSTaskExecutionRolePolicy`)
- CloudWatch Log Group (default `/fluidity/server`)
- Security Group with inbound TCP on the chosen port (default 8443)
- ECS Task Definition (CPU/Memory, logging, environment)
- ECS Service (desired count default 0)

It does not create an ECR repo or push the image. Build and push the image first and pass the ECR image URI to the stack.

### Required parameters
- `ContainerImage`: ECR URI, for example `123456789012.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest`
- `VpcId`: VPC ID
- `PublicSubnets`: One or more public subnet IDs (comma-separated in params file)

### Optional parameters (defaults)
- `ClusterName` (fluidity), `ServiceName` (fluidity-server)
- `ContainerPort` (8443), `Cpu` (256), `Memory` (512)
- `DesiredCount` (0), `AllowedIngressCidr` (0.0.0.0/0)
- `AssignPublicIp` (ENABLED), `LogGroupName` (/fluidity/server), `LogRetentionDays` (14)

### Example params file

Save as `deployments/cloudformation/params.json`:

```json
[
  { "ParameterKey": "ClusterName", "ParameterValue": "fluidity" },
  { "ParameterKey": "ServiceName", "ParameterValue": "fluidity-server" },
  { "ParameterKey": "ContainerImage", "ParameterValue": "123456789012.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest" },
  { "ParameterKey": "ContainerPort", "ParameterValue": "8443" },
  { "ParameterKey": "Cpu", "ParameterValue": "256" },
  { "ParameterKey": "Memory", "ParameterValue": "512" },
  { "ParameterKey": "DesiredCount", "ParameterValue": "0" },
  { "ParameterKey": "VpcId", "ParameterValue": "vpc-0abcd1234ef567890" },
  { "ParameterKey": "PublicSubnets", "ParameterValue": "subnet-0123abcd,subnet-0456efgh" },
  { "ParameterKey": "AllowedIngressCidr", "ParameterValue": "x.x.x.x/32" },
  { "ParameterKey": "AssignPublicIp", "ParameterValue": "ENABLED" },
  { "ParameterKey": "LogGroupName", "ParameterValue": "/fluidity/server" },
  { "ParameterKey": "LogRetentionDays", "ParameterValue": "14" }
]
```

Tip: Replace `x.x.x.x/32` with your current public IP to restrict access.

### Create/Update the stack (PowerShell)

```powershell
$stackName = "fluidity-fargate"

# Create/update stack
aws cloudformation deploy `
  --template-file deployments/cloudformation/fargate.yaml `
  --stack-name $stackName `
  --parameter-overrides (Get-Content deployments/cloudformation/params.json | Out-String) `
  --capabilities CAPABILITY_NAMED_IAM

# Check status
aws cloudformation describe-stacks --stack-name $stackName `
  --query 'Stacks[0].StackStatus'

# Outputs
aws cloudformation describe-stacks --stack-name $stackName `
  --query 'Stacks[0].Outputs'
```

### Start/Stop via stack updates

Update only the `DesiredCount` parameter (1 to start, 0 to stop):

```powershell
aws cloudformation update-stack `
  --stack-name $stackName `
  --use-previous-template `
  --parameters ParameterKey=DesiredCount,ParameterValue=1 `
               ParameterKey=ClusterName,UsePreviousValue=true `
               ParameterKey=ServiceName,UsePreviousValue=true `
               ParameterKey=ContainerImage,UsePreviousValue=true `
               ParameterKey=ContainerPort,UsePreviousValue=true `
               ParameterKey=Cpu,UsePreviousValue=true `
               ParameterKey=Memory,UsePreviousValue=true `
               ParameterKey=VpcId,UsePreviousValue=true `
               ParameterKey=PublicSubnets,UsePreviousValue=true `
               ParameterKey=AllowedIngressCidr,UsePreviousValue=true `
               ParameterKey=AssignPublicIp,UsePreviousValue=true `
               ParameterKey=LogGroupName,UsePreviousValue=true `
               ParameterKey=LogRetentionDays,UsePreviousValue=true
```

Note: The task’s public IP is assigned at runtime and can’t be output directly by CloudFormation; use the start script to retrieve it.

---

## Appendix: Full Task Definition (with env + logs)

```json
{
  "family": "fluidity-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "server",
      "image": "<ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest",
      "essential": true,
      "portMappings": [
        { "containerPort": 8443, "hostPort": 8443, "protocol": "tcp" }
      ],
      "environment": [
        { "name": "FLUIDITY_LOG_LEVEL", "value": "info" }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/fluidity/server",
          "awslogs-region": "<REGION>",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

Replace `<ACCOUNT_ID>` and `<REGION>`; adjust resources and options as needed.

---

## What’s Next

- Optionally add EventBridge rules to auto start/stop on a schedule (work hours)
- Consider moving certs/configs to AWS Secrets Manager and inject at runtime
- Add a tiny PowerShell/Bash wrapper that starts the service, waits, updates Agent config file automatically, and launches the agent
