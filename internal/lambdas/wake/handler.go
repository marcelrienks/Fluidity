package wake

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// WakeRequest represents the input to the Wake Lambda
type WakeRequest struct {
	ClusterName string `json:"cluster_name,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

// WakeResponse represents the output from the Wake Lambda
type WakeResponse struct {
	Status             string `json:"status"`
	DesiredCount       int32  `json:"desiredCount"`
	RunningCount       int32  `json:"runningCount"`
	PendingCount       int32  `json:"pendingCount"`
	EstimatedStartTime string `json:"estimatedStartTime,omitempty"`
	Message            string `json:"message"`
}

// ECSClient interface for testing
type ECSClient interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

// Handler processes wake requests
type Handler struct {
	ecsClient   ECSClient
	clusterName string
	serviceName string
}

// NewHandler creates a new wake handler with AWS SDK clients
func NewHandler(ctx context.Context, clusterName, serviceName string) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	if clusterName == "" {
		return nil, fmt.Errorf("clusterName is required")
	}

	if serviceName == "" {
		return nil, fmt.Errorf("serviceName is required")
	}

	return &Handler{
		ecsClient:   ecs.NewFromConfig(cfg),
		clusterName: clusterName,
		serviceName: serviceName,
	}, nil
}

// NewHandlerWithClient creates a new wake handler with a provided ECS client (for testing)
func NewHandlerWithClient(ecsClient ECSClient, clusterName, serviceName string) *Handler {
	return &Handler{
		ecsClient:   ecsClient,
		clusterName: clusterName,
		serviceName: serviceName,
	}
}

// HandleRequest processes the wake request
func (h *Handler) HandleRequest(ctx context.Context, request WakeRequest) (*WakeResponse, error) {
	// Allow request to override cluster/service names (for testing)
	clusterName := h.clusterName
	if request.ClusterName != "" {
		clusterName = request.ClusterName
	}

	serviceName := h.serviceName
	if request.ServiceName != "" {
		serviceName = request.ServiceName
	}

	// Step 1: Describe the current service state
	describeInput := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	describeOutput, err := h.ecsClient.DescribeServices(ctx, describeInput)
	if err != nil {
		return nil, fmt.Errorf("failed to describe ECS service: %w", err)
	}

	if len(describeOutput.Services) == 0 {
		return nil, fmt.Errorf("service %s not found in cluster %s", serviceName, clusterName)
	}

	service := describeOutput.Services[0]
	desiredCount := service.DesiredCount
	runningCount := service.RunningCount
	pendingCount := service.PendingCount

	// Step 2: Check if service is already running or starting
	if desiredCount > 0 {
		status := "already_running"
		message := fmt.Sprintf("Service already has desiredCount=%d", desiredCount)

		if runningCount == 0 && pendingCount > 0 {
			status = "starting"
			message = fmt.Sprintf("Service is starting (desiredCount=%d, pendingCount=%d)", desiredCount, pendingCount)
		} else if runningCount > 0 {
			message = fmt.Sprintf("Service is running (desiredCount=%d, runningCount=%d)", desiredCount, runningCount)
		}

		return &WakeResponse{
			Status:       status,
			DesiredCount: desiredCount,
			RunningCount: runningCount,
			PendingCount: pendingCount,
			Message:      message,
		}, nil
	}

	// Step 3: Service is stopped (desiredCount=0), start it
	updateInput := &ecs.UpdateServiceInput{
		Cluster:      aws.String(clusterName),
		Service:      aws.String(serviceName),
		DesiredCount: aws.Int32(1),
	}

	_, err = h.ecsClient.UpdateService(ctx, updateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to update ECS service: %w", err)
	}

	// Estimate start time based on Fargate cold start (typically 60-90 seconds)
	estimatedStartTime := time.Now().Add(75 * time.Second).Format(time.RFC3339)

	return &WakeResponse{
		Status:             "waking",
		DesiredCount:       1,
		RunningCount:       0,
		PendingCount:       0,
		EstimatedStartTime: estimatedStartTime,
		Message:            "Service wake initiated. ECS task starting (estimated 60-90 seconds)",
	}, nil
}
