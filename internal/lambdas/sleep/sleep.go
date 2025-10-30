package sleep

import (
	"context"
	"fmt"
	"time"

	"fluidity/internal/shared/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// SleepRequest represents the input to the Sleep Lambda
type SleepRequest struct {
	ClusterName        string `json:"cluster_name,omitempty"`
	ServiceName        string `json:"service_name,omitempty"`
	IdleThresholdMins  int    `json:"idle_threshold_mins,omitempty"`
	LookbackPeriodMins int    `json:"lookback_period_mins,omitempty"`
}

// SleepResponse represents the output from the Sleep Lambda
type SleepResponse struct {
	Action               string  `json:"action"`
	DesiredCount         int32   `json:"desiredCount,omitempty"`
	RunningCount         int32   `json:"runningCount,omitempty"`
	AvgActiveConnections float64 `json:"avgActiveConnections,omitempty"`
	IdleDurationSeconds  int64   `json:"idleDurationSeconds,omitempty"`
	Message              string  `json:"message"`
}

// ECSClient interface for testing
type ECSClient interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

// CloudWatchClient interface for testing
type CloudWatchClient interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

// Handler processes sleep requests
type Handler struct {
	ecsClient          ECSClient
	cloudWatchClient   CloudWatchClient
	clusterName        string
	serviceName        string
	idleThresholdMins  int
	lookbackPeriodMins int
	logger             *logger.Logger
}

// NewHandler creates a new sleep handler with AWS SDK clients
func NewHandler(ctx context.Context, clusterName, serviceName string, idleThresholdMins, lookbackPeriodMins int) (*Handler, error) {
	log := logger.NewFromEnv()

	log.Info("Initializing Sleep Lambda handler", map[string]interface{}{
		"clusterName":        clusterName,
		"serviceName":        serviceName,
		"idleThresholdMins":  idleThresholdMins,
		"lookbackPeriodMins": lookbackPeriodMins,
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

	if idleThresholdMins <= 0 {
		idleThresholdMins = 15 // Default: 15 minutes
		log.Debug("Using default idleThresholdMins", map[string]interface{}{"value": 15})
	}

	if lookbackPeriodMins <= 0 {
		lookbackPeriodMins = 10 // Default: 10 minutes
		log.Debug("Using default lookbackPeriodMins", map[string]interface{}{"value": 10})
	}

	log.Info("Sleep Lambda handler initialized successfully")

	return &Handler{
		ecsClient:          ecs.NewFromConfig(cfg),
		cloudWatchClient:   cloudwatch.NewFromConfig(cfg),
		clusterName:        clusterName,
		serviceName:        serviceName,
		idleThresholdMins:  idleThresholdMins,
		lookbackPeriodMins: lookbackPeriodMins,
		logger:             log,
	}, nil
}

// NewHandlerWithClients creates a new sleep handler with provided clients (for testing)
func NewHandlerWithClients(ecsClient ECSClient, cloudWatchClient CloudWatchClient, clusterName, serviceName string, idleThresholdMins, lookbackPeriodMins int) *Handler {
	if idleThresholdMins <= 0 {
		idleThresholdMins = 15
	}
	if lookbackPeriodMins <= 0 {
		lookbackPeriodMins = 10
	}

	return &Handler{
		ecsClient:          ecsClient,
		cloudWatchClient:   cloudWatchClient,
		clusterName:        clusterName,
		serviceName:        serviceName,
		idleThresholdMins:  idleThresholdMins,
		lookbackPeriodMins: lookbackPeriodMins,
		logger:             logger.New("info"),
	}
}

// HandleRequest processes the sleep request
func (h *Handler) HandleRequest(ctx context.Context, request SleepRequest) (*SleepResponse, error) {
	// Allow request to override parameters (for testing)
	clusterName := h.clusterName
	if request.ClusterName != "" {
		clusterName = request.ClusterName
	}

	serviceName := h.serviceName
	if request.ServiceName != "" {
		serviceName = request.ServiceName
	}

	idleThresholdMins := h.idleThresholdMins
	if request.IdleThresholdMins > 0 {
		idleThresholdMins = request.IdleThresholdMins
	}

	lookbackPeriodMins := h.lookbackPeriodMins
	if request.LookbackPeriodMins > 0 {
		lookbackPeriodMins = request.LookbackPeriodMins
	}

	h.logger.Info("Processing sleep request", map[string]interface{}{
		"clusterName":        clusterName,
		"serviceName":        serviceName,
		"idleThresholdMins":  idleThresholdMins,
		"lookbackPeriodMins": lookbackPeriodMins,
	})

	// Step 1: Check current service state
	describeInput := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	h.logger.Debug("Describing ECS service state")
	describeOutput, err := h.ecsClient.DescribeServices(ctx, describeInput)
	if err != nil {
		h.logger.Error("Failed to describe ECS service", err, map[string]interface{}{
			"clusterName": clusterName,
			"serviceName": serviceName,
		})
		return nil, fmt.Errorf("failed to describe ECS service: %w", err)
	}

	if len(describeOutput.Services) == 0 {
		h.logger.Error("ECS service not found", nil, map[string]interface{}{
			"clusterName": clusterName,
			"serviceName": serviceName,
		})
		return nil, fmt.Errorf("service %s not found in cluster %s", serviceName, clusterName)
	}

	service := describeOutput.Services[0]
	desiredCount := service.DesiredCount
	runningCount := service.RunningCount

	h.logger.Debug("Current service state", map[string]interface{}{
		"desiredCount": desiredCount,
		"runningCount": runningCount,
	})

	// Step 2: If service is already stopped, no action needed
	if desiredCount == 0 {
		h.logger.Info("Service is already stopped, no action needed")
		return &SleepResponse{
			Action:       "no_change",
			DesiredCount: 0,
			RunningCount: runningCount,
			Message:      "Service is already stopped (desiredCount=0)",
		}, nil
	}

	// Step 3: Query CloudWatch metrics
	h.logger.Debug("Querying CloudWatch metrics", map[string]interface{}{
		"lookbackPeriodMins": lookbackPeriodMins,
	})

	now := time.Now()
	startTime := now.Add(-time.Duration(lookbackPeriodMins) * time.Minute)
	endTime := now

	avgActiveConnections, lastActivityTime, err := h.getMetrics(ctx, startTime, endTime)
	if err != nil {
		h.logger.Error("Failed to get CloudWatch metrics", err, map[string]interface{}{
			"startTime": startTime,
			"endTime":   endTime,
		})
		return nil, fmt.Errorf("failed to get CloudWatch metrics: %w", err)
	}

	// Step 4: Calculate idle duration
	idleDurationSeconds := int64(0)
	if !lastActivityTime.IsZero() {
		idleDurationSeconds = int64(now.Sub(lastActivityTime).Seconds())
	}

	h.logger.Debug("Metrics analysis", map[string]interface{}{
		"avgActiveConnections": avgActiveConnections,
		"idleDurationSeconds":  idleDurationSeconds,
		"lastActivityTime":     lastActivityTime,
	})

	// Step 5: Check if service is idle
	idleThresholdSeconds := int64(idleThresholdMins * 60)
	isIdle := avgActiveConnections <= 0 && idleDurationSeconds >= idleThresholdSeconds

	// Step 6: If idle and running, scale down
	if isIdle {
		h.logger.Info("Service is idle, initiating scale down", map[string]interface{}{
			"idleDurationSeconds":  idleDurationSeconds,
			"idleThresholdSeconds": idleThresholdSeconds,
			"avgActiveConnections": avgActiveConnections,
		})
		updateInput := &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceName),
			DesiredCount: aws.Int32(0),
		}

		_, err = h.ecsClient.UpdateService(ctx, updateInput)
		if err != nil {
			h.logger.Error("Failed to update ECS service", err, map[string]interface{}{
				"clusterName": clusterName,
				"serviceName": serviceName,
			})
			return nil, fmt.Errorf("failed to update ECS service: %w", err)
		}

		h.logger.Info("Service scaled down successfully", map[string]interface{}{
			"idleDurationSeconds": idleDurationSeconds,
		})

		return &SleepResponse{
			Action:               "scaled_down",
			DesiredCount:         0,
			RunningCount:         runningCount,
			AvgActiveConnections: avgActiveConnections,
			IdleDurationSeconds:  idleDurationSeconds,
			Message:              fmt.Sprintf("Service scaled down due to inactivity (idle for %d seconds)", idleDurationSeconds),
		}, nil
	}

	// Step 7: Service is active, no action
	h.logger.Info("Service is active, no action needed", map[string]interface{}{
		"avgActiveConnections": avgActiveConnections,
		"idleDurationSeconds":  idleDurationSeconds,
		"desiredCount":         desiredCount,
		"runningCount":         runningCount,
	})

	return &SleepResponse{
		Action:               "no_change",
		DesiredCount:         desiredCount,
		RunningCount:         runningCount,
		AvgActiveConnections: avgActiveConnections,
		IdleDurationSeconds:  idleDurationSeconds,
		Message:              fmt.Sprintf("Service is active (avg connections: %.2f, idle: %d seconds)", avgActiveConnections, idleDurationSeconds),
	}, nil
}

// getMetrics queries CloudWatch for active connections and last activity metrics
func (h *Handler) getMetrics(ctx context.Context, startTime, endTime time.Time) (avgActiveConnections float64, lastActivityTime time.Time, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []cloudwatchtypes.MetricDataQuery{
			{
				Id: aws.String("active_connections"),
				MetricStat: &cloudwatchtypes.MetricStat{
					Metric: &cloudwatchtypes.Metric{
						Namespace:  aws.String("Fluidity"),
						MetricName: aws.String("ActiveConnections"),
						Dimensions: []cloudwatchtypes.Dimension{
							{
								Name:  aws.String("Service"),
								Value: aws.String("fluidity-server"),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String("Average"),
				},
			},
			{
				Id: aws.String("last_activity"),
				MetricStat: &cloudwatchtypes.MetricStat{
					Metric: &cloudwatchtypes.Metric{
						Namespace:  aws.String("Fluidity"),
						MetricName: aws.String("LastActivityEpochSeconds"),
						Dimensions: []cloudwatchtypes.Dimension{
							{
								Name:  aws.String("Service"),
								Value: aws.String("fluidity-server"),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String("Maximum"),
				},
			},
		},
	}

	output, err := h.cloudWatchClient.GetMetricData(ctx, input)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("GetMetricData failed: %w", err)
	}

	// Parse active connections metric
	for _, result := range output.MetricDataResults {
		if aws.ToString(result.Id) == "active_connections" && len(result.Values) > 0 {
			// Calculate average of all values in the lookback period
			sum := 0.0
			for _, val := range result.Values {
				sum += val
			}
			avgActiveConnections = sum / float64(len(result.Values))
		}

		if aws.ToString(result.Id) == "last_activity" && len(result.Values) > 0 {
			// Get the maximum (most recent) last activity timestamp
			maxEpoch := int64(0)
			for _, val := range result.Values {
				epoch := int64(val)
				if epoch > maxEpoch {
					maxEpoch = epoch
				}
			}
			if maxEpoch > 0 {
				lastActivityTime = time.Unix(maxEpoch, 0)
			}
		}
	}

	return avgActiveConnections, lastActivityTime, nil
}
