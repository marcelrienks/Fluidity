# Operational Runbook

This document provides operational procedures for running and troubleshooting Fluidity in production.

---

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Manual Lifecycle Control](#manual-lifecycle-control)
3. [Monitoring and Alerts](#monitoring-and-alerts)
4. [Troubleshooting](#troubleshooting)
5. [Incident Response](#incident-response)
6. [Maintenance](#maintenance)

---

## Daily Operations

### Starting Fluidity Server (Cloud)

**Automated (Recommended):**
Agent handles startup automatically with Lambda control plane:

```bash
# Just run the agent - it will wake the server
./build/fluidity-agent --config ./configs/agent.yaml
```

Agent will:
1. Call Wake Lambda API
2. Wait for Fargate task to start (up to 90 seconds)
3. Connect to server and begin tunneling

**Manual (if Lambda control plane disabled):**

```powershell
# PowerShell
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1
Start-Sleep -Seconds 60  # Wait for task startup

$taskArn = aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text
$eniId = aws ecs describe-tasks --cluster fluidity --tasks $taskArn --query 'tasks[0].attachments[0].details[?name==`"networkInterfaceId"`].value' --output text
$publicIp = aws ec2 describe-network-interfaces --network-interface-ids $eniId --query 'NetworkInterfaces[0].Association.PublicIp' --output text

Write-Host "Server IP: $publicIp"
```

```bash
# Bash
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1
sleep 60

TASK_ARN=$(aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text)
ENI_ID=$(aws ecs describe-tasks --cluster fluidity --tasks "$TASK_ARN" --query 'tasks[0].attachments[0].details[?name==networkInterfaceId].value' --output text)
PUBLIC_IP=$(aws ec2 describe-network-interfaces --network-interface-ids "$ENI_ID" --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "Server IP: $PUBLIC_IP"
```

### Stopping Fluidity Server (Cloud)

**Automated (Recommended):**
Agent handles shutdown automatically:

```bash
# Kill agent process (Ctrl+C)
# Agent will call Kill Lambda and gracefully shutdown
```

Agent will:
1. Call Kill Lambda API
2. Server receives graceful shutdown signal
3. Fargate task terminates cleanly

**Manual (if Lambda control plane disabled):**

```powershell
# PowerShell
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0
```

```bash
# Bash
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0
```

### Local Testing

```bash
# Terminal 1: Start server locally
make -f Makefile.win run-server-local  # Windows
make -f Makefile.macos run-server-local  # macOS
make -f Makefile.linux run-server-local  # Linux

# Terminal 2: Start agent locally
make -f Makefile.win run-agent-local
```

---

## Manual Lifecycle Control

Use these procedures for manual control when Lambda automation is unavailable.

### Invoke Wake Lambda Directly

```powershell
# Get API Gateway URL from CloudFormation outputs
$apiUrl = aws cloudformation describe-stacks --stack-name fluidity-lambda `
  --query 'Stacks[0].Outputs[?OutputKey==`ApiGatewayUrl`].OutputValue' --output text

# Get API key
$apiKey = aws cloudformation describe-stacks --stack-name fluidity-lambda `
  --query 'Stacks[0].Outputs[?OutputKey==`ApiKeyId`].OutputValue' --output text

# Invoke Wake
$response = Invoke-WebRequest -Uri "$apiUrl/wake" `
  -Headers @{"x-api-key" = $apiKey} `
  -Method POST

Write-Host "Wake response: $($response.Content)"
```

```bash
#!/usr/bin/env bash
# Get API Gateway URL
API_URL=$(aws cloudformation describe-stacks --stack-name fluidity-lambda \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiGatewayUrl`].OutputValue' --output text)

# Get API key
API_KEY=$(aws cloudformation describe-stacks --stack-name fluidity-lambda \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiKeyId`].OutputValue' --output text)

# Invoke Wake
curl -X POST "$API_URL/wake" \
  -H "x-api-key: $API_KEY"
```

### Invoke Kill Lambda Directly

```powershell
$response = Invoke-WebRequest -Uri "$apiUrl/kill" `
  -Headers @{"x-api-key" = $apiKey} `
  -Method POST

Write-Host "Kill response: $($response.Content)"
```

```bash
curl -X POST "$API_URL/kill" \
  -H "x-api-key: $API_KEY"
```

### Check ECS Task Status

```powershell
# List tasks
aws ecs list-tasks --cluster fluidity --service-name fluidity-server

# Get task details
aws ecs describe-tasks --cluster fluidity `
  --tasks <TASK_ARN> `
  --query 'tasks[0].[lastStatus,desiredStatus]' --output text
```

```bash
# List tasks
aws ecs list-tasks --cluster fluidity --service-name fluidity-server

# Get task details
aws ecs describe-tasks --cluster fluidity \
  --tasks "$TASK_ARN" \
  --query 'tasks[0].[lastStatus,desiredStatus]' --output text
```

Expected output:
- `lastStatus=RUNNING, desiredStatus=RUNNING` → Server is running
- `lastStatus=STOPPED, desiredStatus=STOPPED` → Server is stopped
- `lastStatus=PROVISIONING, desiredStatus=RUNNING` → Server is starting
- `lastStatus=DEPROVISIONING, desiredStatus=STOPPED` → Server is stopping

---

## Monitoring and Alerts

### CloudWatch Dashboard

**Access:**

```powershell
# Get dashboard URL
aws cloudformation describe-stacks --stack-name fluidity-lambda `
  --query 'Stacks[0].Outputs[?OutputKey==`DashboardURL`].OutputValue' --output text
```

**Dashboard contents:**
- Lambda function invocations and errors
- Fargate task metrics (CPU, memory)
- Server active connections count
- Last activity timestamp
- API Gateway request counts

### CloudWatch Alarms

**View alarms:**

```powershell
aws cloudwatch describe-alarms --alarm-name-prefix fluidity
```

**Supported alarms:**
- `WakeLambdaErrorAlarm` - Wake Lambda failures
- `SleepLambdaErrorAlarm` - Sleep Lambda failures
- `KillLambdaErrorAlarm` - Kill Lambda failures

**Configure SNS notifications:**

```powershell
# Get SNS topic ARN
$topicArn = aws cloudformation describe-stacks --stack-name fluidity-lambda `
  --query 'Stacks[0].Outputs[?OutputKey==`AlarmNotificationTopicArn`].OutputValue' --output text

# Subscribe your email
aws sns subscribe --topic-arn $topicArn --protocol email --notification-endpoint your-email@example.com
```

### Check Lambda Logs

```powershell
# Get recent Wake Lambda logs (last 10 minutes)
aws logs tail /aws/lambda/fluidity-wake --follow --since 10m

# Get recent Sleep Lambda logs
aws logs tail /aws/lambda/fluidity-sleep --follow --since 10m

# Get recent Kill Lambda logs
aws logs tail /aws/lambda/fluidity-kill --follow --since 10m
```

```bash
# Bash
aws logs tail /aws/lambda/fluidity-wake --follow --since 10m
aws logs tail /aws/lambda/fluidity-sleep --follow --since 10m
aws logs tail /aws/lambda/fluidity-kill --follow --since 10m
```

### Check Fargate Logs

```powershell
# Get server logs (last hour)
aws logs tail /fluidity/server --follow --since 1h
```

```bash
aws logs tail /fluidity/server --follow --since 1h
```

---

## Troubleshooting

### Server Won't Start

**Symptom:** Wake Lambda succeeds but Fargate task stays in PROVISIONING

**Steps:**
1. Check Fargate service status:
   ```powershell
   aws ecs describe-services --cluster fluidity --services fluidity-server
   ```

2. Check task status:
   ```powershell
   $taskArn = aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text
   aws ecs describe-tasks --cluster fluidity --tasks $taskArn
   ```

3. Check CloudWatch Logs:
   ```powershell
   aws logs tail /fluidity/server --since 5m
   ```

**Common causes:**
- **Insufficient Fargate capacity:** Switch region or wait for capacity
- **VPC/subnet issues:** Verify security group and subnet configuration
- **Docker image pull failure:** Confirm ECR image exists and is accessible
- **Startup script errors:** Check logs for application startup failures

**Resolution:**
- Verify ECR image is pushed correctly
- Check Fargate quota for region
- Review security group rules allow port 8443
- Confirm IAM role has ECR pull permissions

### Agent Can't Connect

**Symptom:** Agent calls Wake, task starts, but agent times out connecting

**Steps:**
1. Verify server is listening:
   ```powershell
   $taskArn = aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' --output text
   aws ecs describe-tasks --cluster fluidity --tasks $taskArn --query 'tasks[0].containerInstanceArn' --output text
   ```

2. Check server logs:
   ```powershell
   aws logs tail /fluidity/server --since 5m | grep -i "listening\|error"
   ```

3. Check network connectivity:
   ```powershell
   $publicIp = # Get from ECS task description
   Test-NetConnection -ComputerName $publicIp -Port 8443 -InformationLevel Detailed
   ```

**Common causes:**
- **Server not listening on port 8443:** Check configuration
- **Security group blocks traffic:** Verify inbound rule for port 8443
- **Connection timeout too short:** Increase `connection_timeout` to 120s
- **mTLS certificate mismatch:** Verify CA, server cert, and keys match

**Resolution:**
```powershell
# Increase connection timeout in agent config
# Before: connection_timeout: "90s"
# After: connection_timeout: "120s"

# Verify certificates
openssl verify -CAfile ./certs/ca.crt ./certs/server.crt
openssl verify -CAfile ./certs/ca.crt ./certs/client.crt
```

### Lambda Invocation Fails

**Symptom:** CloudWatch alarm triggers, Lambda error in logs

**Steps:**
1. Check Lambda logs:
   ```powershell
   aws logs tail /aws/lambda/fluidity-wake --since 5m
   ```

2. Verify IAM permissions:
   ```powershell
   # Get Lambda execution role
   aws lambda get-function --function-name fluidity-wake --query 'Configuration.Role' --output text
   
   # Check attached policies
   aws iam list-attached-role-policies --role-name <ROLE_NAME>
   ```

3. Check ECS permissions:
   ```powershell
   # Verify cluster exists
   aws ecs describe-clusters --clusters fluidity
   
   # Verify service exists
   aws ecs describe-services --cluster fluidity --services fluidity-server
   ```

**Common causes:**
- **Missing ECS permissions:** Lambda role lacks `ecs:UpdateService`
- **Wrong cluster/service name:** Lambda config doesn't match deployed resources
- **Service doesn't exist:** Fargate stack not deployed

**Resolution:**
```powershell
# Verify Lambda config has correct cluster and service names
aws lambda get-function-configuration --function-name fluidity-wake --query 'Environment.Variables' --output json

# Example should show:
# "CLUSTER_NAME": "fluidity"
# "SERVICE_NAME": "fluidity-server"
```

### Idle Detection Not Working

**Symptom:** Server stays running even when idle for > 10 minutes

**Steps:**
1. Verify metrics are being emitted:
   ```powershell
   aws cloudwatch get-metric-statistics `
     --namespace Fluidity `
     --metric-name ActiveConnections `
     --start-time (Get-Date).AddHours(-1) `
     --end-time (Get-Date) `
     --period 300 `
     --statistics Average
   ```

2. Check Sleep Lambda logs:
   ```powershell
   aws logs tail /aws/lambda/fluidity-sleep --since 15m
   ```

3. Verify EventBridge rule is active:
   ```powershell
   aws events describe-rule --name fluidity-sleep-schedule
   ```

**Common causes:**
- **Metrics not enabled:** `emit_metrics: false` in server config
- **Sleep Lambda not running:** EventBridge rule disabled
- **Active connections always > 0:** Agent connected or lingering connection

**Resolution:**
```yaml
# Enable metrics in server config (configs/server.yaml)
emit_metrics: true
metrics_interval: "60s"

# Enable EventBridge rule
aws events enable-rule --name fluidity-sleep-schedule
```

### High Latency or Slow Connections

**Symptom:** Tunneled traffic is slow or has high latency

**Steps:**
1. Check agent to server connection:
   ```bash
   # In agent logs, look for connection establishment time
   # Example: "Connected to server in 45ms"
   ```

2. Check Lambda invocation time:
   ```powershell
   aws logs tail /aws/lambda/fluidity-wake --since 10m | grep "Duration:"
   ```

3. Monitor server metrics:
   ```powershell
   aws cloudwatch get-metric-statistics `
     --namespace Fluidity `
     --metric-name ActiveConnections `
     --start-time (Get-Date).AddHours(-1) `
     --end-time (Get-Date) `
     --period 60 `
     --statistics Average
   ```

**Common causes:**
- **Cold start latency:** Wake takes 3-5s, first request slower
- **High concurrent connections:** Fargate CPU/memory limited
- **Network latency:** Fargate region far from client
- **TLS handshake overhead:** Normal for new connections

**Resolution:**
- Use connection pooling to avoid new TLS handshakes
- Increase Fargate CPU/memory if load is high
- Use Fargate region closer to your location
- Accept initial cold start lag (resolved in ~30 seconds)

---

## Incident Response

### Complete Service Outage

**Symptom:** Server won't start, Lambda fails, no connectivity

**Emergency steps:**
1. Check AWS service status: https://status.aws.amazon.com/
2. Verify IAM permissions still valid
3. Check CloudWatch Logs for errors
4. Restart Fargate service:
   ```powershell
   aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0 --force-new-deployment
   Start-Sleep -Seconds 10
   aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 1 --force-new-deployment
   ```

5. If still failing, check CloudFormation stack status:
   ```powershell
   aws cloudformation describe-stacks --stack-name fluidity-lambda
   aws cloudformation describe-stacks --stack-name fluidity-fargate
   ```

### Connection Drops

**Immediate action:**
1. Restart agent:
   ```bash
   ./build/fluidity-agent --config ./configs/agent.yaml
   ```
   Agent will automatically re-wake server and reconnect.

2. If persists, check server health:
   ```powershell
   # Manually invoke wake
   # See "Invoke Wake Lambda Directly" section above
   ```

### Memory/CPU Issues

**Symptom:** Server crashes with OOM or task stops frequently

**Check current allocation:**
```powershell
aws ecs describe-task-definition --task-definition fluidity-server `
  --query 'taskDefinition.[cpu,memory]' --output text
```

**Increase resources:**
```powershell
# Current: 256 CPU, 512 MB memory
# New: 512 CPU, 1024 MB memory

$taskDef = aws ecs describe-task-definition --task-definition fluidity-server --output json
$taskDef.taskDefinition.cpu = "512"
$taskDef.taskDefinition.memory = "1024"

# Register new version and update service
aws ecs register-task-definition --cli-input-json ... 
aws ecs update-service --cluster fluidity --service fluidity-server --task-definition fluidity-server:2
```

---

## Maintenance

### Regular Health Checks

**Daily:**
- Check CloudWatch Alarms (SNS email)
- Spot check CloudWatch Dashboard
- Verify no Lambda errors in logs

**Weekly:**
- Test manual wake/kill procedures
- Review CloudWatch Logs for patterns
- Verify metrics are being emitted

**Monthly:**
- Review costs in AWS Billing
- Check for security updates to dependencies
- Test disaster recovery (redeploy from scratch)

### Update Docker Image

```bash
# 1. Build new image
make -f Makefile.win docker-build-server

# 2. Push to ECR
docker tag fluidity-server:latest <ACCOUNT>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest
docker push <ACCOUNT>.dkr.ecr.<REGION>.amazonaws.com/fluidity-server:latest

# 3. Force new deployment
aws ecs update-service --cluster fluidity --service fluidity-server --force-new-deployment
```

### Clean Up Resources

**Stop temporarily:**
```powershell
aws ecs update-service --cluster fluidity --service fluidity-server --desired-count 0
```

**Delete everything:**
```powershell
# Delete Lambda stack first (depends on outputs)
aws cloudformation delete-stack --stack-name fluidity-lambda

# Delete Fargate stack
aws cloudformation delete-stack --stack-name fluidity-fargate

# Wait for deletion
aws cloudformation wait stack-delete-complete --stack-name fluidity-fargate

# Clean up ECR (optional)
aws ecr delete-repository --repository-name fluidity-server --force
```

### Backup Configuration

```bash
# Backup Lambda stack parameters
aws cloudformation describe-stacks --stack-name fluidity-lambda > fluidity-lambda-backup.json

# Backup Fargate stack parameters
aws cloudformation describe-stacks --stack-name fluidity-fargate > fluidity-fargate-backup.json

# Backup logs
aws logs create-export-task --log-group-name /aws/lambda/fluidity-wake --from 0 --to $(date +%s)000 --destination fluidity-logs --destination-prefix backup/
```

---

## Quick Reference

### Essential AWS CLI Commands

```bash
# Check server status
aws ecs describe-services --cluster fluidity --services fluidity-server --query 'services[0].[status,runningCount,desiredCount]' --output text

# Get server public IP
aws ecs list-tasks --cluster fluidity --service-name fluidity-server --query 'taskArns[0]' | xargs -I {} aws ecs describe-tasks --cluster fluidity --tasks {} --query 'tasks[0].attachments[0].details[?name==networkInterfaceId].value' | xargs -I {} aws ec2 describe-network-interfaces --network-interface-ids {} --query 'NetworkInterfaces[0].Association.PublicIp'

# View recent logs
aws logs tail /fluidity/server --follow --since 5m
aws logs tail /aws/lambda/fluidity-wake --follow --since 5m

# Check alarms
aws cloudwatch describe-alarms --state-value ALARM

# Get CloudFormation outputs
aws cloudformation describe-stacks --stack-name fluidity-lambda --query 'Stacks[0].Outputs' --output table
```

---

## Related Documentation

- **[Lambda Functions](lambda.md)** - Complete Lambda setup and configuration
- **[Fargate Deployment](fargate.md)** - Cloud deployment guide
- **[Infrastructure as Code](infrastructure.md)** - CloudFormation templates
- **[Testing](testing.md)** - Test procedures and validation

