package sleep

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// Mock ECS client
type mockECSClient struct {
	describeServicesFunc func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	updateServiceFunc    func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

func (m *mockECSClient) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return m.describeServicesFunc(ctx, params, optFns...)
}

func (m *mockECSClient) UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	return m.updateServiceFunc(ctx, params, optFns...)
}

// Mock CloudWatch client
type mockCloudWatchClient struct {
	getMetricDataFunc func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

func (m *mockCloudWatchClient) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return m.getMetricDataFunc(ctx, params, optFns...)
}

// TestSleepWhenServiceAlreadyStopped tests that no action is taken when service is already stopped
func TestSleepWhenServiceAlreadyStopped(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 0,
						RunningCount: 0,
						PendingCount: 0,
					},
				},
			}, nil
		},
	}

	mockCW := &mockCloudWatchClient{}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	response, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Action != "no_change" {
		t.Errorf("Expected action 'no_change', got: %s", response.Action)
	}

	if response.DesiredCount != 0 {
		t.Errorf("Expected desiredCount 0, got: %d", response.DesiredCount)
	}
}

// TestSleepWhenServiceIsIdle tests scaling down when service is idle
func TestSleepWhenServiceIsIdle(t *testing.T) {
	updateCalled := false

	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 1,
						RunningCount: 1,
						PendingCount: 0,
					},
				},
			}, nil
		},
		updateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			updateCalled = true
			if *params.DesiredCount != 0 {
				t.Errorf("Expected DesiredCount 0, got: %d", *params.DesiredCount)
			}
			return &ecs.UpdateServiceOutput{}, nil
		},
	}

	// Mock metrics showing no active connections and old last activity
	now := time.Now()
	lastActivity := now.Add(-20 * time.Minute) // 20 minutes ago

	mockCW := &mockCloudWatchClient{
		getMetricDataFunc: func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatchtypes.MetricDataResult{
					{
						Id:     aws.String("active_connections"),
						Values: []float64{0.0, 0.0, 0.0},
					},
					{
						Id:     aws.String("last_activity"),
						Values: []float64{float64(lastActivity.Unix())},
					},
				},
			}, nil
		},
	}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	response, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !updateCalled {
		t.Error("Expected UpdateService to be called")
	}

	if response.Action != "scaled_down" {
		t.Errorf("Expected action 'scaled_down', got: %s", response.Action)
	}

	if response.DesiredCount != 0 {
		t.Errorf("Expected desiredCount 0, got: %d", response.DesiredCount)
	}

	if response.AvgActiveConnections != 0.0 {
		t.Errorf("Expected avgActiveConnections 0, got: %f", response.AvgActiveConnections)
	}
}

// TestSleepWhenServiceIsActive tests no action when service has active connections
func TestSleepWhenServiceIsActive(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 1,
						RunningCount: 1,
						PendingCount: 0,
					},
				},
			}, nil
		},
	}

	// Mock metrics showing active connections
	now := time.Now()
	mockCW := &mockCloudWatchClient{
		getMetricDataFunc: func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatchtypes.MetricDataResult{
					{
						Id:     aws.String("active_connections"),
						Values: []float64{2.0, 3.0, 2.5}, // Average = 2.5
					},
					{
						Id:     aws.String("last_activity"),
						Values: []float64{float64(now.Unix())},
					},
				},
			}, nil
		},
	}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	response, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Action != "no_change" {
		t.Errorf("Expected action 'no_change', got: %s", response.Action)
	}

	if response.DesiredCount != 1 {
		t.Errorf("Expected desiredCount 1, got: %d", response.DesiredCount)
	}

	if response.AvgActiveConnections != 2.5 {
		t.Errorf("Expected avgActiveConnections 2.5, got: %f", response.AvgActiveConnections)
	}
}

// TestSleepWhenIdleButBelowThreshold tests no action when idle but below threshold
func TestSleepWhenIdleButBelowThreshold(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 1,
						RunningCount: 1,
						PendingCount: 0,
					},
				},
			}, nil
		},
	}

	// Mock metrics showing no connections but recent activity (only 5 minutes ago)
	now := time.Now()
	lastActivity := now.Add(-5 * time.Minute)

	mockCW := &mockCloudWatchClient{
		getMetricDataFunc: func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatchtypes.MetricDataResult{
					{
						Id:     aws.String("active_connections"),
						Values: []float64{0.0, 0.0, 0.0},
					},
					{
						Id:     aws.String("last_activity"),
						Values: []float64{float64(lastActivity.Unix())},
					},
				},
			}, nil
		},
	}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	response, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Action != "no_change" {
		t.Errorf("Expected action 'no_change', got: %s", response.Action)
	}

	if response.IdleDurationSeconds >= 900 { // 15 minutes
		t.Errorf("Expected idle duration < 900 seconds, got: %d", response.IdleDurationSeconds)
	}
}

// TestSleepServiceNotFound tests error when service doesn't exist
func TestSleepServiceNotFound(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{}, // Empty - service not found
			}, nil
		},
	}

	mockCW := &mockCloudWatchClient{}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	_, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err == nil {
		t.Fatal("Expected error for non-existent service, got nil")
	}

	expectedError := "service test-service not found in cluster test-cluster"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %v", expectedError, err)
	}
}

// TestSleepWithRequestOverrides tests that request parameters override handler defaults
func TestSleepWithRequestOverrides(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			// Verify overridden cluster/service names are used
			if *params.Cluster != "override-cluster" {
				t.Errorf("Expected cluster 'override-cluster', got: %s", *params.Cluster)
			}
			if params.Services[0] != "override-service" {
				t.Errorf("Expected service 'override-service', got: %s", params.Services[0])
			}

			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 0,
						RunningCount: 0,
						PendingCount: 0,
					},
				},
			}, nil
		},
	}

	mockCW := &mockCloudWatchClient{}

	handler := NewHandlerWithClients(mockECS, mockCW, "default-cluster", "default-service", 15, 10)

	request := SleepRequest{
		ClusterName:        "override-cluster",
		ServiceName:        "override-service",
		IdleThresholdMins:  30,
		LookbackPeriodMins: 20,
	}

	_, err := handler.HandleRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// TestSleepCloudWatchError tests handling of CloudWatch API errors
func TestSleepCloudWatchError(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 1,
						RunningCount: 1,
						PendingCount: 0,
					},
				},
			}, nil
		},
	}

	mockCW := &mockCloudWatchClient{
		getMetricDataFunc: func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
			return nil, fmt.Errorf("CloudWatch API error")
		},
	}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	_, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err == nil {
		t.Fatal("Expected error from CloudWatch API, got nil")
	}
}

// TestSleepECSUpdateError tests handling of ECS UpdateService errors
func TestSleepECSUpdateError(t *testing.T) {
	mockECS := &mockECSClient{
		describeServicesFunc: func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						DesiredCount: 1,
						RunningCount: 1,
						PendingCount: 0,
					},
				},
			}, nil
		},
		updateServiceFunc: func(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
			return nil, fmt.Errorf("ECS UpdateService error")
		},
	}

	now := time.Now()
	lastActivity := now.Add(-20 * time.Minute)

	mockCW := &mockCloudWatchClient{
		getMetricDataFunc: func(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatchtypes.MetricDataResult{
					{
						Id:     aws.String("active_connections"),
						Values: []float64{0.0},
					},
					{
						Id:     aws.String("last_activity"),
						Values: []float64{float64(lastActivity.Unix())},
					},
				},
			}, nil
		},
	}

	handler := NewHandlerWithClients(mockECS, mockCW, "test-cluster", "test-service", 15, 10)

	_, err := handler.HandleRequest(context.Background(), SleepRequest{})
	if err == nil {
		t.Fatal("Expected error from ECS UpdateService, got nil")
	}
}
