package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with structured logging
type Logger struct {
	*logrus.Logger
	component string
}

// OrderedJSONFormatter formats logs as JSON with consistent field ordering
type OrderedJSONFormatter struct {
	TimestampFormat string
}

// Format renders a single log entry with consistent field order
func (f *OrderedJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Fixed order: timestamp, level, component, message, then custom fields
	var buf bytes.Buffer
	buf.WriteString("{")

	// 1. Timestamp
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02T15:04:05.000Z"
	}
	fmt.Fprintf(&buf, `"timestamp":"%s",`, entry.Time.Format(timestampFormat))

	// 2. Level
	fmt.Fprintf(&buf, `"level":"%s",`, entry.Level.String())

	// 3. Component (if present)
	if component, ok := entry.Data["component"]; ok {
		componentJSON, _ := json.Marshal(component)
		fmt.Fprintf(&buf, `"component":%s,`, componentJSON)
		delete(entry.Data, "component") // Remove so we don't duplicate later
	}

	// 4. Message
	messageJSON, _ := json.Marshal(entry.Message)
	fmt.Fprintf(&buf, `"message":%s`, messageJSON)

	// 5. Error (if present from logrus.WithError, it's in Data with key "error")
	if err, ok := entry.Data[logrus.ErrorKey]; ok {
		// Handle error specially to ensure it's a string
		var errStr string
		if e, isErr := err.(error); isErr {
			errStr = e.Error()
		} else {
			errStr = fmt.Sprintf("%v", err)
		}
		errJSON, _ := json.Marshal(errStr)
		fmt.Fprintf(&buf, `,"error":%s`, errJSON)
		delete(entry.Data, logrus.ErrorKey)
	} // 6. All other custom fields in sorted order for consistency
	if len(entry.Data) > 0 {
		for key, value := range entry.Data {
			valueJSON, _ := json.Marshal(value)
			fmt.Fprintf(&buf, `,"%s":%s`, key, valueJSON)
		}
	}

	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

// NewLogger creates a new logger instance for a component
func NewLogger(component string) *Logger {
	logger := logrus.New()

	// Set default log level to Info
	logger.SetLevel(logrus.InfoLevel)

	// Use custom ordered JSON formatter
	logger.SetFormatter(&OrderedJSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
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
