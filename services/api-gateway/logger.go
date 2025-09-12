package main

import (
	"askshop/shared/env"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// Debug is the most verbose logging level
	Debug LogLevel = iota
	// Info is the standard logging level
	Info
	// Warn is for warning messages
	Warn
	// Error is for error messages
	Error
	// Fatal is for critical errors
	Fatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogConfig holds the configuration for the logger
type LogConfig struct {
	// Level is the minimum level to log
	Level LogLevel
	// UseJSON determines if logs should be in JSON format
	UseJSON bool
	// UseColor determines if color should be used in logs
	UseColor bool
	// ServiceName is the name of the service
	ServiceName string
}

// NewLogConfig creates a new log configuration based on environment variables
func NewLogConfig() LogConfig {
	config := LogConfig{
		Level:       Info,
		UseJSON:     false,
		UseColor:    true,
		ServiceName: env.GetString("SERVICE_NAME", "api-gateway"),
	}

	// Check if production environment
	if env.GetString("ENV", "development") == "production" {
		config.Level = Info
		config.UseJSON = true
		config.UseColor = false
	}

	// Allow override via environment variables
	levelStr := env.GetString("LOG_LEVEL", "")
	if levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			config.Level = Debug
		case "INFO":
			config.Level = Info
		case "WARN":
			config.Level = Warn
		case "ERROR":
			config.Level = Error
		case "FATAL":
			config.Level = Fatal
		}
	}

	return config
}

// Logger is a structured logger
type Logger struct {
	config LogConfig
	fields map[string]interface{}
}

// NewLogger creates a new logger
func NewLogger(config LogConfig) *Logger {
	return &Logger{
		config: config,
		fields: make(map[string]interface{}),
	}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		config: l.config,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		config: l.config,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, msg string) {
	if level < l.config.Level {
		return
	}

	if l.config.UseJSON {
		// Create JSON log entry
		entry := map[string]interface{}{
			"level":     level.String(),
			"message":   msg,
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   l.config.ServiceName,
		}

		// Add fields
		for k, v := range l.fields {
			entry[k] = v
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Error marshaling log entry: %v", err)
			return
		}
		log.Println(string(jsonBytes))
	} else {
		// Format: [LEVEL] [SERVICE] [timestamp] message fields...
		levelStr := level.String()
		if l.config.UseColor {
			// Add color based on level
			switch level {
			case Debug:
				levelStr = fmt.Sprintf("\033[36m%s\033[0m", levelStr) // Cyan
			case Info:
				levelStr = fmt.Sprintf("\033[32m%s\033[0m", levelStr) // Green
			case Warn:
				levelStr = fmt.Sprintf("\033[33m%s\033[0m", levelStr) // Yellow
			case Error:
				levelStr = fmt.Sprintf("\033[31m%s\033[0m", levelStr) // Red
			case Fatal:
				levelStr = fmt.Sprintf("\033[35m%s\033[0m", levelStr) // Magenta
			}
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logMsg := fmt.Sprintf("[%s] [%s] [%s] %s", levelStr, l.config.ServiceName, timestamp, msg)

		// Add fields if present
		if len(l.fields) > 0 {
			fieldStrs := make([]string, 0, len(l.fields))
			for k, v := range l.fields {
				fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
			}
			logMsg += " " + strings.Join(fieldStrs, " ")
		}

		log.Println(logMsg)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.log(Debug, msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.log(Info, msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.log(Warn, msg)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.log(Error, msg)
}

// Fatal logs a fatal message
func (l *Logger) Fatal(msg string) {
	l.log(Fatal, msg)
}

// TruncateString truncates a string if it's longer than max
func TruncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "... [truncated]"
}

// FilterSensitiveData removes sensitive data from the input
func FilterSensitiveData(input string) string {
	// Define patterns to filter
	patterns := []struct {
		pattern string
		replace string
	}{
		{`"password"\s*:\s*"[^"]*"`, `"password":"[FILTERED]"`},
		{`"accessToken"\s*:\s*"[^"]*"`, `"accessToken":"[FILTERED]"`},
		{`"refreshToken"\s*:\s*"[^"]*"`, `"refreshToken":"[FILTERED]"`},
		// Add more patterns as needed
	}

	result := input
	for _, p := range patterns {
		result = strings.Replace(result, p.pattern, p.replace, -1)
	}
	return result
}

// PrettifyJSON formats JSON for readability
func PrettifyJSON(rawJSON []byte) string {
	var prettyJSON bytes.Buffer

	// Try to parse as JSON
	err := json.Indent(&prettyJSON, rawJSON, "", "  ")
	if err != nil {
		// Not valid JSON, return as is with length info
		if len(rawJSON) > 100 {
			return fmt.Sprintf("%s... [%d bytes total]", string(rawJSON[:100]), len(rawJSON))
		}
		return string(rawJSON)
	}

	result := prettyJSON.String()

	// If the result is too long, truncate it
	if len(result) > 1000 {
		return result[:1000] + "... [truncated]"
	}

	return result
}

// DumpBody reads and restores the request body
func DumpBody(body io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	// Restore the body for further processing
	return bodyBytes, io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
}

// LoggingResponseWriter captures the response status and body
type LoggingResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// Write captures the response body
func (w *LoggingResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString captures the response body from string
func (w *LoggingResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// WriteHeader captures the response status code
func (w *LoggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// NewLoggingResponseWriter creates a new logging response writer
func NewLoggingResponseWriter(w gin.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         200, // Default status is 200
	}
}
