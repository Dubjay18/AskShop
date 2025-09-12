package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"askshop/shared/env"
)

// Level represents the severity level for a log message
type Level int

const (
	// Debug level logs detailed information for debugging
	Debug Level = iota
	// Info level logs general operational information
	Info
	// Warn level logs non-critical issues
	Warn
	// Error level logs critical issues
	Error
	// Fatal level logs fatal issues and exits the program
	Fatal
)

var levelNames = map[Level]string{
	Debug: "DEBUG",
	Info:  "INFO ",
	Warn:  "WARN ",
	Error: "ERROR",
	Fatal: "FATAL",
}

// Config holds configuration for the logger
type Config struct {
	// MinLevel is the minimum level to log
	MinLevel Level
	// EnableColor determines if ANSI color codes should be used
	EnableColor bool
	// EnableJSON determines if logs should be formatted as JSON
	EnableJSON bool
	// EnableTimestamp determines if timestamp should be added to logs
	EnableTimestamp bool
	// EnableCaller determines if caller information should be added to logs
	EnableCaller bool
	// Output is where logs are written to
	Output io.Writer
	// ServiceName is the name of the service using this logger
	ServiceName string
}

// Logger represents a custom logger
type Logger struct {
	config Config
	logger *log.Logger
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	minLevel := Info
	if env.GetString("ENV", "production") != "production" {
		minLevel = Debug
	}

	return Config{
		MinLevel:        minLevel,
		EnableColor:     env.GetString("LOG_COLOR", "true") == "true",
		EnableJSON:      env.GetString("LOG_JSON", "false") == "true",
		EnableTimestamp: true,
		EnableCaller:    true,
		Output:          os.Stdout,
		ServiceName:     env.GetString("SERVICE_NAME", "app"),
	}
}

// New creates a new logger with the provided configuration
func New(config Config) *Logger {
	logger := log.New(config.Output, "", 0)
	return &Logger{
		config: config,
		logger: logger,
	}
}

// Default creates a new logger with default configuration
func Default(serviceName string) *Logger {
	config := DefaultConfig()
	config.ServiceName = serviceName
	return New(config)
}

// WithField returns a copy of the logger with a field added to it
func (l *Logger) WithField(key string, value interface{}) *Logger {
	// This is a simplified implementation
	// In a production version, you'd create a new logger with contextual fields
	newLogger := &Logger{
		config: l.config,
		logger: l.logger,
	}
	return newLogger
}

// WithFields returns a copy of the logger with multiple fields added to it
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// This is a simplified implementation
	// In a production version, you'd create a new logger with contextual fields
	newLogger := &Logger{
		config: l.config,
		logger: l.logger,
	}
	return newLogger
}

// Log logs a message at the specified level
func (l *Logger) Log(level Level, v ...interface{}) {
	if level < l.config.MinLevel {
		return
	}

	// Build the log message
	var sb strings.Builder

	// Add timestamp
	if l.config.EnableTimestamp {
		ts := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
		sb.WriteString(ts)
		sb.WriteString(" ")
	}

	// Add level
	levelColor := ""
	resetColor := ""
	if l.config.EnableColor {
		levelColor = getLevelColor(level)
		resetColor = "\033[0m"
	}
	sb.WriteString(levelColor)
	sb.WriteString(levelNames[level])
	sb.WriteString(resetColor)
	sb.WriteString(" ")

	// Add service name
	sb.WriteString("[")
	sb.WriteString(l.config.ServiceName)
	sb.WriteString("] ")

	// Add caller information
	if l.config.EnableCaller {
		_, file, line, ok := runtime.Caller(2) // Skip this function and the logging function
		if ok {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			sb.WriteString(short)
			sb.WriteString(":")
			sb.WriteString(fmt.Sprintf("%d", line))
			sb.WriteString(" ")
		}
	}

	// Add the log message
	sb.WriteString(fmt.Sprint(v...))

	// Write the log
	l.logger.Println(sb.String())

	// Exit if fatal
	if level == Fatal {
		os.Exit(1)
	}
}

// Logf logs a formatted message at the specified level
func (l *Logger) Logf(level Level, format string, v ...interface{}) {
	if level < l.config.MinLevel {
		return
	}
	l.Log(level, fmt.Sprintf(format, v...))
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	l.Log(Debug, v...)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Logf(Debug, format, v...)
}

// Info logs an info message
func (l *Logger) Info(v ...interface{}) {
	l.Log(Info, v...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Logf(Info, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(v ...interface{}) {
	l.Log(Warn, v...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Logf(Warn, format, v...)
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	l.Log(Error, v...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Logf(Error, format, v...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(v ...interface{}) {
	l.Log(Fatal, v...)
}

// Fatalf logs a formatted fatal message and exits the program
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Logf(Fatal, format, v...)
}

func getLevelColor(level Level) string {
	switch level {
	case Debug:
		return "\033[36m" // Cyan
	case Info:
		return "\033[32m" // Green
	case Warn:
		return "\033[33m" // Yellow
	case Error:
		return "\033[31m" // Red
	case Fatal:
		return "\033[35m" // Magenta
	default:
		return ""
	}
}
