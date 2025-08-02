package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents logging levels
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// String returns the string representation of the field
func (f Field) String() string {
	return fmt.Sprintf("%s=%v", f.Key, f.Value)
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithComponent(component string) Logger
	WithFields(fields ...Field) Logger
	SetLevel(level Level)
}

// SimpleLogger provides a simple implementation of Logger
type SimpleLogger struct {
	level     Level
	component string
	fields    []Field
	output    io.Writer
	logger    *log.Logger
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger(level Level, output io.Writer) *SimpleLogger {
	if output == nil {
		output = os.Stdout
	}
	
	return &SimpleLogger{
		level:  level,
		output: output,
		logger: log.New(output, "", 0), // We'll handle timestamps ourselves
		fields: make([]Field, 0),
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *SimpleLogger {
	return NewSimpleLogger(LevelInfo, os.Stdout)
}

// SetLevel sets the logging level
func (l *SimpleLogger) SetLevel(level Level) {
	l.level = level
}

// WithComponent returns a new logger with a component field
func (l *SimpleLogger) WithComponent(component string) Logger {
	return &SimpleLogger{
		level:     l.level,
		component: component,
		fields:    l.fields,
		output:    l.output,
		logger:    l.logger,
	}
}

// WithFields returns a new logger with additional fields
func (l *SimpleLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)
	
	return &SimpleLogger{
		level:     l.level,
		component: l.component,
		fields:    newFields,
		output:    l.output,
		logger:    l.logger,
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// log handles the actual logging
func (l *SimpleLogger) log(level Level, msg string, fields ...Field) {
	if level < l.level {
		return
	}
	
	// Build the log message
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := level.String()
	
	// Start with timestamp and level
	logMsg := fmt.Sprintf("[%s] %s", timestamp, levelStr)
	
	// Add component if present
	if l.component != "" {
		logMsg += fmt.Sprintf(" [%s]", l.component)
	}
	
	// Add the main message
	logMsg += fmt.Sprintf(" %s", msg)
	
	// Add persistent fields
	allFields := make([]Field, len(l.fields)+len(fields))
	copy(allFields, l.fields)
	copy(allFields[len(l.fields):], fields)
	
	// Add fields if present
	if len(allFields) > 0 {
		fieldStrs := make([]string, len(allFields))
		for i, field := range allFields {
			fieldStrs[i] = field.String()
		}
		logMsg += fmt.Sprintf(" {%s}", strings.Join(fieldStrs, ", "))
	}
	
	// Output the log message
	l.logger.Println(logMsg)
}

// Helper functions for creating fields
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}