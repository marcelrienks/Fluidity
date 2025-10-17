package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with structured logging
type Logger struct {
	*logrus.Logger
	component string
}

// NewLogger creates a new logger instance for a component
func NewLogger(component string) *Logger {
	logger := logrus.New()
	
	// Set default log level to Info
	logger.SetLevel(logrus.InfoLevel)
	
	// Use JSON formatter for structured logs
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	
	// Output to stdout
	logger.SetOutput(os.Stdout)
	
	return &Logger{
		Logger:    logger,
		component: component,
	}
}

// WithComponent creates a logger entry with component field
func (l *Logger) WithComponent() *logrus.Entry {
	return l.WithField("component", l.component)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) {
	switch level {
	case "debug":
		l.Logger.SetLevel(logrus.DebugLevel)
	case "info":
		l.Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		l.Logger.SetLevel(logrus.WarnLevel)
	case "error":
		l.Logger.SetLevel(logrus.ErrorLevel)
	default:
		l.Logger.SetLevel(logrus.InfoLevel)
	}
}

// Info logs an info message with component context
func (l *Logger) Info(msg string, fields ...interface{}) {
	entry := l.WithComponent()
	if len(fields) > 0 {
		entry = l.addFields(entry, fields...)
	}
	entry.Info(msg)
}

// Error logs an error message with component context
func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	entry := l.WithComponent().WithError(err)
	if len(fields) > 0 {
		entry = l.addFields(entry, fields...)
	}
	entry.Error(msg)
}

// Warn logs a warning message with component context
func (l *Logger) Warn(msg string, fields ...interface{}) {
	entry := l.WithComponent()
	if len(fields) > 0 {
		entry = l.addFields(entry, fields...)
	}
	entry.Warn(msg)
}

// Debug logs a debug message with component context
func (l *Logger) Debug(msg string, fields ...interface{}) {
	entry := l.WithComponent()
	if len(fields) > 0 {
		entry = l.addFields(entry, fields...)
	}
	entry.Debug(msg)
}

// addFields adds key-value pairs as fields to the log entry
func (l *Logger) addFields(entry *logrus.Entry, fields ...interface{}) *logrus.Entry {
	if len(fields)%2 != 0 {
		fields = append(fields, "")
	}
	
	for i := 0; i < len(fields); i += 2 {
		if key, ok := fields[i].(string); ok {
			entry = entry.WithField(key, fields[i+1])
		}
	}
	
	return entry
}