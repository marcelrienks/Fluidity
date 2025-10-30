package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"fluidity/internal/lambdas/sleep"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	// Get cluster and service names from environment variables
	clusterName := os.Getenv("ECS_CLUSTER_NAME")
	if clusterName == "" {
		fmt.Println("Error: ECS_CLUSTER_NAME environment variable is required")
		os.Exit(1)
	}

	serviceName := os.Getenv("ECS_SERVICE_NAME")
	if serviceName == "" {
		fmt.Println("Error: ECS_SERVICE_NAME environment variable is required")
		os.Exit(1)
	}

	// Get idle threshold (optional, defaults to 15 minutes)
	idleThresholdMins := 15
	if idleThresholdStr := os.Getenv("IDLE_THRESHOLD_MINUTES"); idleThresholdStr != "" {
		if val, err := strconv.Atoi(idleThresholdStr); err == nil && val > 0 {
			idleThresholdMins = val
		}
	}

	// Get lookback period (optional, defaults to 10 minutes)
	lookbackPeriodMins := 10
	if lookbackPeriodStr := os.Getenv("LOOKBACK_PERIOD_MINUTES"); lookbackPeriodStr != "" {
		if val, err := strconv.Atoi(lookbackPeriodStr); err == nil && val > 0 {
			lookbackPeriodMins = val
		}
	}

	// Initialize handler once at cold start
	handler, err := sleep.NewHandler(context.Background(), clusterName, serviceName, idleThresholdMins, lookbackPeriodMins)
	if err != nil {
		fmt.Printf("Failed to initialize handler: %v\n", err)
		os.Exit(1)
	}

	// Start Lambda runtime
	lambda.Start(handler.HandleRequest)
}
