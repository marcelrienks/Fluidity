# Sleep Lambda Function

This Lambda function monitors the Fluidity ECS Fargate service for inactivity and scales it down to zero when idle for a configured period.

## Overview

The Sleep Lambda is triggered on a schedule (e.g., every 5 minutes) via EventBridge. It queries CloudWatch metrics to determine if the service is idle and scales it down if the idle threshold is exceeded.

## Behavior

1. **Service Already Stopped (DesiredCount=0)**: Returns status "no_change" (idempotent)
2. **Service Idle (no connections, exceeded idle threshold)**: Sets DesiredCount=0, returns status "scaled_down"
3. **Service Active (has connections or recent activity)**: Returns status "no_change"

## Idle Detection Logic

The service is considered idle when **both** conditions are met:
- Average active connections ≤ 0 (over lookback period)
- Time since last activity ≥ idle threshold

## Environment Variables

- `ECS_CLUSTER_NAME`: Name of the ECS cluster (required)
- `ECS_SERVICE_NAME`: Name of the ECS service (required)
- `IDLE_THRESHOLD_MINUTES`: Minutes of inactivity before scaling down (optional, default: 15)
- `LOOKBACK_PERIOD_MINUTES`: Minutes to look back for metrics (optional, default: 10)

## Request Format

```json
{
  "cluster_name": "optional-override",
  "service_name": "optional-override",
  "idle_threshold_mins": 30,
  "lookback_period_mins": 20
}
```

## Response Format

### Scaled Down
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

### No Change (Active)
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

### No Change (Already Stopped)
```json
{
  "action": "no_change",
  "desiredCount": 0,
  "runningCount": 0,
  "message": "Service is already stopped (desiredCount=0)"
}
```

## IAM Permissions Required

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:DescribeServices",
        "ecs:UpdateService"
      ],
      "Resource": [
        "arn:aws:ecs:*:*:service/*/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:GetMetricData"
      ],
      "Resource": "*"
    }
  ]
}
```

## CloudWatch Metrics Used

The Lambda queries these custom metrics from the Fluidity server:

1. **ActiveConnections** (Namespace: Fluidity)
   - Statistic: Average
   - Dimension: Service=fluidity-server
   - Purpose: Track concurrent connections

2. **LastActivityEpochSeconds** (Namespace: Fluidity)
   - Statistic: Maximum
   - Dimension: Service=fluidity-server
   - Purpose: Track time of last activity

## Building

### Local Development

```bash
cd cmd/lambdas/sleep
go mod download
go build -o bootstrap main.go
```

### For Lambda Deployment (Linux x86_64)

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip sleep-lambda.zip bootstrap
```

### For Lambda Deployment (ARM64)

```bash
# Build for ARM64 (Graviton2)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip sleep-lambda.zip bootstrap
```

## Testing

```bash
# Run unit tests
cd internal/lambdas/sleep
go test -v

# Run with coverage
go test -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Local Testing

You can test the Lambda function locally using the AWS SAM CLI:

```bash
# Create a test event
cat > event.json << EOF
{
  "cluster_name": "fluidity",
  "service_name": "fluidity-server",
  "idle_threshold_mins": 15,
  "lookback_period_mins": 10
}
EOF

# Invoke locally
sam local invoke SleepFunction -e event.json
```

## Deployment

### Via CloudFormation

The Sleep Lambda is deployed as part of the `lambda.yaml` CloudFormation stack:

```bash
aws cloudformation deploy \
  --template-file deployments/cloudformation/lambda.yaml \
  --stack-name fluidity-lambda \
  --parameter-overrides \
      EcsClusterName=fluidity \
      EcsServiceName=fluidity-server \
      IdleThresholdMinutes=15 \
      LookbackPeriodMinutes=10 \
  --capabilities CAPABILITY_IAM
```

### Manual Deployment

```bash
# Build and package
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
zip sleep-lambda.zip bootstrap

# Create Lambda function
aws lambda create-function \
  --function-name fluidity-sleep \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-cloudwatch-role \
  --handler bootstrap \
  --zip-file fileb://sleep-lambda.zip \
  --timeout 60 \
  --memory-size 256 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server,IDLE_THRESHOLD_MINUTES=15,LOOKBACK_PERIOD_MINUTES=10}"

# Update existing function
aws lambda update-function-code \
  --function-name fluidity-sleep \
  --zip-file fileb://sleep-lambda.zip
```

## EventBridge Schedule

The Sleep Lambda is typically triggered on a schedule:

```bash
# Create EventBridge rule (every 5 minutes)
aws events put-rule \
  --name fluidity-sleep-schedule \
  --schedule-expression "rate(5 minutes)" \
  --state ENABLED

# Add Lambda as target
aws events put-targets \
  --rule fluidity-sleep-schedule \
  --targets "Id=1,Arn=arn:aws:lambda:us-east-1:123456789012:function:fluidity-sleep"

# Grant EventBridge permission to invoke Lambda
aws lambda add-permission \
  --function-name fluidity-sleep \
  --statement-id AllowEventBridgeInvoke \
  --action lambda:InvokeFunction \
  --principal events.amazonaws.com \
  --source-arn arn:aws:events:us-east-1:123456789012:rule/fluidity-sleep-schedule
```

## Monitoring

### CloudWatch Logs

Logs are automatically sent to CloudWatch Logs:

```bash
# View logs
aws logs tail /aws/lambda/fluidity-sleep --follow

# View specific log stream
aws logs get-log-events \
  --log-group-name /aws/lambda/fluidity-sleep \
  --log-stream-name '2025/10/28/[$LATEST]...'
```

### Metrics

- **Invocations**: Number of times the function was invoked
- **Duration**: Execution time
- **Errors**: Failed invocations
- **Throttles**: Rate-limited invocations

### Custom Metrics

Consider tracking:
- Number of scale-downs performed
- Average idle duration when scaling down
- CloudWatch GetMetricData API calls

### Alarms

Consider setting up CloudWatch alarms for:
- High error rate (> 5% in 15 minutes)
- High duration (> 45 seconds)
- Consecutive failures (> 3 in a row)

## Performance

- **Cold start**: ~500ms - 1s (Go runtime is fast)
- **Warm execution**: ~200ms - 500ms (depends on CloudWatch API latency)
- **Timeout**: 60 seconds (configurable)
- **Memory**: 256 MB (sufficient for ECS and CloudWatch API calls)

## Cost

Based on 288 invocations per day (every 5 minutes):
- **Requests**: 8,640/month = $0.00173 (first 1M free)
- **Compute**: ~72 seconds/month = negligible (first 400,000 GB-seconds free)
- **CloudWatch GetMetricData**: 8,640 requests/month = $0.043 (beyond free tier)
- **Total**: ~$0.045/month (mostly CloudWatch API calls)

## Configuration Recommendations

### Development/Testing
- `IDLE_THRESHOLD_MINUTES=5` - Quick testing
- `LOOKBACK_PERIOD_MINUTES=5`
- Schedule: `rate(2 minutes)`

### Production
- `IDLE_THRESHOLD_MINUTES=15` - Balance between cost and UX
- `LOOKBACK_PERIOD_MINUTES=10` - Sufficient data points
- Schedule: `rate(5 minutes)` - Reasonable granularity

### Cost-Optimized
- `IDLE_THRESHOLD_MINUTES=30` - Longer idle period
- `LOOKBACK_PERIOD_MINUTES=15`
- Schedule: `rate(10 minutes)` - Fewer invocations

## Troubleshooting

### Function times out

- Check CloudWatch API latency (increase timeout to 90s if needed)
- Verify IAM permissions for CloudWatch GetMetricData
- Check if CloudWatch API is experiencing issues

### "Service not found" error

- Verify `ECS_CLUSTER_NAME` environment variable
- Verify `ECS_SERVICE_NAME` environment variable
- Check if service exists: `aws ecs describe-services --cluster ... --services ...`

### Service doesn't scale down when expected

- Check CloudWatch metrics: Are they being published?
  ```bash
  aws cloudwatch get-metric-data \
    --metric-data-queries file://metrics-query.json \
    --start-time 2025-10-28T10:00:00Z \
    --end-time 2025-10-28T11:00:00Z
  ```
- Verify idle threshold and lookback period configuration
- Check Lambda logs for decision logic
- Ensure server is emitting metrics correctly

### Service scales down too aggressively

- Increase `IDLE_THRESHOLD_MINUTES` (e.g., from 15 to 30)
- Increase schedule interval (e.g., from 5 to 10 minutes)
- Review metric definitions (are connections counted correctly?)

### CloudWatch GetMetricData returns no data

- Verify server is publishing metrics
- Check metric namespace and dimensions match
- Ensure server has CloudWatch PutMetricData permissions
- Verify time range is valid (not too far in the past)

## Best Practices

1. **Set reasonable idle thresholds**: Too short = frequent wake/sleep cycles, too long = unnecessary costs
2. **Monitor CloudWatch costs**: GetMetricData can add up with frequent queries
3. **Use metric batching**: Server should batch metric emissions to reduce API calls
4. **Handle metric gaps**: Service may not have published metrics if recently started
5. **Log all decisions**: Include metric values and calculations in logs for debugging
6. **Test thoroughly**: Verify behavior with various metric scenarios
7. **Coordinate with Wake Lambda**: Ensure they don't fight each other

## Edge Cases

### No Metrics Available
If CloudWatch returns no data (service recently started):
- Lambda assumes service is active (safe default)
- Returns "no_change" to avoid premature shutdown

### Service Stuck in PENDING
If service has DesiredCount=1 but RunningCount=0 for extended period:
- Sleep Lambda will not scale it down (respects DesiredCount)
- Operators should investigate why tasks aren't starting

### Race Condition with Wake
If Wake and Sleep run simultaneously:
- Both check current state first
- Last writer wins (ECS UpdateService is idempotent)
- Schedule Sleep and Wake appropriately to minimize overlap

### Partial Metric Data
If only one metric (connections OR last activity) is available:
- Lambda uses available data with safe defaults
- Missing last activity = assume recent activity
- Missing connections = assume zero connections

## Related Documentation

- [Wake Lambda](lambda-wake.md) - Service wake-up function
- [Architecture Design](architecture.md) - Lambda control plane architecture
- [Deployment Guide](deployment.md) - Full deployment instructions
- [Testing Guide](testing.md) - Lambda testing strategies
