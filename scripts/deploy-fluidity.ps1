#!/usr/bin/env pwsh
<#
.SYNOPSIS
Deploy Fluidity infrastructure to AWS using CloudFormation

.DESCRIPTION
This script deploys the complete Fluidity stack including:
- Fargate ECS cluster and service
- Lambda control plane (Wake, Sleep, Kill functions)
- API Gateway with authentication
- EventBridge schedules
- CloudWatch monitoring

Parameters can be provided via command-line arguments (recommended) or a parameters file.

.PARAMETER Action
The action to perform:
- deploy: Create or update the stack
- delete: Delete the stack
- status: Show stack status
- outputs: Show stack outputs
Default: deploy

.PARAMETER StackName
The CloudFormation stack name. Default: fluidity

.PARAMETER ParametersFile
Path to parameters JSON file. Only used if command-line parameters are not provided.
Default: params.json in cloudformation directory

.PARAMETER Capabilities
CloudFormation capabilities. Default: CAPABILITY_NAMED_IAM

.PARAMETER Force
Skip confirmation prompts (for delete action)

.PARAMETER AccountId
[OPTIONAL if AWS CLI configured] AWS Account ID (12 digits).
If not provided, will be automatically detected from your configured AWS credentials.
Get value: aws sts get-caller-identity --query Account --output text

.PARAMETER Region
[REQUIRED] AWS Region (e.g., us-east-1, eu-west-1).

.PARAMETER VpcId
[REQUIRED] VPC ID (e.g., vpc-abc123def456789a).
Get value: aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text

.PARAMETER PublicSubnets
[REQUIRED] Comma-separated list of public subnet IDs (minimum 2 in different AZs).
Get value: aws ec2 describe-subnets --filters Name=vpc-id,Values=<VPC_ID> --query 'Subnets[*].SubnetId' --output text

.PARAMETER AllowedIngressCidr
[REQUIRED] Allowed ingress CIDR for security group (e.g., 1.2.3.4/32).
Get value: (Invoke-WebRequest -Uri 'https://ifconfig.me' -UseBasicParsing).Content.Trim() + '/32'

.PARAMETER ContainerImage
[OPTIONAL] Container image URI. If not provided, defaults to: <AccountId>.dkr.ecr.<Region>.amazonaws.com/fluidity-server:latest

.PARAMETER ClusterName
[OPTIONAL] ECS cluster name. Default: fluidity

.PARAMETER ServiceName
[OPTIONAL] ECS service name. Default: fluidity-server

.PARAMETER ContainerPort
[OPTIONAL] Container port. Default: 8443

.PARAMETER Cpu
[OPTIONAL] Task CPU units (256 = 0.25 vCPU, 512 = 0.5 vCPU, 1024 = 1 vCPU). Default: 256

.PARAMETER Memory
[OPTIONAL] Task memory in MB. Default: 512

.PARAMETER DesiredCount
[OPTIONAL] Initial desired task count (0 = stopped, use Lambda to wake). Default: 0

.PARAMETER LogGroupName
[OPTIONAL] CloudWatch log group name. Default: /ecs/fluidity/server

.PARAMETER LogRetentionDays
[OPTIONAL] CloudWatch log retention in days. Default: 30

.PARAMETER IdleThresholdMinutes
[OPTIONAL] Minutes of idle before auto-sleep. Default: 15

.PARAMETER LookbackPeriodMinutes
[OPTIONAL] CloudWatch metrics lookback period. Default: 10

.PARAMETER SleepCheckIntervalMinutes
[OPTIONAL] How often Sleep Lambda checks for idle state. Default: 5

.PARAMETER DailyKillTime
[OPTIONAL] Daily shutdown cron expression. Default: cron(0 23 * * ? *) [11 PM UTC]

.PARAMETER WakeLambdaTimeout
[OPTIONAL] Wake Lambda timeout in seconds. Default: 30

.PARAMETER SleepLambdaTimeout
[OPTIONAL] Sleep Lambda timeout in seconds. Default: 60

.PARAMETER KillLambdaTimeout
[OPTIONAL] Kill Lambda timeout in seconds. Default: 30

.PARAMETER APIThrottleBurstLimit
[OPTIONAL] API Gateway burst requests. Default: 20

.PARAMETER APIThrottleRateLimit
[OPTIONAL] API Gateway requests per second. Default: 3

.PARAMETER APIQuotaLimit
[OPTIONAL] API Gateway monthly quota. Default: 300

.EXAMPLE
PS> .\deploy-fluidity.ps1 -Action deploy -Region us-east-1 -VpcId vpc-abc123 -PublicSubnets subnet-1,subnet-2 -AllowedIngressCidr 1.2.3.4/32
Deploy stack with required parameters (AccountId auto-detected from AWS credentials)

.EXAMPLE
PS> $VPC_ID = aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text
PS> $SUBNETS = (aws ec2 describe-subnets --filters Name=vpc-id,Values=$VPC_ID --query 'Subnets[*].SubnetId' --output text) -replace '\s+', ','
PS> $MY_IP = (Invoke-WebRequest -Uri 'https://ifconfig.me' -UseBasicParsing).Content.Trim() + '/32'
PS> .\deploy-fluidity.ps1 -Action deploy -Region us-east-1 -VpcId $VPC_ID -PublicSubnets $SUBNETS -AllowedIngressCidr $MY_IP
Quick deploy using AWS CLI to gather required parameter values

.EXAMPLE
PS> .\deploy-fluidity.ps1 -Action deploy -Region us-east-1 -VpcId vpc-abc123 -PublicSubnets subnet-1,subnet-2 -AllowedIngressCidr 1.2.3.4/32 -Cpu 512 -Memory 1024 -DesiredCount 1
Deploy with custom optional parameters (override defaults)

.EXAMPLE
PS> .\deploy-fluidity.ps1 -Action status
Check stack status

.EXAMPLE
PS> .\deploy-fluidity.ps1 -Action delete -Force
Delete stack without confirmation

.EXAMPLE
PS> .\deploy-fluidity.ps1 -Action outputs
View stack outputs
#>

param(
    [ValidateSet('deploy', 'delete', 'status', 'outputs')]
    [string]$Action = 'deploy',
    
    [string]$StackName = "fluidity",
    
    [string]$ParametersFile,
    
    [string]$Capabilities = 'CAPABILITY_NAMED_IAM',
    
    [switch]$Force,
    
    # AWS Configuration
    [string]$AccountId,
    [string]$Region,
    [string]$VpcId,
    [string]$PublicSubnets,
    [string]$AllowedIngressCidr,
    
    # Container Configuration
    [string]$ContainerImage,
    [string]$ClusterName = "fluidity",
    [string]$ServiceName = "fluidity-server",
    [string]$ContainerPort = "8443",
    [string]$Cpu = "256",
    [string]$Memory = "512",
    [string]$DesiredCount = "0",
    
    # Logging Configuration
    [string]$LogGroupName = "/ecs/fluidity/server",
    [string]$LogRetentionDays = "30",
    
    # Lambda Configuration
    [string]$IdleThresholdMinutes = "15",
    [string]$LookbackPeriodMinutes = "10",
    [string]$SleepCheckIntervalMinutes = "5",
    [string]$DailyKillTime = "cron(0 23 * * ? *)",
    [string]$WakeLambdaTimeout = "30",
    [string]$SleepLambdaTimeout = "60",
    [string]$KillLambdaTimeout = "30",
    
    # API Gateway Configuration
    [string]$APIThrottleBurstLimit = "20",
    [string]$APIThrottleRateLimit = "3",
    [string]$APIQuotaLimit = "300"
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$CloudFormationDir = Join-Path (Split-Path -Parent $ScriptDir) "deployments\cloudformation"
$CertsDir = Join-Path (Split-Path -Parent $ScriptDir) "certs"

# Load certificates (plain text, not base64)
Write-Host "Loading TLS certificates..." -ForegroundColor Yellow
$ServerCertPath = Join-Path $CertsDir "server.crt"
$ServerKeyPath = Join-Path $CertsDir "server.key"
$CaCertPath = Join-Path $CertsDir "ca.crt"

if (-not (Test-Path $ServerCertPath)) {
    Write-Host "[ERROR] Server certificate not found: $ServerCertPath" -ForegroundColor Red
    Write-Host "Run the certificate generation script first:" -ForegroundColor Yellow
    Write-Host "  .\scripts\manage-certs.ps1" -ForegroundColor White
    exit 1
}

if (-not (Test-Path $ServerKeyPath)) {
    Write-Host "[ERROR] Server key not found: $ServerKeyPath" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $CaCertPath)) {
    Write-Host "[ERROR] CA certificate not found: $CaCertPath" -ForegroundColor Red
    exit 1
}

# Read certificate files as plain text (CloudFormation will handle storage in Secrets Manager)
$CertPem = (Get-Content $ServerCertPath -Raw).Trim()
$KeyPem = (Get-Content $ServerKeyPath -Raw).Trim()
$CaPem = (Get-Content $CaCertPath -Raw).Trim()

Write-Host "[OK] Certificates loaded" -ForegroundColor Green

$FargateTemplate = Join-Path $CloudFormationDir 'fargate.yaml'
$LambdaTemplate = Join-Path $CloudFormationDir 'lambda.yaml'
$StackPolicy = Join-Path $CloudFormationDir 'stack-policy.json'
$FargateStackName = "$StackName-fargate"
$LambdaStackName = "$StackName-lambda"

# Check if AWS CLI is installed and configured
Write-Host "Checking AWS CLI configuration..." -ForegroundColor Yellow

$AwsConfigured = $false
try {
    $CallerIdentity = aws sts get-caller-identity 2>&1
    if ($LASTEXITCODE -eq 0) {
        $AwsConfigured = $true
        $Identity = $CallerIdentity | ConvertFrom-Json
        Write-Host "[OK] AWS CLI is configured" -ForegroundColor Green
        Write-Host "  Account: $($Identity.Account)" -ForegroundColor Gray
        Write-Host "  User: $($Identity.Arn)" -ForegroundColor Gray
        
        # Auto-populate AccountId if not provided
        if (-not $AccountId) {
            $AccountId = $Identity.Account
            Write-Host "[OK] Using Account ID from AWS credentials: $AccountId" -ForegroundColor Green
        }
    }
}
catch {
    $AwsConfigured = $false
}

if (-not $AwsConfigured) {
    Write-Host "`n[ERROR] AWS CLI is not configured" -ForegroundColor Red
    Write-Host "`nThe AWS CLI needs to be configured with your credentials before running this script." -ForegroundColor Yellow
    
    Write-Host "`nTo configure AWS CLI:" -ForegroundColor Cyan
    Write-Host "  1. Run: aws configure" -ForegroundColor White
    Write-Host "  2. Enter your AWS Access Key ID" -ForegroundColor White
    Write-Host "  3. Enter your AWS Secret Access Key" -ForegroundColor White
    Write-Host "  4. Enter your default region (e.g., us-east-1)" -ForegroundColor White
    Write-Host "  5. Enter default output format: json" -ForegroundColor White
    
    Write-Host "`nTo get AWS credentials:" -ForegroundColor Cyan
    Write-Host "  1. Log in to AWS Console: https://console.aws.amazon.com" -ForegroundColor White
    Write-Host "  2. Go to: IAM > Users > [Your User] > Security Credentials" -ForegroundColor White
    Write-Host "  3. Create Access Key > CLI" -ForegroundColor White
    Write-Host "  4. Copy the Access Key ID and Secret Access Key" -ForegroundColor White
    
    Write-Host "`nAfter configuring AWS CLI, run this script again." -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

# Auto-gather missing required parameters from AWS
Write-Host "Checking required parameters..." -ForegroundColor Yellow

$UseCommandLineParams = $true
$GatheredParams = @()

# Try to auto-detect Region from AWS CLI default configuration if not provided
if (-not $Region) {
    Write-Host "Region not provided, attempting to auto-detect..." -ForegroundColor Gray
    try {
        $Region = aws configure get region 2>&1
        if ($LASTEXITCODE -eq 0 -and $Region) {
            $GatheredParams += "Region"
            Write-Host "[OK] Auto-detected Region: $Region" -ForegroundColor Green
        }
        else {
            $Region = $null
        }
    }
    catch {
        $Region = $null
    }
}

# Try to auto-detect VpcId from default VPC if not provided
if (-not $VpcId -and $Region) {
    Write-Host "VpcId not provided, attempting to auto-detect default VPC..." -ForegroundColor Gray
    try {
        $VpcId = aws ec2 describe-vpcs --region $Region --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text 2>&1
        if ($LASTEXITCODE -eq 0 -and $VpcId -and $VpcId -ne "None") {
            $GatheredParams += "VpcId"
            Write-Host "[OK] Auto-detected VpcId: $VpcId" -ForegroundColor Green
        }
        else {
            $VpcId = $null
        }
    }
    catch {
        $VpcId = $null
    }
}

# Try to auto-detect PublicSubnets from VPC if not provided
if (-not $PublicSubnets -and $VpcId -and $Region) {
    Write-Host "PublicSubnets not provided, attempting to auto-detect from VPC..." -ForegroundColor Gray
    try {
        $SubnetList = aws ec2 describe-subnets --region $Region --filters Name=vpc-id,Values=$VpcId Name=map-public-ip-on-launch,Values=true --query 'Subnets[*].SubnetId' --output text 2>&1
        if ($LASTEXITCODE -eq 0 -and $SubnetList) {
            $PublicSubnets = $SubnetList -replace "`t", ","
            if ($PublicSubnets) {
                $GatheredParams += "PublicSubnets"
                Write-Host "[OK] Auto-detected PublicSubnets: $PublicSubnets" -ForegroundColor Green
            }
            else {
                $PublicSubnets = $null
            }
        }
        else {
            $PublicSubnets = $null
        }
    }
    catch {
        $PublicSubnets = $null
    }
}

# Try to auto-detect public IP for AllowedIngressCidr if not provided
if (-not $AllowedIngressCidr) {
    Write-Host "AllowedIngressCidr not provided, attempting to auto-detect public IP..." -ForegroundColor Gray
    try {
        $PublicIp = $null
        # Try multiple IP services for reliability
        $IpServices = @(
            "https://api.ipify.org",
            "https://ifconfig.me/ip",
            "https://icanhazip.com"
        )
        
        foreach ($service in $IpServices) {
            try {
                $response = Invoke-WebRequest -Uri $service -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
                $PublicIp = $response.Content.Trim()
                if ($PublicIp -match '^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$') {
                    break
                }
            }
            catch {
                continue
            }
        }
        
        if ($PublicIp -and $PublicIp -match '^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$') {
            $AllowedIngressCidr = "$PublicIp/32"
            $GatheredParams += "AllowedIngressCidr"
            Write-Host "[OK] Auto-detected AllowedIngressCidr: $AllowedIngressCidr" -ForegroundColor Green
        }
        else {
            $AllowedIngressCidr = $null
        }
    }
    catch {
        $AllowedIngressCidr = $null
    }
}

if ($GatheredParams.Count -gt 0) {
    Write-Host "`nAuto-gathered parameters: $($GatheredParams -join ', ')" -ForegroundColor Cyan
}

# Validate all required parameters are now available
$MissingParams = @()
if (-not $Region) { $MissingParams += "Region" }
if (-not $VpcId) { $MissingParams += "VpcId" }
if (-not $PublicSubnets) { $MissingParams += "PublicSubnets" }
if (-not $AllowedIngressCidr) { $MissingParams += "AllowedIngressCidr" }

if ($MissingParams.Count -gt 0) {
    Write-Host "`n[ERROR] Missing required parameters:" -ForegroundColor Red
    foreach ($param in $MissingParams) {
        Write-Host "  - $param" -ForegroundColor Yellow
    }
    
    Write-Host "`nUnable to auto-detect all required parameters." -ForegroundColor Yellow
    Write-Host "Please provide these parameters as command-line arguments:" -ForegroundColor Cyan
    Write-Host "  Region            : Your AWS region (e.g., us-east-1, eu-west-1)" -ForegroundColor White
    Write-Host "  VpcId             : aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text" -ForegroundColor White
    Write-Host "  PublicSubnets     : aws ec2 describe-subnets --filters Name=vpc-id,Values=<VPC_ID> --query 'Subnets[*].SubnetId' --output text" -ForegroundColor White
    Write-Host "  AllowedIngressCidr: Your public IP with /32 CIDR (e.g., 1.2.3.4/32)" -ForegroundColor White
    
    Write-Host "`nExample usage:" -ForegroundColor Cyan
    Write-Host "  .\deploy-fluidity.ps1 -Action deploy -Region us-east-1 -VpcId vpc-abc123 -PublicSubnets subnet-1,subnet-2 -AllowedIngressCidr 1.2.3.4/32" -ForegroundColor White
    Write-Host "`nNote: AccountId is automatically detected from your AWS credentials" -ForegroundColor Gray
    
    Write-Host "`nFor detailed help: Get-Help .\deploy-fluidity.ps1 -Detailed" -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

# Set ContainerImage default if not provided
if (-not $ContainerImage) {
    $ContainerImage = "$AccountId.dkr.ecr.$Region.amazonaws.com/fluidity-server:latest"
}

Write-Host "[OK] All required parameters validated successfully" -ForegroundColor Green

# Skip the parameters file logic since we're using command-line/auto-detected params
if ($false) {
    # Use parameters file
    if (-not $ParametersFile) {
        $ParametersFile = Join-Path $CloudFormationDir "params.json"
    }
    
    if (-not (Test-Path $ParametersFile)) {
        Write-Error "Parameters file not found: $ParametersFile`n`nEither provide command-line parameters or create the parameters file."
    }
    
    # Validate parameters file has been configured
    Write-Host "Validating parameters file..." -ForegroundColor Yellow
    $ParamsContent = Get-Content $ParametersFile -Raw
    $PlaceholderPattern = '<[A-Z_]+>'
    $Placeholders = [regex]::Matches($ParamsContent, $PlaceholderPattern) | ForEach-Object { $_.Value } | Select-Object -Unique
    
    if ($Placeholders.Count -gt 0) {
        Write-Host "`n[ERROR] Parameters file contains unconfigured placeholders" -ForegroundColor Red
        Write-Host "`nFound placeholders in $ParametersFile :" -ForegroundColor Red
        
        foreach ($placeholder in $Placeholders) {
            Write-Host "  - $placeholder" -ForegroundColor Yellow
        }
        
        Write-Host "`nThe params.json file is for reference only. You must provide parameters via command-line arguments:" -ForegroundColor Cyan
        Write-Host "  .\deploy-fluidity.ps1 -Action deploy -AccountId <id> -Region <region> -VpcId <vpc> -PublicSubnets <subnet1,subnet2> -AllowedIngressCidr <ip>/32" -ForegroundColor White
        
        Write-Host "`nHow to get required parameter values:" -ForegroundColor Cyan
        Write-Host "  AccountId         : aws sts get-caller-identity --query Account --output text" -ForegroundColor White
        Write-Host "  Region            : Your AWS region (e.g., us-east-1, eu-west-1)" -ForegroundColor White
        Write-Host "  VpcId             : aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text" -ForegroundColor White
        Write-Host "  PublicSubnets     : aws ec2 describe-subnets --filters Name=vpc-id,Values=<VPC_ID> --query 'Subnets[*].SubnetId' --output text" -ForegroundColor White
        Write-Host "  AllowedIngressCidr: (Invoke-WebRequest -Uri 'https://ifconfig.me' -UseBasicParsing).Content.Trim() + '/32'" -ForegroundColor White
        
        Write-Host "`nFor detailed help: Get-Help .\deploy-fluidity.ps1 -Detailed" -ForegroundColor Cyan
        Write-Host ""
        exit 1
    }
    
    Write-Host "[OK] Parameters file validated successfully" -ForegroundColor Green
}

if (-not (Test-Path $FargateTemplate)) {
    Write-Error "Fargate template not found: $FargateTemplate"
}

if (-not (Test-Path $LambdaTemplate)) {
    Write-Error "Lambda template not found: $LambdaTemplate"
}

function Get-StackStatus {
    param([string]$Stack)
    
    $ErrorActionPreference = 'Continue'
    $Status = aws cloudformation describe-stacks --stack-name $Stack --region $Region --query 'Stacks[0].StackStatus' --output text 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        $ErrorMessage = $Status | Out-String
        if ($ErrorMessage -match 'does not exist') {
            return 'DOES_NOT_EXIST'
        }
        Write-Host "[WARNING] Unable to get stack status (may not exist yet)" -ForegroundColor Yellow
        return 'DOES_NOT_EXIST'
    }
    
    return $Status
}

function Deploy-Stack {
    param(
        [string]$Stack,
        [string]$Template,
        [string]$StackType  # 'fargate' or 'lambda'
    )
    
    Write-Host "`n=== Deploying $Stack ===" -ForegroundColor Green
    
    try {
        $Status = Get-StackStatus $Stack
    }
    catch {
        Write-Host "[ERROR] Failed to get stack status: $_" -ForegroundColor Red
        Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
        throw
    }
    
    # Build parameters based on stack type
    if ($StackType -eq 'fargate') {
        # Create parameters JSON file for certificates (too large for command-line)
        # Note: PublicSubnets should NOT be escaped in JSON format
        $ParamsJson = @(
            @{ParameterKey='ClusterName'; ParameterValue=$ClusterName},
            @{ParameterKey='ServiceName'; ParameterValue=$ServiceName},
            @{ParameterKey='ContainerImage'; ParameterValue=$ContainerImage},
            @{ParameterKey='ContainerPort'; ParameterValue=$ContainerPort},
            @{ParameterKey='Cpu'; ParameterValue=$Cpu},
            @{ParameterKey='Memory'; ParameterValue=$Memory},
            @{ParameterKey='DesiredCount'; ParameterValue=$DesiredCount},
            @{ParameterKey='VpcId'; ParameterValue=$VpcId},
            @{ParameterKey='PublicSubnets'; ParameterValue=$PublicSubnets},
            @{ParameterKey='AllowedIngressCidr'; ParameterValue=$AllowedIngressCidr},
            @{ParameterKey='AssignPublicIp'; ParameterValue='ENABLED'},
            @{ParameterKey='LogGroupName'; ParameterValue=$LogGroupName},
            @{ParameterKey='LogRetentionDays'; ParameterValue=$LogRetentionDays},
            @{ParameterKey='CertPem'; ParameterValue=$CertPem},
            @{ParameterKey='KeyPem'; ParameterValue=$KeyPem},
            @{ParameterKey='CaPem'; ParameterValue=$CaPem}
        )
        
        $TempParamsFile = Join-Path $env:TEMP "fluidity-fargate-params-$((Get-Date).Ticks).json"
        $ParamsJson | ConvertTo-Json -Depth 10 | Out-File -FilePath $TempParamsFile -Encoding ASCII -NoNewline
        $ParametersArg = "file://$TempParamsFile"
    }
    elseif ($StackType -eq 'lambda') {
        # Lambda stack parameters (command-line is fine, no large values)
        $ParamsArg = @(
            "ParameterKey=ECSClusterName,ParameterValue=$ClusterName",
            "ParameterKey=ECSServiceName,ParameterValue=$ServiceName",
            "ParameterKey=IdleThresholdMinutes,ParameterValue=$IdleThresholdMinutes",
            "ParameterKey=LookbackPeriodMinutes,ParameterValue=$LookbackPeriodMinutes",
            "ParameterKey=SleepCheckIntervalMinutes,ParameterValue=$SleepCheckIntervalMinutes",
            "ParameterKey=DailyKillTime,ParameterValue=$DailyKillTime",
            "ParameterKey=WakeLambdaTimeout,ParameterValue=$WakeLambdaTimeout",
            "ParameterKey=SleepLambdaTimeout,ParameterValue=$SleepLambdaTimeout",
            "ParameterKey=KillLambdaTimeout,ParameterValue=$KillLambdaTimeout",
            "ParameterKey=APIThrottleBurstLimit,ParameterValue=$APIThrottleBurstLimit",
            "ParameterKey=APIThrottleRateLimit,ParameterValue=$APIThrottleRateLimit",
            "ParameterKey=APIQuotaLimit,ParameterValue=$APIQuotaLimit"
        )
        $ParametersArg = $null  # Will use command-line for lambda
    }
    
    if ($Status -eq 'DOES_NOT_EXIST') {
        Write-Host "Creating new stack: $Stack" -ForegroundColor Yellow
        
        if ($StackType -eq 'fargate') {
            # Use parameters file for Fargate (contains large certificate values)
            aws cloudformation create-stack `
                --stack-name $Stack `
                --template-body file://$Template `
                --parameters $ParametersArg `
                --capabilities $Capabilities `
                --region $Region `
                --tags Key=Application,Value=Fluidity Key=ManagedBy,Value=CloudFormation
        }
        else {
            # Use command-line parameters for Lambda (smaller values)
            $AwsParams = @(
                'cloudformation', 'create-stack',
                '--stack-name', $Stack,
                '--template-body', "file://$Template",
                '--parameters'
            )
            $AwsParams += $ParamsArg
            $AwsParams += @(
                '--capabilities', $Capabilities,
                '--region', $Region,
                '--tags', 'Key=Application,Value=Fluidity', 'Key=ManagedBy,Value=CloudFormation'
            )
            
            & aws $AwsParams
        }
        
        Write-Host "Waiting for stack creation..." -ForegroundColor Yellow
        aws cloudformation wait stack-create-complete --stack-name $Stack --region $Region
        
        # Clean up temp file if Fargate
        if ($StackType -eq 'fargate' -and (Test-Path $TempParamsFile)) {
            Remove-Item $TempParamsFile -Force
        }
        
        Write-Host "[OK] Stack created successfully" -ForegroundColor Green
    }
    else {
        Write-Host "Updating existing stack: $Stack (Current status: $Status)" -ForegroundColor Yellow
        
        try {
            if ($StackType -eq 'fargate') {
                # Use parameters file for Fargate
                aws cloudformation update-stack `
                    --stack-name $Stack `
                    --template-body file://$Template `
                    --parameters $ParametersArg `
                    --capabilities $Capabilities `
                    --region $Region `
                    --tags Key=Application,Value=Fluidity Key=ManagedBy,Value=CloudFormation | Out-Null
            }
            else {
                # Use command-line parameters for Lambda
                $AwsParams = @(
                    'cloudformation', 'update-stack',
                    '--stack-name', $Stack,
                    '--template-body', "file://$Template",
                    '--parameters'
                )
                $AwsParams += $ParamsArg
                $AwsParams += @(
                    '--capabilities', $Capabilities,
                    '--region', $Region,
                    '--tags', 'Key=Application,Value=Fluidity', 'Key=ManagedBy,Value=CloudFormation'
                )
                
                & aws $AwsParams | Out-Null
            }
            
            Write-Host "Waiting for stack update..." -ForegroundColor Yellow
            aws cloudformation wait stack-update-complete --stack-name $Stack --region $Region
            
            # Clean up temp file if Fargate
            if ($StackType -eq 'fargate' -and (Test-Path $TempParamsFile)) {
                Remove-Item $TempParamsFile -Force
            }
            
            Write-Host "[OK] Stack updated successfully" -ForegroundColor Green
        }
        catch {
            # Clean up temp file on error
            if ($StackType -eq 'fargate' -and (Test-Path $TempParamsFile)) {
                Remove-Item $TempParamsFile -Force
            }
            
            if ($_ -match 'No updates are to be performed') {
                Write-Host "[OK] Stack is already up to date (no changes)" -ForegroundColor Green
            }
            else {
                throw
            }
        }
    }
}

function Get-StackOutputs {
    param([string]$Stack)
    
    $Outputs = aws cloudformation describe-stacks --stack-name $Stack --query 'Stacks[0].Outputs' --output json
    if ($null -eq $Outputs) {
        Write-Host "No outputs found for stack: $Stack" -ForegroundColor Yellow
    }
    else {
        Write-Host "`n=== Stack Outputs ===" -ForegroundColor Green
        $Outputs | ConvertFrom-Json | ForEach-Object {
            Write-Host "$($_.OutputKey): $($_.OutputValue)"
        }
    }
}

function Get-StackDriftStatus {
    param([string]$Stack)
    
    Write-Host "`n=== Checking Stack Drift ===" -ForegroundColor Green
    
    $DriftCheck = aws cloudformation detect-stack-drift `
        --stack-name $Stack `
        --query 'StackDriftDetectionId' `
        --output text
    
    Write-Host "Drift detection started: $DriftCheck" -ForegroundColor Yellow
    Write-Host "Checking status..." -ForegroundColor Yellow
    
    Start-Sleep -Seconds 3
    
    $DriftStatus = aws cloudformation describe-stack-drift-detection-status `
        --stack-drift-detection-id $DriftCheck `
        --query 'StackDriftDetectionStatus' `
        --output text
    
    $DriftSummary = aws cloudformation describe-stack-drift-detection-status `
        --stack-drift-detection-id $DriftCheck `
        --output json | ConvertFrom-Json
    
    if ($DriftStatus -eq 'DETECTION_COMPLETE') {
        $Drift = $DriftSummary.StackDriftStatus
        if ($Drift -eq 'DRIFTED') {
            Write-Host "[WARNING] DRIFTED: Stack has manual changes" -ForegroundColor Red
        }
        elseif ($Drift -eq 'IN_SYNC') {
            Write-Host "[OK] IN_SYNC: Stack matches template" -ForegroundColor Green
        }
        else {
            Write-Host "Status: $Drift" -ForegroundColor Yellow
        }
    }
    else {
        Write-Host "Detection status: $DriftStatus" -ForegroundColor Yellow
    }
}

# Main execution
try {
    switch ($Action) {
        'deploy' {
            Deploy-Stack $FargateStackName $FargateTemplate 'fargate'
            Deploy-Stack $LambdaStackName $LambdaTemplate 'lambda'
            Write-Host "`n=== Applying Stack Policy ===" -ForegroundColor Green
            aws cloudformation set-stack-policy `
                --stack-name $FargateStackName `
                --stack-policy-body file://$StackPolicy | Out-Null
            Write-Host "[OK] Stack policy applied" -ForegroundColor Green
            Get-StackOutputs $FargateStackName
            Get-StackOutputs $LambdaStackName
        }
        'delete' {
            if (-not $Force) {
                $Confirm = Read-Host "Are you sure you want to DELETE $StackName? (yes/no)"
                if ($Confirm -ne 'yes') {
                    Write-Host "Delete cancelled" -ForegroundColor Yellow
                    exit 0
                }
            }
            
            Write-Host "`n=== Deleting $StackName ===" -ForegroundColor Red
            
            # Remove stack policies first (they prevent deletion)
            try {
                aws cloudformation delete-stack-policy `
                    --stack-name $FargateStackName 2>&1 | Out-Null
            }
            catch { }
            
            aws cloudformation delete-stack --stack-name $FargateStackName
            aws cloudformation delete-stack --stack-name $LambdaStackName
            
            Write-Host "Waiting for stacks to be deleted..." -ForegroundColor Yellow
            aws cloudformation wait stack-delete-complete --stack-name $FargateStackName
            aws cloudformation wait stack-delete-complete --stack-name $LambdaStackName
            Write-Host "[OK] Stacks deleted" -ForegroundColor Green
        }
        'status' {
            Write-Host "`n=== Stack Status ===" -ForegroundColor Green
            
            $FargateStatus = Get-StackStatus $FargateStackName
            $LambdaStatus = Get-StackStatus $LambdaStackName
            
            Write-Host "Fargate Stack: $FargateStatus"
            Write-Host "Lambda Stack: $LambdaStatus"
            
            if ($FargateStatus -notmatch 'DELETE' -and $FargateStatus -ne 'DOES_NOT_EXIST') {
                Get-StackDriftStatus $FargateStackName
            }
        }
        'outputs' {
            Get-StackOutputs $FargateStackName
            Get-StackOutputs $LambdaStackName
        }
    }
    
    Write-Host "`n[OK] Operation completed successfully" -ForegroundColor Green
}
catch {
    Write-Host "`n[ERROR] Error: $_" -ForegroundColor Red
    exit 1
}
