# AWS Fargate Deployment Guide

Deploy Fluidity server to AWS ECS using Fargate for pay-per-use operation.

---

## Overview

**Goals:**
- Run server on AWS without managing EC2 instances
- Pay only when running (set desired count to 0 when idle)
- Expose port 8443 with secure access
- Simple start/stop workflow

**Architecture:**
- ECS Cluster (Fargate launch type)
- Task Definition (0.25 vCPU, 512 MB RAM)
- Service with desired count toggle (0 = stopped, 1 = running)
- Public subnet with public IP
- Security Group restricting port 8443
- CloudWatch Logs

---

## Prerequisites

- AWS account with ECR, ECS, EC2, IAM, CloudWatch permissions
- AWS CLI v2 configured
- Docker installed
- Server Docker image built (see **[Docker Guide](docker.md)**)

---

## Step 1: Build and Push Image to ECR

```powershell
# Create ECR repository (one-time)
aws ecr create-repository --repository-name fluidity-server

# Authenticate Docker to ECR
aws ecr get-login-password --region <REGION> | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com

# Build, tag, and push
make -f Makefile.win docker-build-server
docker tag fluidity-server:latest <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest
docker push <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest
```

---

## Step 2: Create IAM, Networking, and Logs

**CloudWatch Log Group:**
```powershell
aws logs create-log-group --log-group-name /fluidity/server
```

**Security Group:**
```powershell
# Get default VPC ID
$defaultVpcId = (aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text)

# Create security group
$sgId = aws ec2 create-security-group --group-name fluidity-sg --description "Fluidity server SG" --vpc-id $defaultVpcId --query 'GroupId' --output text

# Allow port 8443 from your IP only (replace x.x.x.x/32)
aws ec2 authorize-security-group-ingress --group-id $sgId --protocol tcp --port 8443 --cidr x.x.x.x/32
```

**IAM Role:**
Use AWS managed `ecsTaskExecutionRole` (grants ECR pull and CloudWatch Logs write).

---

## Step 3: Create Task Definition

**task-definition.json:**
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

**Register:**
```powershell
aws ecs register-task-definition --cli-input-json file://task-definition.json
```

---

## Step 4: Create ECS Cluster and Service

```powershell
# Create cluster
aws ecs create-cluster --cluster-name fluidity

# Get public subnet IDs
$subnetIds = aws ec2 describe-subnets --filters Name=vpc-id,Values=$defaultVpcId Name=defaultForAz,Values=true --query 'Subnets[].SubnetId' --output text

# Create service (desired count 0 = stopped by default)
aws ecs create-service `
  --cluster fluidity `
  --service-name fluidity-server `
  --task-definition fluidity-server `
  --desired-count 0 `
  --launch-type FARGATE `
  --network-configuration "awsvpcConfiguration={subnets=[$subnetIds],securityGroups=[$sgId],assignPublicIp=ENABLED}"
```

---

## Step 5: Start/Stop On Demand

### Start Server and Get Public IP

**PowerShell (scripts/start-cloud-server.ps1):**
```powershell
Write-Host "Starting Fluidity server..."
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
Write-Host "Run agent: .\build\fluidity-agent --config .\configs\agent.local.yaml --server-ip $publicIp"
```

**Bash (scripts/start-cloud-server.sh):**
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Starting Fluidity server..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1 >/dev/null

echo "Waiting for server to start (60 seconds)..."
sleep 60

TASK_ARN=$(aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text)
[[ -z "$TASK_ARN" || "$TASK_ARN" == "None" ]] && { echo "No running task found"; exit 1; }

ENI_ID=$(aws ecs describe-tasks --cluster fluidity --tasks "$TASK_ARN" \
  --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text)
PUBLIC_IP=$(aws ec2 describe-network-interfaces --network-interface-ids "$ENI_ID" \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "Server started. Public IP: $PUBLIC_IP"
echo "Run agent: ./build/fluidity-agent --config ./configs/agent.local.yaml --server-ip $PUBLIC_IP"
```

### Stop Server

**PowerShell (scripts/stop-cloud-server.ps1):**
```powershell
Write-Host "Stopping Fluidity server..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0 | Out-Null
Write-Host "Server stopped. Billing paused."
```

**Bash (scripts/stop-cloud-server.sh):**
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Stopping Fluidity server..."
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0 >/dev/null
echo "Server stopped. Billing paused."
```

---

## CloudFormation Deployment (Alternative)

For repeatable infrastructure, use the CloudFormation template.

**Template:** `deployments/cloudformation/fargate.yaml`

**Parameters file (deployments/cloudformation/params.json):**
```json
[
  { "ParameterKey": "ClusterName", "ParameterValue": "fluidity" },
  { "ParameterKey": "ServiceName", "ParameterValue": "fluidity-server" },
  { "ParameterKey": "ContainerImage", "ParameterValue": "123456789012.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest" },
  { "ParameterKey": "ContainerPort", "ParameterValue": "8443" },
  { "ParameterKey": "Cpu", "ParameterValue": "256" },
  { "ParameterKey": "Memory", "ParameterValue": "512" },
  { "ParameterKey": "DesiredCount", "ParameterValue": "0" },
  { "ParameterKey": "VpcId", "ParameterValue": "vpc-xxxxx" },
  { "ParameterKey": "PublicSubnets", "ParameterValue": "subnet-xxxxx,subnet-yyyyy" },
  { "ParameterKey": "AllowedIngressCidr", "ParameterValue": "x.x.x.x/32" },
  { "ParameterKey": "AssignPublicIp", "ParameterValue": "ENABLED" },
  { "ParameterKey": "LogGroupName", "ParameterValue": "/fluidity/server" },
  { "ParameterKey": "LogRetentionDays", "ParameterValue": "14" }
]
```

**Deploy:**
```powershell
aws cloudformation deploy `
  --template-file deployments/cloudformation/fargate.yaml `
  --stack-name fluidity-fargate `
  --parameter-overrides (Get-Content deployments/cloudformation/params.json | Out-String) `
  --capabilities CAPABILITY_NAMED_IAM
```

**Start/Stop:**
```powershell
# Start
aws cloudformation update-stack `
  --stack-name fluidity-fargate `
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

# Stop (set DesiredCount to 0)
```

---

## Dynamic IP Handling

Fargate tasks get new public IPs on each start.

**Options:**
1. **Simple (recommended for personal use):** Fetch IP after start (scripts above) and pass to agent via `--server-ip`
2. **Network Load Balancer:** Adds static endpoint but increases cost/complexity
3. **Elastic IP via NAT:** Requires additional architecture

For most use cases, option 1 is sufficient.

---

## Lambda Control Plane Integration

For automated lifecycle management with cost optimization, integrate with Lambda functions.

**Features:**
- **Wake Lambda:** Agent calls on startup → sets ECS desired count to 1
- **Sleep Lambda:** EventBridge scheduler → checks CloudWatch metrics → scales down if idle
- **Kill Lambda:** Agent calls on shutdown OR daily scheduled shutdown

**Prerequisites:**
- Fargate infrastructure deployed (this guide or CloudFormation)
- Server configured to emit CloudWatch metrics

**Configuration:**

Server (`configs/server.yaml`):
```yaml
emit_metrics: true
metrics_interval: "60s"
```

Agent (`configs/agent.local.yaml`):
```yaml
wake_api_endpoint: "https://xxx.execute-api.us-east-1.amazonaws.com/prod/wake"
kill_api_endpoint: "https://xxx.execute-api.us-east-1.amazonaws.com/prod/kill"
api_key: "your-api-key"
connection_timeout: "90s"
connection_retry_interval: "5s"
```

**Full instructions:** See **[Lambda Functions Guide](lambda.md)**

**Cost impact:** Lambda < $0.05/month + Fargate usage = $0.55-$3.05/month total

---

## Costs

**Fargate (0.25 vCPU, 0.5 GB):** ~$0.012/hour
- 2 hours/day, 20 days/month: ~$0.50/month
- 8 hours/day, 20 days/month: ~$3/month
- 24/7: ~$9/month

**Additional:** CloudWatch Logs, ECR storage, data transfer (minimal)

**With Lambda Control Plane:** $0.55-$3.05/month (automatic idle shutdown)

---

## Security Best Practices

1. **Restrict Security Group:** Limit port 8443 to your IP (`/32` CIDR)
2. **Keep mTLS enabled:** Certificate validation required
3. **Protect private keys:** Never commit to repos
4. **Use Secrets Manager:** Store certificates in AWS Secrets Manager for production
5. **Set log retention:** 7-30 days in CloudWatch Logs
6. **Keep ECR private:** Restrict repository access

---

## Troubleshooting

**Task stuck in PENDING:**
- Check subnets are public
- Verify `assignPublicIp=ENABLED`
- Confirm Fargate quota in region
- Check security group configuration

**Can't connect:**
- Verify security group allows port 8443 from your IP
- Confirm server is listening on 0.0.0.0:8443
- Check CloudWatch Logs for startup errors

**No logs in CloudWatch:**
- Verify log group exists (`/fluidity/server`)
- Check `awslogs` configuration in task definition
- Confirm IAM execution role has CloudWatch permissions

**TLS errors:**
- Verify certificates match (CA, server cert, keys)
- Confirm server is using correct cert files
- Check certificate paths in task definition

---

## Next Steps

- **Add Lambda control plane:** Automated lifecycle with cost optimization (see **[Lambda Functions](lambda.md)**)
- **Use Secrets Manager:** Move certificates to AWS Secrets Manager
- **Add alarms:** CloudWatch alarms for connection failures or high error rates
- **Enable Container Insights:** Advanced ECS monitoring

---

## Related Documentation

- **[Deployment Guide](deployment.md)** - All deployment options overview
- **[Lambda Functions](lambda.md)** - Automated lifecycle management
- **[Docker Guide](docker.md)** - Container build process
- **[Architecture](architecture.md)** - System design details
