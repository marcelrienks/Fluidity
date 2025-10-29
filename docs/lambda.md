# Lambda Control Plane Functions

This document describes the three Lambda functions that manage the Fluidity ECS Fargate service lifecycle: Wake, Sleep, and Kill.

---

## Table of Contents

- [Overview](#overview)
- [Wake Lambda](#wake-lambda)
- [Sleep Lambda](#sleep-lambda)
- [Kill Lambda](#kill-lambda)
- [Building & Deployment](#building--deployment)
- [Monitoring](#monitoring)
- [Cost Analysis](#cost-analysis)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

---

## Overview

The Lambda control plane provides automated lifecycle management for the Fluidity ECS service:

- **Wake Lambda**: Starts the ECS service when the agent needs connectivity
- **Sleep Lambda**: Stops the service after a period of inactivity to save costs
- **Kill Lambda**: Immediately stops the service on demand or schedule

### Architecture

```
Agent Startup → Wake Lambda → ECS DesiredCount=1 → Server Starts
                     ↓
               Agent Connects (retry up to 90s)

EventBridge (every 5 min) → Sleep Lambda → Check CloudWatch Metrics
                                                 ↓
                                          If idle → ECS DesiredCount=0

Agent Shutdown → Kill Lambda → ECS DesiredCount=0 (immediate)

EventBridge (daily 11 PM) → Kill Lambda → ECS DesiredCount=0
```

### Common Configuration

All three functions share common environment variables:

- `ECS_CLUSTER_NAME`: Name of the ECS cluster (required)
- `ECS_SERVICE_NAME`: Name of the ECS service (required)

---

## Wake Lambda

### Purpose

Starts the ECS service when the agent needs to connect, ensuring the server is available.

### Behavior

| Current State | Action | Response Status |
|--------------|--------|----------------|
| DesiredCount=0, RunningCount=0 | Set DesiredCount=1 | `waking` |
| DesiredCount=1, RunningCount=0 | No change | `starting` |
| DesiredCount=1, RunningCount>0 | No change | `already_running` |

### Triggers

- Agent startup via API Gateway `/wake` endpoint
- Manual invocation for testing

### Request/Response

**Request:**
```json
{
  "cluster_name": "optional-override",
  "service_name": "optional-override"
}
```

**Response:**
```json
{
  "status": "waking|starting|already_running",
  "desiredCount": 1,
  "runningCount": 0,
  "pendingCount": 0,
  "estimatedStartTime": "2025-10-29T12:00:00Z",
  "message": "Service wake initiated. ECS task starting (estimated 60-90 seconds)"
}
```

### IAM Permissions

```json
{
  "Effect": "Allow",
  "Action": [
    "ecs:DescribeServices",
    "ecs:UpdateService"
  ],
  "Resource": "arn:aws:ecs:*:*:service/*/*"
}
```

### Configuration

- **Timeout**: 30 seconds
- **Memory**: 256 MB
- **Runtime**: Go (provided.al2)

---

## Sleep Lambda

### Purpose

Monitors CloudWatch metrics and automatically scales down the ECS service when idle to minimize costs.

### Behavior

| Condition | Action | Response Status |
|-----------|--------|----------------|
| DesiredCount=0 | No change | `no_change` (already stopped) |
| Idle threshold exceeded & no connections | Set DesiredCount=0 | `scaled_down` |
| Recent activity or active connections | No change | `no_change` (active) |

### Idle Detection Logic

Service is considered idle when **BOTH** conditions are met:
- Average active connections ≤ 0 (over lookback period)
- Time since last activity ≥ idle threshold

### Triggers

- EventBridge scheduled rule (default: every 5 minutes)
- Manual invocation for testing

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `IDLE_THRESHOLD_MINUTES` | 15 | Minutes of inactivity before scaling down |
| `LOOKBACK_PERIOD_MINUTES` | 10 | Minutes to look back for metrics |

### Request/Response

**Request:**
```json
{
  "cluster_name": "optional-override",
  "service_name": "optional-override",
  "idle_threshold_mins": 30,
  "lookback_period_mins": 20
}
```

**Response (Scaled Down):**
```json
{
  "action": "scaled_down",
  "desiredCount": 0,
  "runningCount": 1,
  "avgActiveConnections": 0.0,
  "idleDurationSeconds": 1200,
  "message": "Service scaled down due to inactivity (idle for 1200 seconds)"
}
```

**Response (Active):**
```json
{
  "action": "no_change",
  "desiredCount": 1,
  "runningCount": 1,
  "avgActiveConnections": 2.5,
  "idleDurationSeconds": 120,
  "message": "Service is active (avg connections: 2.50, idle: 120 seconds)"
}
```

### CloudWatch Metrics

Queries these custom metrics from the Fluidity server (Namespace: `Fluidity`):

1. **ActiveConnections**
   - Statistic: Average
   - Dimension: Service=fluidity-server
   - Purpose: Track concurrent connections

2. **LastActivityEpochSeconds**
   - Statistic: Maximum
   - Dimension: Service=fluidity-server
   - Purpose: Track time of last activity

### IAM Permissions

```json
{
  "Effect": "Allow",
  "Action": [
    "ecs:DescribeServices",
    "ecs:UpdateService",
    "cloudwatch:GetMetricData"
  ],
  "Resource": ["arn:aws:ecs:*:*:service/*/*", "*"]
}
```

### Configuration

- **Timeout**: 60 seconds
- **Memory**: 256 MB
- **Runtime**: Go (provided.al2)

### Configuration Recommendations

| Environment | Idle Threshold | Lookback Period | Schedule |
|-------------|---------------|----------------|----------|
| Development | 5 minutes | 5 minutes | Every 2 minutes |
| Production | 15 minutes | 10 minutes | Every 5 minutes |
| Cost-Optimized | 30 minutes | 15 minutes | Every 10 minutes |

---

## Kill Lambda

### Purpose

Immediately terminates the ECS service without validation, used for graceful agent shutdown or scheduled overnight stops.

### Behavior

- **Always**: Sets DesiredCount=0 immediately
- **Idempotent**: Safe to call multiple times
- **No Validation**: Does not check current state

### Triggers

- Agent shutdown via API Gateway `/kill` endpoint
- EventBridge scheduled rule (default: 11 PM UTC daily)
- Manual invocation for emergency shutdown

### Request/Response

**Request:**
```json
{
  "cluster_name": "optional-override",
  "service_name": "optional-override"
}
```

**Response:**
```json
{
  "status": "killed",
  "desiredCount": 0,
  "message": "Service shutdown initiated. ECS tasks will terminate immediately."
}
```

### IAM Permissions

```json
{
  "Effect": "Allow",
  "Action": ["ecs:UpdateService"],
  "Resource": "arn:aws:ecs:*:*:service/*/*"
}
```

### Configuration

- **Timeout**: 30 seconds
- **Memory**: 128 MB (minimal)
- **Runtime**: Go (provided.al2)

### Usage Patterns

1. **Agent Shutdown**: Automatic call when agent stops gracefully
2. **Manual Kill**: Emergency shutdown via AWS CLI
3. **Scheduled Kill**: Nightly cost-saving shutdown

---

## Building & Deployment

### Build All Functions

```bash
# Navigate to each function directory
cd cmd/lambdas/wake
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
zip wake-lambda.zip bootstrap

cd ../sleep
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
zip sleep-lambda.zip bootstrap

cd ../kill
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
zip kill-lambda.zip bootstrap
```

### Deploy via CloudFormation (Recommended)

```bash
aws cloudformation deploy \
  --template-file deployments/cloudformation/lambda.yaml \
  --stack-name fluidity-lambda \
  --parameter-overrides \
      EcsClusterName=fluidity \
      EcsServiceName=fluidity-server \
      IdleThresholdMinutes=15 \
      LookbackPeriodMinutes=10 \
      SleepSchedule="rate(5 minutes)" \
      KillSchedule="cron(0 23 * * ? *)" \
  --capabilities CAPABILITY_IAM
```

### Manual Deployment

**Wake Lambda:**
```bash
aws lambda create-function \
  --function-name fluidity-wake \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-role \
  --handler bootstrap \
  --zip-file fileb://wake-lambda.zip \
  --timeout 30 \
  --memory-size 256 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server}"
```

**Sleep Lambda:**
```bash
aws lambda create-function \
  --function-name fluidity-sleep \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-cloudwatch-role \
  --handler bootstrap \
  --zip-file fileb://sleep-lambda.zip \
  --timeout 60 \
  --memory-size 256 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server,IDLE_THRESHOLD_MINUTES=15,LOOKBACK_PERIOD_MINUTES=10}"
```

**Kill Lambda:**
```bash
aws lambda create-function \
  --function-name fluidity-kill \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-role \
  --handler bootstrap \
  --zip-file fileb://kill-lambda.zip \
  --timeout 30 \
  --memory-size 128 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server}"
```

### EventBridge Schedules

**Sleep Schedule (every 5 minutes):**
```bash
aws events put-rule \
  --name fluidity-sleep-schedule \
  --schedule-expression "rate(5 minutes)" \
  --state ENABLED

aws events put-targets \
  --rule fluidity-sleep-schedule \
  --targets "Id=1,Arn=arn:aws:lambda:REGION:ACCOUNT:function:fluidity-sleep"
```

**Kill Schedule (11 PM UTC daily):**
```bash
aws events put-rule \
  --name fluidity-kill-schedule \
  --schedule-expression "cron(0 23 * * ? *)" \
  --state ENABLED

aws events put-targets \
  --rule fluidity-kill-schedule \
  --targets "Id=1,Arn=arn:aws:lambda:REGION:ACCOUNT:function:fluidity-kill"
```

---

## Monitoring

### CloudWatch Logs

View logs for each function:

```bash
# Wake Lambda
aws logs tail /aws/lambda/fluidity-wake --follow

# Sleep Lambda
aws logs tail /aws/lambda/fluidity-sleep --follow

# Kill Lambda
aws logs tail /aws/lambda/fluidity-kill --follow
```

### CloudWatch Metrics

All functions emit standard Lambda metrics:

- **Invocations**: Total number of invocations
- **Duration**: Execution time (ms)
- **Errors**: Failed invocations
- **Throttles**: Rate-limited invocations
- **ConcurrentExecutions**: Number of concurrent invocations

### Recommended Alarms

**Wake Lambda:**
- Error rate > 5% in 5 minutes
- Duration > 20 seconds
- Throttling events

**Sleep Lambda:**
- Error rate > 5% in 15 minutes
- Duration > 45 seconds
- Consecutive failures > 3

**Kill Lambda:**
- Consecutive errors > 3
- Duration > 20 seconds

### Custom Dashboards

Create a CloudWatch dashboard to monitor all three functions plus ECS service status:

```bash
aws cloudwatch put-dashboard \
  --dashboard-name fluidity-lambda \
  --dashboard-body file://dashboard.json
```

---

## Cost Analysis

### Monthly Cost Breakdown

| Function | Invocations/Month | Compute Time | Requests Cost | Compute Cost | Total |
|----------|------------------|--------------|---------------|--------------|-------|
| Wake | 300 (10/day) | 2s | $0.00006 | ~$0 | ~$0 |
| Sleep | 8,640 (288/day) | 72s | $0.00173 | ~$0 | $0.043* |
| Kill | 300 (10/day) | 1s | $0.00006 | ~$0 | ~$0 |
| **Total** | **9,240** | **75s** | **$0.00185** | **~$0** | **~$0.045** |

\* Sleep Lambda cost is primarily CloudWatch GetMetricData API calls ($0.043/month for 8,640 requests)

### Combined Infrastructure Costs

| Scenario | Fargate ($/month) | Lambda ($/month) | Total ($/month) |
|----------|------------------|------------------|-----------------|
| 2 hours/day active | $0.50 | $0.05 | **$0.55** |
| 8 hours/day active | $3.00 | $0.05 | **$3.05** |
| 24/7 (no automation) | $9.00 | $0.00 | **$9.00** |

**Savings**: Lambda control plane reduces costs by 70-94% compared to 24/7 operation.

### Cost Optimization Tips

1. **Adjust Sleep schedule**: Change from 5 to 10 minutes reduces invocations by 50%
2. **Increase idle threshold**: Longer threshold = fewer scale-downs = fewer Wake calls
3. **Use ARM64**: Graviton2 functions cost 20% less
4. **Batch metrics**: Server should batch CloudWatch metric emissions

---

## Troubleshooting

### Common Issues

#### Function Timeout

**Symptoms**: Lambda execution exceeds timeout limit

**Solutions:**
- Check AWS service health (ECS/CloudWatch API issues)
- Verify IAM permissions are correct
- Increase timeout (Wake/Kill: 30→60s, Sleep: 60→90s)
- Check CloudWatch logs for slow API calls

#### "Service not found" Error

**Symptoms**: Lambda reports ECS service doesn't exist

**Solutions:**
- Verify `ECS_CLUSTER_NAME` environment variable
- Verify `ECS_SERVICE_NAME` environment variable
- Check service exists: `aws ecs describe-services --cluster X --services Y`
- Ensure Lambda has permission to access the cluster

#### Service Doesn't Start After Wake

**Symptoms**: Wake succeeds but service stays at RunningCount=0

**Solutions:**
- Check ECS service events: `aws ecs describe-services`
- Verify task definition is valid
- Check Fargate capacity/quotas in region
- Review ECS task CloudWatch logs for startup errors

#### Service Doesn't Scale Down

**Symptoms**: Sleep Lambda runs but service stays at DesiredCount=1

**Solutions:**
- Verify server is emitting CloudWatch metrics
- Check metric namespace and dimensions match
- Review Sleep Lambda logs for decision logic
- Confirm idle threshold is appropriate for usage pattern
- Ensure server has CloudWatch PutMetricData permissions

#### Rapid Wake/Kill Cycles

**Symptoms**: Service constantly starting and stopping

**Solutions:**
- Check for race conditions between Wake and Sleep
- Increase idle threshold (15→30 minutes)
- Reduce Sleep schedule frequency (5→10 minutes)
- Review agent connection retry logic

### Debug Mode

Enable verbose logging in Lambda functions:

```bash
# Set log level environment variable
aws lambda update-function-configuration \
  --function-name fluidity-wake \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server,LOG_LEVEL=DEBUG}"
```

### Local Testing

Test functions locally using AWS SAM CLI:

```bash
# Create test event
cat > event.json << EOF
{
  "cluster_name": "fluidity",
  "service_name": "fluidity-server"
}
EOF

# Test Wake Lambda
sam local invoke -e event.json WakeFunction

# Test Sleep Lambda
sam local invoke -e event.json SleepFunction

# Test Kill Lambda
sam local invoke -e event.json KillFunction
```

---

## Best Practices

### General

1. **Use CloudFormation**: Deploy all functions together for consistency
2. **Enable X-Ray tracing**: Better debugging and performance analysis
3. **Set up alarms**: Monitor errors and performance degradation
4. **Version functions**: Use aliases for staged deployments
5. **Test locally**: Use SAM CLI before deploying to AWS
6. **Document changes**: Track configuration changes in version control

### Wake Lambda

1. **Agent retry logic**: Agent should retry connection for 60-90 seconds
2. **Idempotent design**: Multiple wake calls should be safe
3. **Monitor invocations**: Track frequency to detect issues

### Sleep Lambda

1. **Tune thresholds**: Balance cost savings vs. user experience
2. **Monitor CloudWatch costs**: GetMetricData can add up
3. **Handle missing metrics**: Assume active if metrics unavailable
4. **Log decisions**: Include metric values in logs for debugging
5. **Coordinate with Wake**: Ensure schedules don't conflict

### Kill Lambda

1. **Graceful shutdown**: Always call from agent before exiting
2. **Protect endpoints**: Use API keys or IAM authentication
3. **Scheduled kills**: Use for overnight cost savings
4. **Emergency use**: Keep as manual option for immediate shutdown
5. **Audit logging**: Track all kill invocations

### Security

1. **Least privilege IAM**: Only grant necessary permissions
2. **Encrypt logs**: Enable CloudWatch Logs encryption
3. **API Gateway auth**: Use API keys or IAM SigV4
4. **Rate limiting**: Prevent abuse via API Gateway throttling
5. **Resource scoping**: Limit IAM permissions to specific cluster/service

### Edge Cases

#### No Metrics Available (Sleep)
- **Behavior**: Assume service is active (safe default)
- **Action**: Return "no_change" to avoid premature shutdown

#### Service Stuck in PENDING
- **Behavior**: Sleep respects DesiredCount, won't scale down
- **Action**: Operators investigate why tasks aren't starting

#### Wake/Sleep Race Condition
- **Behavior**: Last writer wins (ECS UpdateService is atomic)
- **Mitigation**: Schedule appropriately to minimize overlap

#### Kill During Wake
- **Behavior**: Service stops (Kill sets DesiredCount=0)
- **Impact**: Agent fails to connect, triggers Wake again
- **Mitigation**: Coordinate timing or use distributed locks

---

## Related Documentation

- [Architecture Design](architecture.md) - Lambda control plane architecture
- [Deployment Guide](deployment.md) - Full deployment instructions with Lambda integration
- [Testing Guide](testing.md) - Lambda testing strategies
- [Fargate Deployment](fargate.md) - ECS Fargate setup

---

## Summary

The Lambda control plane provides:

✅ **Automated lifecycle management** - Start/stop based on demand  
✅ **Cost optimization** - 70-94% savings vs. 24/7 operation  
✅ **Operational simplicity** - No manual ECS management  
✅ **High reliability** - Idempotent operations, graceful handling  
✅ **Low overhead** - ~$0.05/month in Lambda costs

**Total monthly cost**: $0.55-$3.05 (vs. $9 for manual 24/7 operation)
