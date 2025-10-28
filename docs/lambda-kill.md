# Kill Lambda Function

This Lambda function immediately terminates the Fluidity ECS Fargate service by setting the desired count to 0.

## Overview

The Kill Lambda is triggered when the Fluidity agent shuts down or via a scheduled EventBridge rule (e.g., nightly shutdown). It performs an immediate shutdown without any validation or state checks.

## Behavior

- **Always**: Sets DesiredCount=0 immediately, no questions asked
- **Idempotent**: Can be called multiple times safely
- **No Validation**: Does not check current state before shutdown

## Environment Variables

- `ECS_CLUSTER_NAME`: Name of the ECS cluster (required)
- `ECS_SERVICE_NAME`: Name of the ECS service (required)

## Request Format

```json
{
  "cluster_name": "optional-override",
  "service_name": "optional-override"
}
```

## Response Format

```json
{
  "status": "killed",
  "desiredCount": 0,
  "message": "Service shutdown initiated. ECS tasks will terminate immediately."
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
        "ecs:UpdateService"
      ],
      "Resource": [
        "arn:aws:ecs:*:*:service/*/*"
      ]
    }
  ]
}
```

## Building

### Local Development

```bash
cd cmd/lambdas/kill
go mod download
go build -o bootstrap main.go
```

### For Lambda Deployment (Linux x86_64)

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip kill-lambda.zip bootstrap
```

### For Lambda Deployment (ARM64)

```bash
# Build for ARM64 (Graviton2)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip kill-lambda.zip bootstrap
```

## Testing

```bash
# Run unit tests
cd internal/lambdas/kill
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
  "service_name": "fluidity-server"
}
EOF

# Invoke locally
sam local invoke KillFunction -e event.json
```

## Deployment

### Via CloudFormation

The Kill Lambda is deployed as part of the `lambda.yaml` CloudFormation stack:

```bash
aws cloudformation deploy \
  --template-file deployments/cloudformation/lambda.yaml \
  --stack-name fluidity-lambda \
  --parameter-overrides \
      EcsClusterName=fluidity \
      EcsServiceName=fluidity-server \
  --capabilities CAPABILITY_IAM
```

### Manual Deployment

```bash
# Build and package
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
zip kill-lambda.zip bootstrap

# Create Lambda function
aws lambda create-function \
  --function-name fluidity-kill \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-role \
  --handler bootstrap \
  --zip-file fileb://kill-lambda.zip \
  --timeout 30 \
  --memory-size 128 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server}"

# Update existing function
aws lambda update-function-code \
  --function-name fluidity-kill \
  --zip-file fileb://kill-lambda.zip
```

## API Gateway Integration

The Kill Lambda is typically exposed via API Gateway for agent shutdown:

```bash
# Create API Gateway REST API
aws apigateway create-rest-api \
  --name fluidity-api \
  --description "Fluidity Control Plane API"

# Create /kill resource
aws apigateway create-resource \
  --rest-api-id <api-id> \
  --parent-id <root-resource-id> \
  --path-part kill

# Create POST method
aws apigateway put-method \
  --rest-api-id <api-id> \
  --resource-id <kill-resource-id> \
  --http-method POST \
  --authorization-type AWS_IAM

# Integrate with Lambda
aws apigateway put-integration \
  --rest-api-id <api-id> \
  --resource-id <kill-resource-id> \
  --http-method POST \
  --type AWS_PROXY \
  --integration-http-method POST \
  --uri arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:fluidity-kill/invocations
```

## EventBridge Schedule

The Kill Lambda can be scheduled to run nightly for automatic shutdown:

```bash
# Create EventBridge rule (11 PM UTC daily)
aws events put-rule \
  --name fluidity-kill-schedule \
  --schedule-expression "cron(0 23 * * ? *)" \
  --state ENABLED

# Add Lambda as target
aws events put-targets \
  --rule fluidity-kill-schedule \
  --targets "Id=1,Arn=arn:aws:lambda:us-east-1:123456789012:function:fluidity-kill"

# Grant EventBridge permission to invoke Lambda
aws lambda add-permission \
  --function-name fluidity-kill \
  --statement-id AllowEventBridgeInvoke \
  --action lambda:InvokeFunction \
  --principal events.amazonaws.com \
  --source-arn arn:aws:events:us-east-1:123456789012:rule/fluidity-kill-schedule
```

## Monitoring

### CloudWatch Logs

Logs are automatically sent to CloudWatch Logs:

```bash
# View logs
aws logs tail /aws/lambda/fluidity-kill --follow

# View specific log stream
aws logs get-log-events \
  --log-group-name /aws/lambda/fluidity-kill \
  --log-stream-name '2025/10/28/[$LATEST]...'
```

### Metrics

- **Invocations**: Number of times the function was invoked
- **Duration**: Execution time
- **Errors**: Failed invocations
- **Throttles**: Rate-limited invocations

### Alarms

Consider setting up CloudWatch alarms for:
- Consecutive errors (> 3 in a row)
- High duration (> 20 seconds)

## Performance

- **Cold start**: ~500ms - 1s (Go runtime is fast)
- **Warm execution**: ~50ms - 150ms (simple ECS API call)
- **Timeout**: 30 seconds (more than sufficient)
- **Memory**: 128 MB (minimal requirements)

## Cost

Based on 10 invocations per day (2 agent shutdowns + 1 nightly kill × 5 weekdays):

- **Requests**: 300/month = $0.00006 (first 1M free)
- **Compute**: ~1 second/month = negligible (first 400,000 GB-seconds free)
- **Total**: Effectively free (within free tier)

If using API Gateway:
- **API Gateway requests**: 300/month = $0.0012 (first 1M free)
- **Still effectively free**

## Usage Patterns

### 1. Agent Shutdown (via API Gateway)

When the Fluidity agent stops gracefully:

```bash
curl -X POST https://api.example.com/kill \
  -H "x-api-key: your-api-key"
```

### 2. Manual Kill (via AWS CLI)

For manual emergency shutdown:

```bash
aws lambda invoke \
  --function-name fluidity-kill \
  --payload '{}' \
  response.json
```

### 3. Scheduled Kill (via EventBridge)

Automatic nightly shutdown at 11 PM UTC to save costs.

## Troubleshooting

### Function times out

- Check IAM permissions for ECS UpdateService
- Verify cluster and service names are correct
- Check if ECS API is experiencing issues

### "Access Denied" error

- Verify Lambda execution role has `ecs:UpdateService` permission
- Check service resource ARN in IAM policy
- Ensure API Gateway uses correct IAM role/API key

### Function succeeds but service doesn't stop

- Check ECS service events: `aws ecs describe-services ...`
- Verify service name and cluster name are correct
- Review CloudWatch Logs for any warnings

### Service restarts after kill

- Check if Sleep Lambda is running too frequently
- Verify EventBridge rules aren't conflicting
- Ensure no other automation is starting the service

## Best Practices

1. **Use for graceful shutdowns**: Call Kill when agent stops cleanly
2. **Schedule nightly kills**: Save costs by stopping service overnight
3. **Monitor invocations**: Track kill frequency to understand usage
4. **Protect API endpoints**: Use API keys or IAM authentication
5. **Test idempotency**: Verify multiple kills don't cause issues
6. **Log all kills**: Track who/what triggered the shutdown
7. **Coordinate with Wake**: Ensure Wake isn't immediately restarting

## Edge Cases

### Kill During Wake

If Kill is called while Wake is starting the service:
- **Last writer wins**: ECS UpdateService is atomic
- **Service stops**: Kill sets desiredCount=0
- **Agent fails to connect**: Agent will retry and trigger Wake again
- **Mitigation**: Coordinate timing or use distributed locks

### Kill During Sleep Evaluation

If Kill runs while Sleep is evaluating metrics:
- **Both may call UpdateService**: Both set desiredCount=0
- **Result**: Service stops (correct outcome)
- **Impact**: Two API calls instead of one (minor)

### Rapid Wake/Kill Cycles

If agent frequently starts and stops:
- **ECS throttling**: May hit rate limits
- **Unnecessary costs**: Frequent task start/stop
- **Mitigation**: Add backoff, consolidate starts/stops

### Service Already Stopped

If service is already at desiredCount=0:
- **Kill still succeeds**: UpdateService is idempotent
- **No harm done**: Setting 0→0 is safe
- **Expected behavior**: Kill doesn't validate first

## Security Considerations

### API Gateway Protection

If exposing via API Gateway:
- **Use API keys**: Prevent unauthorized kills
- **Rate limiting**: Prevent DoS attacks
- **CloudWatch alarms**: Alert on unusual activity
- **IP whitelisting**: Restrict to known agent IPs

### Lambda Permissions

- **Least privilege**: Only `ecs:UpdateService`, nothing more
- **Resource restrictions**: Scope to specific cluster/service
- **No DescribeServices**: Kill doesn't need read access

### Audit Logging

- **CloudTrail**: Track ECS UpdateService API calls
- **Custom metrics**: Count kills per hour/day
- **Alerting**: Notify on unexpected kills

## Related Documentation

- [Wake Lambda](lambda-wake.md) - Service wake-up function
- [Sleep Lambda](lambda-sleep.md) - Idle detection and scale-down
- [Architecture Design](architecture.md) - Lambda control plane architecture
- [Deployment Guide](deployment.md) - Full deployment instructions
- [Testing Guide](testing.md) - Lambda testing strategies
