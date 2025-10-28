# Wake Lambda Function

This Lambda function wakes up the Fluidity ECS Fargate service by setting the desired count to 1.

## Overview

The Wake Lambda is triggered when the Fluidity agent starts up and needs to connect to the server. It checks the current state of the ECS service and starts it if it's stopped.

## Behavior

1. **Service Stopped (DesiredCount=0)**: Sets DesiredCount=1, returns status "waking"
2. **Service Starting (DesiredCount=1, RunningCount=0)**: Returns status "starting" (idempotent)
3. **Service Running (DesiredCount=1, RunningCount>0)**: Returns status "already_running" (idempotent)

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
  "status": "waking|starting|already_running",
  "desiredCount": 1,
  "runningCount": 0,
  "pendingCount": 0,
  "estimatedStartTime": "2025-10-28T12:00:00Z",
  "message": "Service wake initiated. ECS task starting (estimated 60-90 seconds)"
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
    }
  ]
}
```

## Building

### Local Development

```bash
cd deployments/lambda/wake
go mod download
go build -o bootstrap main.go
```

### For Lambda Deployment (Linux)

```bash
# Build for Linux (Lambda runtime)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip wake-lambda.zip bootstrap
```

### For Lambda Deployment (ARM64)

```bash
# Build for ARM64 (Graviton2)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go

# Create deployment package
zip wake-lambda.zip bootstrap
```

## Testing

```bash
# Run unit tests
go test -v

# Run with coverage
go test -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Local Testing

You can test the Lambda function locally using the AWS SAM CLI:

```bash
# Install AWS SAM CLI first
# https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html

# Create a test event
cat > event.json << EOF
{
  "cluster_name": "fluidity",
  "service_name": "fluidity-server"
}
EOF

# Invoke locally
sam local invoke -e event.json
```

## Deployment

### Via CloudFormation

The Wake Lambda is deployed as part of the `lambda.yaml` CloudFormation stack:

```bash
aws cloudformation deploy \
  --template-file ../cloudformation/lambda.yaml \
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
zip wake-lambda.zip bootstrap

# Create Lambda function
aws lambda create-function \
  --function-name fluidity-wake \
  --runtime provided.al2 \
  --role arn:aws:iam::123456789012:role/lambda-ecs-role \
  --handler bootstrap \
  --zip-file fileb://wake-lambda.zip \
  --timeout 30 \
  --memory-size 256 \
  --environment Variables="{ECS_CLUSTER_NAME=fluidity,ECS_SERVICE_NAME=fluidity-server}"

# Update existing function
aws lambda update-function-code \
  --function-name fluidity-wake \
  --zip-file fileb://wake-lambda.zip
```

## Monitoring

### CloudWatch Logs

Logs are automatically sent to CloudWatch Logs:

```bash
# View logs
aws logs tail /aws/lambda/fluidity-wake --follow

# View specific log stream
aws logs get-log-events \
  --log-group-name /aws/lambda/fluidity-wake \
  --log-stream-name '2025/10/28/[$LATEST]...'
```

### Metrics

- **Invocations**: Number of times the function was invoked
- **Duration**: Execution time
- **Errors**: Failed invocations
- **Throttles**: Rate-limited invocations

### Alarms

Consider setting up CloudWatch alarms for:
- High error rate (> 5% in 5 minutes)
- High duration (> 20 seconds)
- Throttling events

## Performance

- **Cold start**: ~500ms - 1s (Go runtime is fast)
- **Warm execution**: ~100ms - 300ms
- **Timeout**: 30 seconds (configurable)
- **Memory**: 256 MB (sufficient for ECS API calls)

## Cost

Based on 10 invocations per day (2 per workday):
- **Requests**: 300/month = $0.00006 (first 1M free)
- **Compute**: ~2 seconds/month = negligible (first 400,000 GB-seconds free)
- **Total**: Effectively free (within free tier)

## Troubleshooting

### Function times out

- Check IAM permissions for ECS API calls
- Verify cluster and service names are correct
- Check if ECS API is experiencing issues

### "Service not found" error

- Verify `ECS_CLUSTER_NAME` environment variable
- Verify `ECS_SERVICE_NAME` environment variable
- Check if service exists: `aws ecs describe-services --cluster ... --services ...`

### Function succeeds but service doesn't start

- Check ECS service events: `aws ecs describe-services ...`
- Verify task definition is valid
- Check if service has sufficient capacity/quotas
- Review CloudWatch Logs for ECS task failures

## Related Documentation

- [Architecture Design](../../docs/architecture.md) - Lambda control plane architecture
- [Deployment Guide](../../docs/deployment.md) - Full deployment instructions
- [Testing Guide](../../docs/testing.md) - Lambda testing strategies
