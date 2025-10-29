package metrics

import (
	"testing"
	"time"

	"fluidity/internal/shared/logging"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantEnabled bool
		wantErr     bool
	}{
		{
			name:        "default config",
			envVars:     map[string]string{},
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name: "custom config",
			envVars: map[string]string{
				"AWS_REGION":            "us-west-2",
				"METRICS_NAMESPACE":     "CustomNamespace",
				"ECS_SERVICE_NAME":      "my-service",
				"ECS_CLUSTER_NAME":      "my-cluster",
				"METRICS_EMIT_INTERVAL": "30s",
			},
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name: "disabled",
			envVars: map[string]string{
				"METRICS_ENABLED": "false",
			},
			wantEnabled: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			config, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if config.Enabled != tt.wantEnabled {
				t.Errorf("LoadConfig() Enabled = %v, want %v", config.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Region:       "us-east-1",
				Namespace:    "Fluidity",
				EmitInterval: 60 * time.Second,
				Enabled:      true,
			},
			wantErr: false,
		},
		{
			name: "disabled is valid",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing region",
			config: &Config{
				Namespace:    "Fluidity",
				EmitInterval: 60 * time.Second,
				Enabled:      true,
			},
			wantErr: true,
		},
		{
			name: "missing namespace",
			config: &Config{
				Region:       "us-east-1",
				EmitInterval: 60 * time.Second,
				Enabled:      true,
			},
			wantErr: true,
		},
		{
			name: "interval too short",
			config: &Config{
				Region:       "us-east-1",
				Namespace:    "Fluidity",
				EmitInterval: 5 * time.Second,
				Enabled:      true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmitterDisabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	logger := logging.NewLogger("test")
	emitter, err := NewEmitter(config, logger)
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}

	// All methods should be safe to call when disabled
	emitter.Start()
	emitter.IncrementConnections()
	emitter.DecrementConnections()
	emitter.UpdateLastActivity()
	emitter.Stop()

	// Should have no active connections since it's disabled
	if count := emitter.GetActiveConnections(); count != 0 {
		t.Errorf("GetActiveConnections() = %d, want 0", count)
	}
}

func TestActiveConnectionsTracking(t *testing.T) {
	config := &Config{
		Region:       "us-east-1",
		Namespace:    "Fluidity",
		EmitInterval: 60 * time.Second,
		Enabled:      true,
		ServiceName:  "test-service",
		ClusterName:  "test-cluster",
	}

	logger := logging.NewLogger("test")
	emitter, err := NewEmitter(config, logger)
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}
	defer emitter.Stop()

	// Initial count should be 0
	if count := emitter.GetActiveConnections(); count != 0 {
		t.Errorf("Initial GetActiveConnections() = %d, want 0", count)
	}

	// Increment connections
	emitter.IncrementConnections()
	if count := emitter.GetActiveConnections(); count != 1 {
		t.Errorf("After increment GetActiveConnections() = %d, want 1", count)
	}

	emitter.IncrementConnections()
	emitter.IncrementConnections()
	if count := emitter.GetActiveConnections(); count != 3 {
		t.Errorf("After 3 increments GetActiveConnections() = %d, want 3", count)
	}

	// Decrement connections
	emitter.DecrementConnections()
	if count := emitter.GetActiveConnections(); count != 2 {
		t.Errorf("After decrement GetActiveConnections() = %d, want 2", count)
	}

	emitter.DecrementConnections()
	emitter.DecrementConnections()
	if count := emitter.GetActiveConnections(); count != 0 {
		t.Errorf("After all decrements GetActiveConnections() = %d, want 0", count)
	}

	// Decrement below 0 should clamp to 0
	emitter.DecrementConnections()
	if count := emitter.GetActiveConnections(); count != 0 {
		t.Errorf("After decrement below 0 GetActiveConnections() = %d, want 0", count)
	}
}

func TestLastActivityTracking(t *testing.T) {
	config := &Config{
		Region:       "us-east-1",
		Namespace:    "Fluidity",
		EmitInterval: 60 * time.Second,
		Enabled:      true,
		ServiceName:  "test-service",
		ClusterName:  "test-cluster",
	}

	logger := logging.NewLogger("test")
	emitter, err := NewEmitter(config, logger)
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}
	defer emitter.Stop()

	// Get initial time
	initialTime := emitter.GetLastActivityTime()
	if initialTime.IsZero() {
		t.Error("Initial last activity time should not be zero")
	}

	// Wait a bit and update
	time.Sleep(1100 * time.Millisecond) // Wait longer to ensure time change
	emitter.UpdateLastActivity()

	// Should be updated
	updatedTime := emitter.GetLastActivityTime()
	if !updatedTime.After(initialTime) {
		t.Errorf("Updated time (%v) should be after initial time (%v)", updatedTime, initialTime)
	}

	// Increment should also update activity
	time.Sleep(1100 * time.Millisecond) // Wait longer to ensure time change
	beforeIncrement := emitter.GetLastActivityTime()
	emitter.IncrementConnections()
	afterIncrement := emitter.GetLastActivityTime()

	if !afterIncrement.After(beforeIncrement) || afterIncrement.Equal(beforeIncrement) {
		t.Errorf("IncrementConnections should update last activity time: before=%v, after=%v", beforeIncrement, afterIncrement)
	}

	// Decrement should also update activity
	time.Sleep(1100 * time.Millisecond) // Wait longer to ensure time change
	beforeDecrement := emitter.GetLastActivityTime()
	emitter.DecrementConnections()
	afterDecrement := emitter.GetLastActivityTime()

	if !afterDecrement.After(beforeDecrement) || afterDecrement.Equal(beforeDecrement) {
		t.Errorf("DecrementConnections should update last activity time: before=%v, after=%v", beforeDecrement, afterDecrement)
	}
}

func TestStartStop(t *testing.T) {
	config := &Config{
		Region:       "us-east-1",
		Namespace:    "Fluidity",
		EmitInterval: 100 * time.Millisecond, // Short interval for testing
		Enabled:      true,
		ServiceName:  "test-service",
		ClusterName:  "test-cluster",
	}

	logger := logging.NewLogger("test")
	emitter, err := NewEmitter(config, logger)
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}

	// Start emitter
	emitter.Start()

	// Wait for a couple emit cycles
	time.Sleep(250 * time.Millisecond)

	// Update some data
	emitter.IncrementConnections()
	emitter.IncrementConnections()

	// Stop should not panic
	emitter.Stop()
}

func TestConcurrentConnectionUpdates(t *testing.T) {
	config := &Config{
		Region:       "us-east-1",
		Namespace:    "Fluidity",
		EmitInterval: 60 * time.Second,
		Enabled:      true,
		ServiceName:  "test-service",
		ClusterName:  "test-cluster",
	}

	logger := logging.NewLogger("test")
	emitter, err := NewEmitter(config, logger)
	if err != nil {
		t.Fatalf("NewEmitter() error = %v", err)
	}
	defer emitter.Stop()

	// Concurrent increments
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				emitter.IncrementConnections()
			}
			done <- true
		}()
	}

	// Wait for all increments
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 1000 connections
	if count := emitter.GetActiveConnections(); count != 1000 {
		t.Errorf("After concurrent increments GetActiveConnections() = %d, want 1000", count)
	}

	// Concurrent decrements
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				emitter.DecrementConnections()
			}
			done <- true
		}()
	}

	// Wait for all decrements
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should be back to 0
	if count := emitter.GetActiveConnections(); count != 0 {
		t.Errorf("After concurrent decrements GetActiveConnections() = %d, want 0", count)
	}
}
