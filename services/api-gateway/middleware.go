package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey is the key used for the request ID in the context
type RequestIDKey struct{}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header if present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context
		c.Set("request_id", requestID)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), RequestIDKey{}, requestID),
		)

		// Set response header
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs request information with configurable verbosity
func LoggingMiddleware() gin.HandlerFunc {
	// Initialize logger with environment-specific configuration
	logConfig := NewLogConfig()
	logger := NewLogger(logConfig)

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Get request ID
		requestID, _ := c.Get("request_id")
		reqLogger := logger.WithField("request_id", requestID)

		// Log basic request info at Info level
		reqLogger.Info(fmt.Sprintf("Request started: %s %s", c.Request.Method, c.Request.URL.Path))

		// Capture request body for debugging if needed
		var requestBody []byte
		var err error
		if c.Request.Body != nil && logConfig.Level == Debug {
			requestBody, c.Request.Body, err = DumpBody(c.Request.Body)
			if err != nil {
				reqLogger.Error(fmt.Sprintf("Error reading request body: %v", err))
			}
		}

		// For more detailed debugging, log headers and body
		if logConfig.Level == Debug {
			// Log headers
			headerFields := make(map[string]interface{})
			for k, v := range c.Request.Header {
				if len(v) == 1 {
					headerFields[k] = v[0]
				} else {
					headerFields[k] = v
				}
			}
			reqLogger.WithField("headers", headerFields).Debug("Request headers")

			// Log request body if present
			if len(requestBody) > 0 {
				// Filter sensitive data and format for readability
				filteredBody := FilterSensitiveData(string(requestBody))
				prettyBody := PrettifyJSON([]byte(filteredBody))
				reqLogger.WithField("body", prettyBody).Debug("Request body")
			}
		}

		// Create a custom response writer to capture the response
		responseWriter := NewLoggingResponseWriter(c.Writer)
		c.Writer = responseWriter

		// Process request
		c.Next()

		// Calculate duration
		latency := time.Since(start)

		// Log response details based on status code
		status := responseWriter.status
		logFn := reqLogger.Info
		if status >= 400 && status < 500 {
			logFn = reqLogger.Warn
		} else if status >= 500 {
			logFn = reqLogger.Error
		}

		// Include request latency and status code
		respLogger := reqLogger.WithFields(map[string]interface{}{
			"status":      status,
			"latency_ms":  latency.Milliseconds(),
			"user_agent":  c.Request.UserAgent(),
			"client_ip":   c.ClientIP(),
			"path":        c.Request.URL.Path,
			"referer":     c.Request.Referer(),
			"method":      c.Request.Method,
			"request_uri": c.Request.RequestURI,
		})

		logFn(fmt.Sprintf("Request completed: %s %s %d %v", c.Request.Method, c.Request.URL.Path, status, latency))

		// For debugging, log response body
		if logConfig.Level == Debug {
			responseBody := responseWriter.body.String()
			if responseBody != "" {
				// Filter sensitive data and truncate if needed
				filteredBody := FilterSensitiveData(responseBody)
				prettyBody := PrettifyJSON([]byte(filteredBody))
				respLogger.WithField("body", prettyBody).Debug("Response body")
			}
		}
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GetRequestIDFromContext retrieves the request ID from the context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey{}).(string); ok {
		return requestID
	}
	return ""
}
