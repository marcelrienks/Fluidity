package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"fluidity/internal/shared/logging"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Emitter manages CloudWatch metrics emission
type Emitter struct {
	config       *Config
	client       *cloudwatch.Client
	logger       *logging.Logger
	activeConns  atomic.Int64
	lastActivity atomic.Int64 // Unix epoch seconds
	ctx          context.Context
	cancel       context.CancelFunc
	emitTicker   *time.Ticker
}

// NewEmitter creates a new metrics emitter
func NewEmitter(cfg *Config, logger *logging.Logger) (*Emitter, error) {
	if cfg == nil {
		cfg = &Config{Enabled: false}
	}

	if logger == nil {
		logger = logging.NewLogger("metrics")
	}

	// If disabled, return emitter that does nothing
	if !cfg.Enabled {
		logger.Info("CloudWatch metrics disabled")
		return &Emitter{
			config: cfg,
			logger: logger,
		}, nil
	}

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		logger.Warn("Failed to load AWS config, metrics will be disabled", "error", err.Error())
		cfg.Enabled = false
		return &Emitter{
			config: cfg,
			logger: logger,
		}, nil
	}

	// Create CloudWatch client
	client := cloudwatch.NewFromConfig(awsConfig)

	ctx, cancel := context.WithCancel(context.Background())

	emitter := &Emitter{
		config:     cfg,
		client:     client,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		emitTicker: time.NewTicker(cfg.EmitInterval),
	}

	// Initialize last activity to now
	emitter.lastActivity.Store(time.Now().Unix())

	logger.Info("CloudWatch metrics emitter initialized",
		"namespace", cfg.Namespace,
		"region", cfg.Region,
		"emitInterval", cfg.EmitInterval,
	)

	return emitter, nil
}

// Start begins emitting metrics at the configured interval
func (e *Emitter) Start() {
	if !e.config.Enabled {
		return
	}

	e.logger.Info("Starting metrics emission")

	go func() {
		// Emit initial metrics
		e.emitMetrics()

		for {
			select {
			case <-e.ctx.Done():
				e.logger.Info("Metrics emission stopped")
				return
			case <-e.emitTicker.C:
				e.emitMetrics()
			}
		}
	}()
}

// Stop stops the metrics emitter
func (e *Emitter) Stop() {
	if !e.config.Enabled {
		return
	}

	e.logger.Info("Stopping metrics emitter")
	e.cancel()
	e.emitTicker.Stop()

	// Emit final metrics
	e.emitMetrics()
}

// IncrementConnections increments the active connections counter
func (e *Emitter) IncrementConnections() {
	if !e.config.Enabled {
		return
	}

	count := e.activeConns.Add(1)
	e.UpdateLastActivity()
	e.logger.Debug("Active connections incremented", "count", count)
}

// DecrementConnections decrements the active connections counter
func (e *Emitter) DecrementConnections() {
	if !e.config.Enabled {
		return
	}

	count := e.activeConns.Add(-1)
	if count < 0 {
		e.activeConns.Store(0)
		count = 0
	}
	e.UpdateLastActivity()
	e.logger.Debug("Active connections decremented", "count", count)
}

// GetActiveConnections returns the current active connections count
func (e *Emitter) GetActiveConnections() int64 {
	return e.activeConns.Load()
}

// UpdateLastActivity updates the last activity timestamp to now
func (e *Emitter) UpdateLastActivity() {
	if !e.config.Enabled {
		return
	}

	e.lastActivity.Store(time.Now().Unix())
}

// GetLastActivityTime returns the last activity timestamp
func (e *Emitter) GetLastActivityTime() time.Time {
	return time.Unix(e.lastActivity.Load(), 0)
}

// emitMetrics sends metrics to CloudWatch
func (e *Emitter) emitMetrics() {
	if !e.config.Enabled {
		return
	}

	activeConns := e.activeConns.Load()
	lastActivity := e.lastActivity.Load()

	e.logger.Debug("Emitting metrics",
		"activeConnections", activeConns,
		"lastActivityEpoch", lastActivity,
	)

	// Build metric data
	now := time.Now()
	metricData := []types.MetricDatum{
		{
			MetricName: aws.String("ActiveConnections"),
			Value:      aws.Float64(float64(activeConns)),
			Unit:       types.StandardUnitCount,
			Timestamp:  &now,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("ServiceName"),
					Value: aws.String(e.config.ServiceName),
				},
				{
					Name:  aws.String("ClusterName"),
					Value: aws.String(e.config.ClusterName),
				},
			},
		},
		{
			MetricName: aws.String("LastActivityEpochSeconds"),
			Value:      aws.Float64(float64(lastActivity)),
			Unit:       types.StandardUnitNone,
			Timestamp:  &now,
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("ServiceName"),
					Value: aws.String(e.config.ServiceName),
				},
				{
					Name:  aws.String("ClusterName"),
					Value: aws.String(e.config.ClusterName),
				},
			},
		},
	}

	// Send to CloudWatch (batched automatically by SDK)
	input := &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(e.config.Namespace),
		MetricData: metricData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := e.client.PutMetricData(ctx, input)
	if err != nil {
		e.logger.Warn("Failed to emit metrics to CloudWatch", "error", err.Error())
		// Don't fail the application - graceful degradation
		return
	}

	e.logger.Debug("Metrics emitted successfully",
		"activeConnections", activeConns,
		"lastActivityEpoch", lastActivity,
	)
}
