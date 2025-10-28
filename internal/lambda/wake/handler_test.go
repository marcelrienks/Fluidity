package wake

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// MockECSClient implements a mock ECS client for testing
type MockECSClient struct {
	DescribeServicesFunc func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	UpdateServiceFunc    func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

func (m *MockECSClient) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return m.DescribeServicesFunc(ctx, params, optFns...)
}

func (m *MockECSClient) UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	return m.UpdateServiceFunc(ctx, params, optFns...)
}

// TestWakeWhenServiceStopped verifies wake sets DesiredCount=1 when service is stopped
func TestWakeWhenServiceStopped(t *testing.T) {
	mockECS := &MockECSClient{
		DescribeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []types.Service{
					{
						ServiceName:  stringPtr("fluidity-server"),
						DesiredCount: int32(0),
						RunningCount: int32(0),
						PendingCount: int32(0),
					},
				},
			}, nil
		},
		UpdateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			if *params.DesiredCount != 1 {
				t.Errorf("Expected DesiredCount=1, got %d", *params.DesiredCount)
			}
			return &ecs.UpdateServiceOutput{}, nil
		},
	}

	handler := NewHandlerWithClient(mockECS, "test-cluster", "fluidity-server")

	response, err := handler.HandleRequest(context.Background(), WakeRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Status != "waking" {
		t.Errorf("Expected status 'waking', got '%s'", response.Status)
	}

	if response.DesiredCount != 1 {
		t.Errorf("Expected DesiredCount=1, got %d", response.DesiredCount)
	}

	if response.EstimatedStartTime == "" {
		t.Error("Expected EstimatedStartTime to be set")
	}
}

// TestWakeWhenServiceAlreadyRunning verifies idempotent behavior when service is running
func TestWakeWhenServiceAlreadyRunning(t *testing.T) {
	mockECS := &MockECSClient{
		DescribeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []types.Service{
					{
						ServiceName:  stringPtr("fluidity-server"),
						DesiredCount: int32(1),
						RunningCount: int32(1),
						PendingCount: int32(0),
					},
				},
			}, nil
		},
		UpdateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			t.Error("UpdateService should not be called when service is already running")
			return &ecs.UpdateServiceOutput{}, nil
		},
	}

	handler := NewHandlerWithClient(mockECS, "test-cluster", "fluidity-server")

	response, err := handler.HandleRequest(context.Background(), WakeRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Status != "already_running" {
		t.Errorf("Expected status 'already_running', got '%s'", response.Status)
	}

	if response.DesiredCount != 1 {
		t.Errorf("Expected DesiredCount=1, got %d", response.DesiredCount)
	}

	if response.RunningCount != 1 {
		t.Errorf("Expected RunningCount=1, got %d", response.RunningCount)
	}
}

// TestWakeWhenServiceStarting verifies status when service is pending
func TestWakeWhenServiceStarting(t *testing.T) {
	mockECS := &MockECSClient{
		DescribeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []types.Service{
					{
						ServiceName:  stringPtr("fluidity-server"),
						DesiredCount: int32(1),
						RunningCount: int32(0),
						PendingCount: int32(1),
					},
				},
			}, nil
		},
		UpdateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			t.Error("UpdateService should not be called when service is starting")
			return &ecs.UpdateServiceOutput{}, nil
		},
	}

	handler := NewHandlerWithClient(mockECS, "test-cluster", "fluidity-server")

	response, err := handler.HandleRequest(context.Background(), WakeRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Status != "starting" {
		t.Errorf("Expected status 'starting', got '%s'", response.Status)
	}

	if response.DesiredCount != 1 {
		t.Errorf("Expected DesiredCount=1, got %d", response.DesiredCount)
	}

	if response.PendingCount != 1 {
		t.Errorf("Expected PendingCount=1, got %d", response.PendingCount)
	}
}

// TestWakeServiceNotFound verifies error handling when service doesn't exist
func TestWakeServiceNotFound(t *testing.T) {
	mockECS := &MockECSClient{
		DescribeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []types.Service{}, // Empty list
			}, nil
		},
	}

	handler := NewHandlerWithClient(mockECS, "test-cluster", "non-existent-service")

	_, err := handler.HandleRequest(context.Background(), WakeRequest{})
	if err == nil {
		t.Fatal("Expected error when service not found, got nil")
	}

	expectedError := "service non-existent-service not found in cluster test-cluster"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestWakeWithRequestOverrides verifies cluster/service name can be overridden
func TestWakeWithRequestOverrides(t *testing.T) {
	mockECS := &MockECSClient{
		DescribeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			if *params.Cluster != "override-cluster" {
				t.Errorf("Expected cluster 'override-cluster', got '%s'", *params.Cluster)
			}
			if params.Services[0] != "override-service" {
				t.Errorf("Expected service 'override-service', got '%s'", params.Services[0])
			}
			return &ecs.DescribeServicesOutput{
				Services: []types.Service{
					{
						ServiceName:  stringPtr("override-service"),
						DesiredCount: int32(0),
						RunningCount: int32(0),
						PendingCount: int32(0),
					},
				},
			}, nil
		},
		UpdateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			return &ecs.UpdateServiceOutput{}, nil
		},
	}

	handler := NewHandlerWithClient(mockECS, "default-cluster", "default-service")

	request := WakeRequest{
		ClusterName: "override-cluster",
		ServiceName: "override-service",
	}

	_, err := handler.HandleRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
