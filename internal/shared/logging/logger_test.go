package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-component")

	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}

	if logger.component != "test-component" {
		t.Errorf("Expected component 'test-component', got '%s'", logger.component)
	}

	if logger.Logger.Level != logrus.InfoLevel {
		t.Errorf("Expected default log level Info, got %v", logger.Logger.Level)
	}
}

func TestSetLevel(t *testing.T) {
	logger := NewLogger("test")

	tests := []struct {
		input    string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"invalid", logrus.InfoLevel}, // Should default to Info
	}

	for _, tt := range tests {
		logger.SetLevel(tt.input)
		if logger.Logger.Level != tt.expected {
			t.Errorf("SetLevel(%s): expected %v, got %v", tt.input, tt.expected, logger.Logger.Level)
		}
	}
}

func TestOrderedJSONFormatterBasic(t *testing.T) {
	formatter := &OrderedJSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
	}

	logger := logrus.New()
	logger.SetFormatter(formatter)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.WithField("component", "test").Info("Test message")

	output := buf.String()

	// Parse JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info', got '%v'", logEntry["level"])
	}
	if logEntry["message"] != "Test message" {
		t.Errorf("Expected message 'Test message', got '%v'", logEntry["message"])
	}
	if logEntry["component"] != "test" {
		t.Errorf("Expected component 'test', got '%v'", logEntry["component"])
	}

	// Verify field order in raw string
	if !strings.HasPrefix(output, `{"timestamp":`) {
		t.Error("Timestamp should be first field")
	}
}

func TestOrderedJSONFormatterWithError(t *testing.T) {
	formatter := &OrderedJSONFormatter{}

	logger := logrus.New()
	logger.SetFormatter(formatter)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	testErr := errors.New("test error")
	logger.WithError(testErr).WithField("component", "test").Error("Error occurred")

	output := buf.String()

	// Parse JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify error field
	if logEntry["error"] != "test error" {
		t.Errorf("Expected error 'test error', got '%v'", logEntry["error"])
	}
	if logEntry["level"] != "error" {
		t.Errorf("Expected level 'error', got '%v'", logEntry["level"])
	}
}

func TestLoggerInfoMethod(t *testing.T) {
	logger := NewLogger("info-test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	logger.Info("Info message")

	output := buf.String()

	if !strings.Contains(output, "Info message") {
		t.Error("Output should contain 'Info message'")
	}
	if !strings.Contains(output, `"component":"info-test"`) {
		t.Error("Output should contain component field")
	}
	if !strings.Contains(output, `"level":"info"`) {
		t.Error("Output should contain info level")
	}
}

func TestLoggerInfoWithFields(t *testing.T) {
	logger := NewLogger("test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	logger.Info("Message with fields", "key1", "value1", "key2", 42)

	output := buf.String()

	var logEntry map[string]interface{}
	json.Unmarshal([]byte(output), &logEntry)

	if logEntry["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", logEntry["key1"])
	}
	// JSON numbers are float64
	if logEntry["key2"] != float64(42) {
		t.Errorf("Expected key2=42, got '%v'", logEntry["key2"])
	}
}

func TestLoggerErrorMethod(t *testing.T) {
	logger := NewLogger("error-test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	testErr := errors.New("something went wrong")
	logger.Error("Error occurred", testErr)

	output := buf.String()

	var logEntry map[string]interface{}
	json.Unmarshal([]byte(output), &logEntry)

	if logEntry["error"] != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got '%v'", logEntry["error"])
	}
	if logEntry["level"] != "error" {
		t.Errorf("Expected level 'error', got '%v'", logEntry["level"])
	}
}

func TestLoggerErrorWithFields(t *testing.T) {
	logger := NewLogger("test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	testErr := errors.New("test error")
	logger.Error("Failed operation", testErr, "operation", "save", "retry", 3)

	output := buf.String()

	var logEntry map[string]interface{}
	json.Unmarshal([]byte(output), &logEntry)

	if logEntry["operation"] != "save" {
		t.Errorf("Expected operation='save', got '%v'", logEntry["operation"])
	}
	if logEntry["retry"] != float64(3) {
		t.Errorf("Expected retry=3, got '%v'", logEntry["retry"])
	}
}

func TestLoggerWarnMethod(t *testing.T) {
	logger := NewLogger("warn-test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	logger.Warn("Warning message")

	output := buf.String()

	if !strings.Contains(output, `"level":"warning"`) {
		t.Error("Output should contain warning level")
	}
	if !strings.Contains(output, "Warning message") {
		t.Error("Output should contain warning message")
	}
}

func TestLoggerDebugMethod(t *testing.T) {
	logger := NewLogger("debug-test")
	logger.SetLevel("debug")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	logger.Debug("Debug message")

	output := buf.String()

	if !strings.Contains(output, `"level":"debug"`) {
		t.Error("Output should contain debug level")
	}
	if !strings.Contains(output, "Debug message") {
		t.Error("Output should contain debug message")
	}
}

func TestLoggerDebugNotLoggedAtInfoLevel(t *testing.T) {
	logger := NewLogger("test")
	logger.SetLevel("info") // Default level

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	logger.Debug("This should not be logged")

	output := buf.String()

	if output != "" {
		t.Error("Debug message should not be logged at Info level")
	}
}

func TestAddFieldsWithOddNumberOfArgs(t *testing.T) {
	logger := NewLogger("test")

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)

	// Odd number of fields - should handle gracefully
	logger.Info("Message", "key1", "value1", "key2")

	output := buf.String()

	var logEntry map[string]interface{}
	json.Unmarshal([]byte(output), &logEntry)

	if logEntry["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", logEntry["key1"])
	}
	// key2 should have empty string value
	if logEntry["key2"] != "" {
		t.Errorf("Expected key2='', got '%v'", logEntry["key2"])
	}
}

func TestWithComponent(t *testing.T) {
	logger := NewLogger("my-component")

	entry := logger.WithComponent()

	if entry.Data["component"] != "my-component" {
		t.Errorf("Expected component 'my-component', got '%v'", entry.Data["component"])
	}
}

func TestOrderedJSONFormatterFieldOrder(t *testing.T) {
	formatter := &OrderedJSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
	}

	logger := logrus.New()
	logger.SetFormatter(formatter)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.WithField("component", "test").WithField("extra", "data").Info("Test")

	output := buf.String()

	// Verify order: timestamp, level, component, message should come first
	timestampIdx := strings.Index(output, `"timestamp"`)
	levelIdx := strings.Index(output, `"level"`)
	componentIdx := strings.Index(output, `"component"`)
	messageIdx := strings.Index(output, `"message"`)

	if timestampIdx == -1 || levelIdx == -1 || componentIdx == -1 || messageIdx == -1 {
		t.Fatal("Missing required fields")
	}

	if !(timestampIdx < levelIdx && levelIdx < componentIdx && componentIdx < messageIdx) {
		t.Error("Fields are not in correct order: timestamp, level, component, message")
	}
}
