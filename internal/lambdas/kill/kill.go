package kill

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// KillRequest represents the input to the Kill Lambda
type KillRequest struct {
	ClusterName string `json:"cluster_name,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

// KillResponse represents the output from the Kill Lambda
type KillResponse struct {
	Status       string `json:"status"`
	DesiredCount int32  `json:"desiredCount"`
	Message      string `json:"message"`
}

// ECSClient interface for testing
type ECSClient interface {
	UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

// Handler processes kill requests
type Handler struct {
	ecsClient   ECSClient
	clusterName string
	serviceName string
}

// NewHandler creates a new kill handler with AWS SDK clients
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

// NewHandlerWithClient creates a new kill handler with a provided ECS client (for testing)
func NewHandlerWithClient(ecsClient ECSClient, clusterName, serviceName string) *Handler {
	return &Handler{
		ecsClient:   ecsClient,
		clusterName: clusterName,
		serviceName: serviceName,
	}
}

// HandleRequest processes the kill request
func (h *Handler) HandleRequest(ctx context.Context, request KillRequest) (*KillResponse, error) {
	// Allow request to override cluster/service names (for testing)
	clusterName := h.clusterName
	if request.ClusterName != "" {
		clusterName = request.ClusterName
	}

	serviceName := h.serviceName
	if request.ServiceName != "" {
		serviceName = request.ServiceName
	}

	// Set desired count to 0 immediately (no checks, no validation)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:      aws.String(clusterName),
		Service:      aws.String(serviceName),
		DesiredCount: aws.Int32(0),
	}

	_, err := h.ecsClient.UpdateService(ctx, updateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to update ECS service: %w", err)
	}

	return &KillResponse{
		Status:       "killed",
		DesiredCount: 0,
		Message:      "Service shutdown initiated. ECS tasks will terminate immediately.",
	}, nil
}
