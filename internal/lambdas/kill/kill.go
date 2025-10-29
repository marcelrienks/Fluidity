package kill

import (
	"context"
	"fmt"

	"fluidity/internal/shared/logger"

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
	logger      *logger.Logger
}

// NewHandler creates a new kill handler with AWS SDK clients
func NewHandler(ctx context.Context, clusterName, serviceName string) (*Handler, error) {
	log := logger.NewFromEnv()

	log.Info("Initializing Kill Lambda handler", map[string]interface{}{
		"clusterName": clusterName,
		"serviceName": serviceName,
	})

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS SDK config", err)
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	if clusterName == "" {
		log.Error("Missing required parameter: clusterName", nil)
		return nil, fmt.Errorf("clusterName is required")
	}

	if serviceName == "" {
		log.Error("Missing required parameter: serviceName", nil)
		return nil, fmt.Errorf("serviceName is required")
	}

	log.Info("Kill Lambda handler initialized successfully")

	return &Handler{
		ecsClient:   ecs.NewFromConfig(cfg),
		clusterName: clusterName,
		serviceName: serviceName,
		logger:      log,
	}, nil
}

// NewHandlerWithClient creates a new kill handler with a provided ECS client (for testing)
func NewHandlerWithClient(ecsClient ECSClient, clusterName, serviceName string) *Handler {
	return &Handler{
		ecsClient:   ecsClient,
		clusterName: clusterName,
		serviceName: serviceName,
		logger:      logger.New("info"),
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

	h.logger.Info("Processing kill request", map[string]interface{}{
		"clusterName": clusterName,
		"serviceName": serviceName,
	})

	// Set desired count to 0 immediately (no checks, no validation)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:      aws.String(clusterName),
		Service:      aws.String(serviceName),
		DesiredCount: aws.Int32(0),
	}

	h.logger.Info("Initiating immediate service shutdown", map[string]interface{}{
		"clusterName": clusterName,
		"serviceName": serviceName,
	})

	_, err := h.ecsClient.UpdateService(ctx, updateInput)
	if err != nil {
		h.logger.Error("Failed to update ECS service", err, map[string]interface{}{
			"clusterName": clusterName,
			"serviceName": serviceName,
		})
		return nil, fmt.Errorf("failed to update ECS service: %w", err)
	}

	h.logger.Info("Service shutdown initiated successfully", map[string]interface{}{
		"desiredCount": 0,
	})

	return &KillResponse{
		Status:       "killed",
		DesiredCount: 0,
		Message:      "Service shutdown initiated. ECS tasks will terminate immediately.",
	}, nil
}
