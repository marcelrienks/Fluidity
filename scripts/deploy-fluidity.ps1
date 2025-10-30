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

.PARAMETER Environment
The environment to deploy to: dev or prod
Default: dev

.PARAMETER Action
The action to perform:
- deploy: Create or update the stack
- delete: Delete the stack
- status: Show stack status
- outputs: Show stack outputs
Default: deploy

.PARAMETER StackName
The CloudFormation stack name. Default: fluidity-{environment}

.PARAMETER ParametersFile
Path to parameters JSON file. Default: params-{environment}.json in current directory

.PARAMETER Capabilities
CloudFormation capabilities. Default: CAPABILITY_NAMED_IAM

.EXAMPLE
PS> ./deploy-fluidity.ps1 -Environment prod -Action deploy
Deploy production stack with parameters from params-prod.json

.EXAMPLE
PS> ./deploy-fluidity.ps1 -Environment dev -Action status
Check status of dev stack

.EXAMPLE
PS> ./deploy-fluidity.ps1 -Environment dev -Action delete
Delete dev stack (requires confirmation)
#>

param(
    [ValidateSet('dev', 'prod')]
    [string]$Environment = 'dev',
    
    [ValidateSet('deploy', 'delete', 'status', 'outputs')]
    [string]$Action = 'deploy',
    
    [string]$StackName = "fluidity-$Environment",
    
    [string]$ParametersFile = "params-$Environment.json",
    
    [string]$Capabilities = 'CAPABILITY_NAMED_IAM',
    
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$FargateTemplate = Join-Path $ScriptDir 'fargate.yaml'
$LambdaTemplate = Join-Path $ScriptDir 'lambda.yaml'
$StackPolicy = Join-Path $ScriptDir 'stack-policy.json'

# Validate prerequisites
if (-not (Test-Path $ParametersFile)) {
    Write-Error "Parameters file not found: $ParametersFile`n`nEdit the file and update YOUR_* placeholders with actual values."
}

if (-not (Test-Path $FargateTemplate)) {
    Write-Error "Fargate template not found: $FargateTemplate"
}

if (-not (Test-Path $LambdaTemplate)) {
    Write-Error "Lambda template not found: $LambdaTemplate"
}

function Get-StackStatus {
    param([string]$Stack)
    try {
        $Status = aws cloudformation describe-stacks --stack-name $Stack --query 'Stacks[0].StackStatus' --output text 2>&1
        return $Status
    }
    catch {
        if ($_ -match 'does not exist') {
            return 'DOES_NOT_EXIST'
        }
        throw
    }
}

function Deploy-Stack {
    param([string]$Stack, [string]$Template)
    
    Write-Host "`n=== Deploying $Stack ===" -ForegroundColor Green
    
    $Status = Get-StackStatus $Stack
    
    if ($Status -eq 'DOES_NOT_EXIST') {
        Write-Host "Creating new stack: $Stack" -ForegroundColor Yellow
        
        aws cloudformation create-stack `
            --stack-name $Stack `
            --template-body file://$Template `
            --parameters file://$ParametersFile `
            --capabilities $Capabilities `
            --tags Key=Environment,Value=$Environment Key=Application,Value=Fluidity Key=ManagedBy,Value=CloudFormation
        
        Write-Host "Waiting for stack creation..." -ForegroundColor Yellow
        aws cloudformation wait stack-create-complete --stack-name $Stack
        Write-Host "✓ Stack created successfully" -ForegroundColor Green
    }
    else {
        Write-Host "Updating existing stack: $Stack (Current status: $Status)" -ForegroundColor Yellow
        
        try {
            aws cloudformation update-stack `
                --stack-name $Stack `
                --template-body file://$Template `
                --parameters file://$ParametersFile `
                --capabilities $Capabilities `
                --tags Key=Environment,Value=$Environment Key=Application,Value=Fluidity Key=ManagedBy,Value=CloudFormation | Out-Null
            
            Write-Host "Waiting for stack update..." -ForegroundColor Yellow
            aws cloudformation wait stack-update-complete --stack-name $Stack
            Write-Host "✓ Stack updated successfully" -ForegroundColor Green
        }
        catch {
            if ($_ -match 'No updates are to be performed') {
                Write-Host "✓ Stack is already up to date (no changes)" -ForegroundColor Green
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
            Write-Host "⚠ DRIFTED: Stack has manual changes" -ForegroundColor Red
        }
        elseif ($Drift -eq 'IN_SYNC') {
            Write-Host "✓ IN_SYNC: Stack matches template" -ForegroundColor Green
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
            Deploy-Stack "fluidity-$Environment-fargate" $FargateTemplate
            Deploy-Stack "fluidity-$Environment-lambda" $LambdaTemplate
            Write-Host "`n=== Applying Stack Policy ===" -ForegroundColor Green
            aws cloudformation set-stack-policy `
                --stack-name "fluidity-$Environment-fargate" `
                --stack-policy-body file://$StackPolicy | Out-Null
            Write-Host "✓ Stack policy applied" -ForegroundColor Green
            Get-StackOutputs "fluidity-$Environment-fargate"
            Get-StackOutputs "fluidity-$Environment-lambda"
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
                    --stack-name "fluidity-$Environment-fargate" 2>&1 | Out-Null
            }
            catch { }
            
            aws cloudformation delete-stack --stack-name "fluidity-$Environment-fargate"
            aws cloudformation delete-stack --stack-name "fluidity-$Environment-lambda"
            
            Write-Host "Waiting for stacks to be deleted..." -ForegroundColor Yellow
            aws cloudformation wait stack-delete-complete --stack-name "fluidity-$Environment-fargate"
            aws cloudformation wait stack-delete-complete --stack-name "fluidity-$Environment-lambda"
            Write-Host "✓ Stacks deleted" -ForegroundColor Green
        }
        'status' {
            Write-Host "`n=== Stack Status ===" -ForegroundColor Green
            
            $FargateStatus = Get-StackStatus "fluidity-$Environment-fargate"
            $LambdaStatus = Get-StackStatus "fluidity-$Environment-lambda"
            
            Write-Host "Fargate Stack: $FargateStatus"
            Write-Host "Lambda Stack: $LambdaStatus"
            
            if ($FargateStatus -notmatch 'DELETE' -and $FargateStatus -ne 'DOES_NOT_EXIST') {
                Get-StackDriftStatus "fluidity-$Environment-fargate"
            }
        }
        'outputs' {
            Get-StackOutputs "fluidity-$Environment-fargate"
            Get-StackOutputs "fluidity-$Environment-lambda"
        }
    }
    
    Write-Host "`n✓ Operation completed successfully" -ForegroundColor Green
}
catch {
    Write-Host "`n✗ Error: $_" -ForegroundColor Red
    exit 1
}
