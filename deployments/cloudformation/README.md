# Fluidity - AWS Fargate CloudFormation Deployment

This directory contains CloudFormation templates and supporting files for deploying Fluidity tunnel server on AWS Fargate (ECS).

## Files

- **`fargate.yaml`** - Main CloudFormation template
- **`params.example.json`** - Example parameters file (copy and customize)
- **`README.md`** - This file

## Prerequisites

1. **AWS CLI** v2 installed and configured
2. **Docker image** pushed to Amazon ECR
3. **VPC and Subnets** - Default VPC or your own with public subnets
4. **Certificates baked into Docker image** (see below)

## Quick Start

### 1. Build and Push Docker Image to ECR

```bash
# Create ECR repository
aws ecr create-repository --repository-name fluidity-server --region us-east-1

# Get ECR login
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Build image with certificates baked in
# IMPORTANT: Certificates must be in the certs/ directory before building
cd ../..  # Go to project root
make -f Makefile.win docker-build-server  # or Makefile.macos/Makefile.linux

# Tag for ECR
docker tag fluidity-server:latest 123456789012.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest

# Push to ECR
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/fluidity-server:latest
```

**Important**: The Docker image must include TLS certificates at `/root/certs/`. The simplest way is to have certificates generated in your local `certs/` directory before building the image, then rebuild the Dockerfile to explicitly copy them:

```dockerfile
# Add to deployments/server/Dockerfile after COPY build/fluidity-server:
COPY certs/ca.crt ./certs/
COPY certs/server.crt ./certs/
COPY certs/server.key ./certs/
```

### 2. Create Parameters File

Copy the example and customize:

```bash
cp params.example.json params.json
```

Edit `params.json` with your actual values:
- **ContainerImage**: Your ECR image URI
- **VpcId**: Your VPC ID (find with: `aws ec2 describe-vpcs`)
- **PublicSubnets**: Comma-separated subnet IDs (find with: `aws ec2 describe-subnets --filters "Name=vpc-id,Values=<VPC_ID>"`)
- **AllowedIngressCidr**: Your public IP with `/32` for security (find with: `curl ifconfig.me`)

### 3. Deploy Stack

```bash
aws cloudformation deploy \
  --template-file fargate.yaml \
  --stack-name fluidity-fargate \
  --parameter-overrides file://params.json \
  --capabilities CAPABILITY_NAMED_IAM \
  --region us-east-1
```

### 4. Start the Service

```bash
# Start (DesiredCount=1)
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 1 \
  --region us-east-1

# Wait ~60 seconds for task to start
sleep 60

# Get public IP
aws cloudformation describe-stacks \
  --stack-name fluidity-fargate \
  --query 'Stacks[0].Outputs[?OutputKey==`GetPublicIPCommand`].OutputValue' \
  --output text | bash
```

### 5. Configure Agent

On your local machine, update agent config:

```yaml
# configs/agent.yaml
server_ip: "<PUBLIC_IP_FROM_STEP_4>"
server_port: 8443
local_proxy_port: 8080
cert_file: "./certs/client.crt"
key_file: "./certs/client.key"
ca_cert_file: "./certs/ca.crt"
log_level: "info"
```

Run the agent:

```bash
# Windows
make -f Makefile.win run-agent-local

# macOS/Linux
make -f Makefile.macos run-agent-local  # or Makefile.linux
```

### 6. Test the Tunnel

```bash
# Windows
curl.exe -x http://127.0.0.1:8080 https://example.com -I --ssl-no-revoke

# macOS/Linux
curl -x http://127.0.0.1:8080 https://example.com -I
```

### 7. Stop the Service (to save costs)

```bash
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 0 \
  --region us-east-1
```

## Certificate Management

### Option A: Bake Certificates into Image (Simplest)

1. Generate certificates locally:
   ```bash
   ./scripts/generate-certs.ps1  # or .sh
   ```

2. Modify `deployments/server/Dockerfile` to copy certificates:
   ```dockerfile
   # Add after COPY build/fluidity-server .
   COPY certs/ca.crt ./certs/
   COPY certs/server.crt ./certs/
   COPY certs/server.key ./certs/
   ```

3. Rebuild and push image:
   ```bash
   make -f Makefile.win docker-build-server
   docker tag fluidity-server:latest <ECR_URI>
   docker push <ECR_URI>
   ```

### Option B: AWS Secrets Manager (Production)

1. Store certificates in Secrets Manager:
   ```bash
   aws secretsmanager create-secret \
     --name fluidity/server/ca-cert \
     --secret-string file://certs/ca.crt
   
   aws secretsmanager create-secret \
     --name fluidity/server/server-cert \
     --secret-string file://certs/server.crt
   
   aws secretsmanager create-secret \
     --name fluidity/server/server-key \
     --secret-string file://certs/server.key
   ```

2. Modify CloudFormation template to add secrets to task definition:
   ```yaml
   Secrets:
     - Name: CA_CERT
       ValueFrom: arn:aws:secretsmanager:us-east-1:123456789012:secret:fluidity/server/ca-cert
     - Name: SERVER_CERT
       ValueFrom: arn:aws:secretsmanager:us-east-1:123456789012:secret:fluidity/server/server-cert
     - Name: SERVER_KEY
       ValueFrom: arn:aws:secretsmanager:us-east-1:123456789012:secret:fluidity/server/server-key
   ```

3. Update application code to read from environment variables or write secrets to files on startup.

### Option C: Amazon EFS (Shared Storage)

1. Create EFS file system and mount target in your VPC
2. Copy certificates to EFS
3. Add EFS volume to task definition
4. Mount volume to `/root/certs` in container

## Cost Estimation

### Fargate Costs (us-east-1)
- **0.25 vCPU**: $0.04048/hour
- **0.5 GB RAM**: $0.004445/hour
- **Total**: ~$0.012/hour

### Usage Scenarios
- **24/7 operation**: ~$9/month
- **8 hours/day**: ~$3/month
- **2 hours/day**: ~$0.50/month
- **On-demand (pay per second)**: Spin up when needed, DesiredCount=0 when not in use

### Additional Costs
- **CloudWatch Logs**: Minimal (~$0.50/month for moderate usage)
- **ECR Storage**: ~$0.10/GB/month
- **Data Transfer**: $0.09/GB outbound (first 100GB/month free)

## Stack Management

### Update Stack

```bash
# After modifying parameters or template
aws cloudformation deploy \
  --template-file fargate.yaml \
  --stack-name fluidity-fargate \
  --parameter-overrides file://params.json \
  --capabilities CAPABILITY_NAMED_IAM
```

### View Stack Outputs

```bash
aws cloudformation describe-stacks \
  --stack-name fluidity-fargate \
  --query 'Stacks[0].Outputs'
```

### View Logs

```bash
# Get log stream names
aws logs describe-log-streams \
  --log-group-name /ecs/fluidity/server \
  --order-by LastEventTime \
  --descending \
  --max-items 1

# View latest logs
aws logs tail /ecs/fluidity/server --follow
```

### Delete Stack

```bash
# Stop service first
aws ecs update-service \
  --cluster fluidity \
  --service fluidity-server \
  --desired-count 0

# Delete stack
aws cloudformation delete-stack \
  --stack-name fluidity-fargate

# Wait for deletion
aws cloudformation wait stack-delete-complete \
  --stack-name fluidity-fargate
```

## Troubleshooting

### Task fails to start

**Check logs:**
```bash
aws logs tail /ecs/fluidity/server --since 10m
```

**Common issues:**
- Missing certificates in image
- Invalid certificate paths in config
- Port already in use
- Insufficient memory/CPU

### Cannot get public IP

**Cause**: Task not running yet or AssignPublicIp=DISABLED

**Solution:**
```bash
# Check task status
aws ecs describe-services \
  --cluster fluidity \
  --services fluidity-server

# Ensure AssignPublicIp=ENABLED in params.json
```

### Agent cannot connect

**Check:**
- Correct public IP (task IPs change when restarted)
- Security group allows your IP on port 8443
- Certificates match (same CA on agent and server)
- Server is actually running (check logs)

### High costs

**Optimize:**
- Set DesiredCount=0 when not in use
- Use smaller CPU/Memory (256/512 is minimum for Fargate)
- Set short CloudWatch log retention (7-14 days)
- Delete stack completely when not needed for extended periods

## Security Best Practices

1. **Restrict ingress**: Use your public IP/32 instead of 0.0.0.0/0
2. **Rotate certificates**: Update image with new certs periodically
3. **Use Secrets Manager**: Don't bake production certificates into images
4. **Monitor logs**: Set up CloudWatch alarms for errors
5. **VPC security**: Use private subnets with NAT gateway for production
6. **IAM least privilege**: Review and restrict TaskRole permissions

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                     AWS Cloud                       │
│                                                     │
│  ┌─────────────────────────────────────────────┐  │
│  │              VPC (Your VPC)                  │  │
│  │                                              │  │
│  │  ┌──────────────────────────────────────┐  │  │
│  │  │        Public Subnet                  │  │  │
│  │  │                                       │  │  │
│  │  │  ┌─────────────────────────────┐    │  │  │
│  │  │  │   ECS Fargate Task          │    │  │  │
│  │  │  │   ┌──────────────────────┐  │    │  │  │
│  │  │  │   │ fluidity-server      │  │    │  │  │
│  │  │  │   │ Alpine container     │  │    │  │  │
│  │  │  │   │ Listens: 0.0.0.0:8443 │  │    │  │  │
│  │  │  │   └──────────────────────┘  │    │  │  │
│  │  │  │   Public IP: xxx.xxx.xxx.xxx │    │  │  │
│  │  │  └─────────────────────────────┘    │  │  │
│  │  │                 │                    │  │  │
│  │  └─────────────────┼────────────────────┘  │  │
│  │                    │                        │  │
│  │         ┌──────────▼─────────┐             │  │
│  │         │  Security Group     │             │  │
│  │         │  Allow: YourIP:8443 │             │  │
│  │         └─────────────────────┘             │  │
│  └─────────────────────────────────────────────┘  │
│                                                     │
│  ┌─────────────────────────────────────────────┐  │
│  │        CloudWatch Logs                       │  │
│  │        /ecs/fluidity/server                  │  │
│  │        Retention: 14 days                    │  │
│  └─────────────────────────────────────────────┘  │
│                                                     │
│  ┌─────────────────────────────────────────────┐  │
│  │        Amazon ECR                            │  │
│  │        fluidity-server:latest                │  │
│  │        Size: ~44MB                           │  │
│  └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                        ▲
                        │ mTLS on port 8443
                        │
┌───────────────────────┼─────────────────────────────┐
│              Local Network                          │
│                      │                              │
│  ┌───────────────────▼──────────────────────────┐  │
│  │    fluidity-agent (local binary/container)   │  │
│  │    Proxy: 127.0.0.1:8080                     │  │
│  │    Connects to: Public IP:8443                │  │
│  └───────────────────────────────────────────────┘  │
│                      ▲                              │
│                      │ HTTP/HTTPS proxy             │
│         ┌────────────┴────────────┐                 │
│         │   Browser / curl        │                 │
│         │   Proxy: 127.0.0.1:8080 │                 │
│         └─────────────────────────┘                 │
└─────────────────────────────────────────────────────┘
```

## Additional Resources

- [AWS Fargate Pricing](https://aws.amazon.com/fargate/pricing/)
- [ECS Task Definitions](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definitions.html)
- [CloudFormation Reference](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html)
- [Fluidity Documentation](../../docs/)
