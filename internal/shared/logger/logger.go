package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger provides structured logging for Lambda functions
type Logger struct {
	level   LogLevel
	context map[string]interface{}
}

// New creates a new Logger with the specified log level
func New(level string) *Logger {
	logLevel := LevelInfo
	switch level {
	case "DEBUG", "debug":
		logLevel = LevelDebug
	case "INFO", "info":
		logLevel = LevelInfo
	case "WARN", "warn":
		logLevel = LevelWarn
	case "ERROR", "error":
		logLevel = LevelError
	}

	return &Logger{
		level:   logLevel,
		context: make(map[string]interface{}),
	}
}

// NewFromEnv creates a new Logger using the LOG_LEVEL environment variable
func NewFromEnv() *Logger {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}
	return New(level)
}

// WithContext adds context fields that will be included in all subsequent log entries
func (l *Logger) WithContext(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:   l.level,
		context: make(map[string]interface{}),
	}

	// Copy existing context
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// Add new context
	newLogger.context[key] = value

	return newLogger
}

// WithFields adds multiple context fields at once
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:   l.level,
		context: make(map[string]interface{}),
	}

	// Copy existing context
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.context[k] = v
	}

	return newLogger
}

// log writes a structured log entry to stdout
func (l *Logger) log(level LogLevel, message string, err error, additionalContext map[string]interface{}) {
	// Check if this log level should be output
	if !l.shouldLog(level) {
		return
	}

	// Merge contexts
	context := make(map[string]interface{})
	for k, v := range l.context {
		context[k] = v
	}
	for k, v := range additionalContext {
		context[k] = v
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Context:   context,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Marshal to JSON
	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to basic logging if JSON marshaling fails
		fmt.Fprintf(os.Stdout, "{\"timestamp\":\"%s\",\"level\":\"ERROR\",\"message\":\"Failed to marshal log entry\",\"error\":\"%v\"}\n",
			time.Now().UTC().Format(time.RFC3339), marshalErr)
		return
	}

	fmt.Fprintln(os.Stdout, string(jsonBytes))
}

// shouldLog determines if a log entry should be output based on the configured level
func (l *Logger) shouldLog(level LogLevel) bool {
	levelPriority := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}

	return levelPriority[level] >= levelPriority[l.level]
}

// Debug logs a debug-level message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var context map[string]interface{}
	if len(fields) > 0 {
		context = fields[0]
	}
	l.log(LevelDebug, message, nil, context)
}

// Info logs an info-level message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var context map[string]interface{}
	if len(fields) > 0 {
		context = fields[0]
	}
	l.log(LevelInfo, message, nil, context)
}

// Warn logs a warning-level message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var context map[string]interface{}
	if len(fields) > 0 {
		context = fields[0]
	}
	l.log(LevelWarn, message, nil, context)
}

// Error logs an error-level message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	var context map[string]interface{}
	if len(fields) > 0 {
		context = fields[0]
	}
	l.log(LevelError, message, err, context)
}

// WithError is a convenience method that returns a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return l.WithContext("error", err.Error())
}
