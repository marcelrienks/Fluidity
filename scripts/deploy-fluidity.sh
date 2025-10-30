#!/usr/bin/env bash
set -euo pipefail

#
# deploy-fluidity.sh - Deploy Fluidity infrastructure to AWS using CloudFormation
#
# Usage:
#   ./deploy-fluidity.sh [OPTIONS]
#
# Options:
#   -e, --environment ENV      Environment to deploy to: dev or prod (default: dev)
#   -a, --action ACTION        Action to perform: deploy, delete, status, outputs (default: deploy)
#   -s, --stack-name NAME      CloudFormation stack name (default: fluidity-{environment})
#   -p, --parameters FILE      Parameters JSON file (default: params-{environment}.json)
#   -f, --force                Skip confirmation prompts
#   -h, --help                 Show this help message
#
# Examples:
#   ./deploy-fluidity.sh -e prod -a deploy
#   ./deploy-fluidity.sh -e dev -a status
#   ./deploy-fluidity.sh -e dev -a delete -f
#

# Default values
ENVIRONMENT="${ENVIRONMENT:-dev}"
ACTION="${ACTION:-deploy}"
FORCE=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLOUDFORMATION_DIR="$(dirname "$SCRIPT_DIR")/cloudformation"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -a|--action)
            ACTION="$2"
            shift 2
            ;;
        -s|--stack-name)
            STACK_NAME="$2"
            shift 2
            ;;
        -p|--parameters)
            PARAMETERS_FILE="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -h|--help)
            grep '^#' "$0" | tail -n +3
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|prod)$ ]]; then
    echo "Error: Environment must be 'dev' or 'prod'"
    exit 1
fi

# Validate action
if [[ ! "$ACTION" =~ ^(deploy|delete|status|outputs)$ ]]; then
    echo "Error: Action must be 'deploy', 'delete', 'status', or 'outputs'"
    exit 1
fi

# Set defaults based on environment
STACK_NAME="${STACK_NAME:-fluidity-$ENVIRONMENT}"
FARGATE_STACK_NAME="${STACK_NAME}-fargate"
LAMBDA_STACK_NAME="${STACK_NAME}-lambda"
PARAMETERS_FILE="${PARAMETERS_FILE:-$CLOUDFORMATION_DIR/params-$ENVIRONMENT.json}"
FARGATE_TEMPLATE="$CLOUDFORMATION_DIR/fargate.yaml"
LAMBDA_TEMPLATE="$CLOUDFORMATION_DIR/lambda.yaml"
STACK_POLICY="$CLOUDFORMATION_DIR/stack-policy.json"
CAPABILITIES="CAPABILITY_NAMED_IAM"

# Validate prerequisites
if [[ ! -f "$PARAMETERS_FILE" ]]; then
    echo "Error: Parameters file not found: $PARAMETERS_FILE"
    echo ""
    echo "Edit the file and update YOUR_* placeholders with actual values."
    exit 1
fi

if [[ ! -f "$FARGATE_TEMPLATE" ]]; then
    echo "Error: Fargate template not found: $FARGATE_TEMPLATE"
    exit 1
fi

if [[ ! -f "$LAMBDA_TEMPLATE" ]]; then
    echo "Error: Lambda template not found: $LAMBDA_TEMPLATE"
    exit 1
fi

# Helper functions
get_stack_status() {
    local stack_name="$1"
    aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --query 'Stacks[0].StackStatus' \
        --output text 2>/dev/null || echo "DOES_NOT_EXIST"
}

deploy_stack() {
    local stack_name="$1"
    local template="$2"
    
    echo ""
    echo "=== Deploying $stack_name ==="
    
    local status
    status=$(get_stack_status "$stack_name")
    
    if [[ "$status" == "DOES_NOT_EXIST" ]]; then
        echo "Creating new stack: $stack_name"
        
        aws cloudformation create-stack \
            --stack-name "$stack_name" \
            --template-body "file://$template" \
            --parameters "file://$PARAMETERS_FILE" \
            --capabilities "$CAPABILITIES" \
            --tags \
                "Key=Environment,Value=$ENVIRONMENT" \
                "Key=Application,Value=Fluidity" \
                "Key=ManagedBy,Value=CloudFormation" \
            --output text
        
        echo "Waiting for stack creation..."
        aws cloudformation wait stack-create-complete --stack-name "$stack_name"
        echo "✓ Stack created successfully"
    else
        echo "Updating existing stack: $stack_name (Current status: $status)"
        
        if aws cloudformation update-stack \
            --stack-name "$stack_name" \
            --template-body "file://$template" \
            --parameters "file://$PARAMETERS_FILE" \
            --capabilities "$CAPABILITIES" \
            --tags \
                "Key=Environment,Value=$ENVIRONMENT" \
                "Key=Application,Value=Fluidity" \
                "Key=ManagedBy,Value=CloudFormation" \
            --output text 2>&1 | grep -q "No updates are to be performed"; then
            echo "✓ Stack is already up to date (no changes)"
        else
            echo "Waiting for stack update..."
            aws cloudformation wait stack-update-complete --stack-name "$stack_name"
            echo "✓ Stack updated successfully"
        fi
    fi
}

get_stack_outputs() {
    local stack_name="$1"
    
    local outputs
    outputs=$(aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --query 'Stacks[0].Outputs' \
        --output json 2>/dev/null || echo "null")
    
    if [[ "$outputs" == "null" ]] || [[ -z "$outputs" ]]; then
        echo "No outputs found for stack: $stack_name"
    else
        echo ""
        echo "=== Stack Outputs ===" 
        echo "$outputs" | jq -r '.[] | "\(.OutputKey): \(.OutputValue)"'
    fi
}

get_stack_drift_status() {
    local stack_name="$1"
    
    echo ""
    echo "=== Checking Stack Drift ==="
    
    local drift_id
    drift_id=$(aws cloudformation detect-stack-drift \
        --stack-name "$stack_name" \
        --query 'StackDriftDetectionId' \
        --output text)
    
    echo "Drift detection started: $drift_id"
    echo "Checking status..."
    
    sleep 3
    
    local status
    status=$(aws cloudformation describe-stack-drift-detection-status \
        --stack-drift-detection-id "$drift_id" \
        --query 'StackDriftDetectionStatus' \
        --output text)
    
    if [[ "$status" == "DETECTION_COMPLETE" ]]; then
        local drift
        drift=$(aws cloudformation describe-stack-drift-detection-status \
            --stack-drift-detection-id "$drift_id" \
            --query 'StackDriftStatus' \
            --output text)
        
        if [[ "$drift" == "DRIFTED" ]]; then
            echo "⚠ DRIFTED: Stack has manual changes"
        elif [[ "$drift" == "IN_SYNC" ]]; then
            echo "✓ IN_SYNC: Stack matches template"
        else
            echo "Status: $drift"
        fi
    else
        echo "Detection status: $status"
    fi
}

# Main execution
case "$ACTION" in
    deploy)
        deploy_stack "$FARGATE_STACK_NAME" "$FARGATE_TEMPLATE"
        deploy_stack "$LAMBDA_STACK_NAME" "$LAMBDA_TEMPLATE"
        
        echo ""
        echo "=== Applying Stack Policy ==="
        aws cloudformation set-stack-policy \
            --stack-name "$FARGATE_STACK_NAME" \
            --stack-policy-body "file://$STACK_POLICY"
        echo "✓ Stack policy applied"
        
        get_stack_outputs "$FARGATE_STACK_NAME"
        get_stack_outputs "$LAMBDA_STACK_NAME"
        ;;
    delete)
        if [[ "$FORCE" != true ]]; then
            echo "Are you sure you want to DELETE $STACK_NAME? (type 'yes' to confirm)"
            read -r confirm
            if [[ "$confirm" != "yes" ]]; then
                echo "Delete cancelled"
                exit 0
            fi
        fi
        
        echo ""
        echo "=== Deleting $STACK_NAME ==="
        
        # Remove stack policies first (they prevent deletion)
        aws cloudformation delete-stack-policy \
            --stack-name "$FARGATE_STACK_NAME" 2>/dev/null || true
        
        aws cloudformation delete-stack --stack-name "$FARGATE_STACK_NAME"
        aws cloudformation delete-stack --stack-name "$LAMBDA_STACK_NAME"
        
        echo "Waiting for stacks to be deleted..."
        aws cloudformation wait stack-delete-complete --stack-name "$FARGATE_STACK_NAME"
        aws cloudformation wait stack-delete-complete --stack-name "$LAMBDA_STACK_NAME"
        echo "✓ Stacks deleted"
        ;;
    status)
        echo ""
        echo "=== Stack Status ==="
        
        local fargate_status
        fargate_status=$(get_stack_status "$FARGATE_STACK_NAME")
        local lambda_status
        lambda_status=$(get_stack_status "$LAMBDA_STACK_NAME")
        
        echo "Fargate Stack: $fargate_status"
        echo "Lambda Stack: $lambda_status"
        
        if [[ ! "$fargate_status" =~ DELETE ]] && [[ "$fargate_status" != "DOES_NOT_EXIST" ]]; then
            get_stack_drift_status "$FARGATE_STACK_NAME"
        fi
        ;;
    outputs)
        get_stack_outputs "$FARGATE_STACK_NAME"
        get_stack_outputs "$LAMBDA_STACK_NAME"
        ;;
esac

echo ""
echo "✓ Operation completed successfully"
