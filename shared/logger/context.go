package logger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	// ContextKeyRequestID is the key used to store request ID in context
	ContextKeyRequestID contextKey = "request_id"
	// ContextKeyUserID is the key used to store user ID in context
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyTraceID is the key used to store trace ID in context
	ContextKeyTraceID contextKey = "trace_id"
)

// FromContext creates a logger from the context, adding any request-specific fields
func FromContext(ctx context.Context) *Logger {
	logger := Default("")

	// Extract request ID from context if available
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok && requestID != "" {
		logger = logger.WithField("request_id", requestID)
	}

	// Extract user ID from context if available
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok && userID != "" {
		logger = logger.WithField("user_id", userID)
	}

	// Extract trace ID from context if available
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok && traceID != "" {
		logger = logger.WithField("trace_id", traceID)
	}

	return logger
}

// FromGin creates a logger from a gin context, adding any request-specific fields
func FromGin(c *gin.Context) *Logger {
	logger := Default("")

	// Extract request ID from gin context
	if requestID, exists := c.Get("request_id"); exists {
		if rid, ok := requestID.(string); ok {
			logger = logger.WithField("request_id", rid)
		}
	}

	// Extract user ID if available from gin context
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			logger = logger.WithField("user_id", uid)
		}
	}

	// Add request method and path
	logger = logger.WithFields(map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	})

	// Add client IP if available
	clientIP := c.ClientIP()
	if clientIP != "" {
		logger = logger.WithField("client_ip", clientIP)
	}

	return logger
}

// Middleware creates a gin middleware that injects a request ID and logger into the context
func Middleware(serviceName string) gin.HandlerFunc {
	baseLogger := Default(serviceName)

	return func(c *gin.Context) {
		// Add request ID to context if not already present
		reqID, exists := c.Get("request_id")
		if !exists || reqID == "" {
			reqID = generateRequestID()
			c.Set("request_id", reqID)
		}

		// Create a request-scoped logger
		reqLogger := baseLogger.WithField("request_id", reqID)

		// Add more request info
		reqLogger = reqLogger.WithFields(map[string]interface{}{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		})

		// Extract user info if present in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// We don't decode the token here, but in a real app you might
			// extract the user ID and add it to the logger
			reqLogger = reqLogger.WithField("auth", "yes")
		}

		// Add logger to request context
		c.Set("logger", reqLogger)

		// Log request start
		reqLogger.Infof("Request started: %s %s", c.Request.Method, c.Request.URL.Path)

		// Process request
		c.Next()

		// Log request completion
		statusCode := c.Writer.Status()
		level := Info
		if statusCode >= 400 {
			level = Warn
		}
		if statusCode >= 500 {
			level = Error
		}

		reqLogger.Logf(level, "Request completed: %s %s [%d]",
			c.Request.Method, c.Request.URL.Path, statusCode)
	}
}

// GetLogger gets the logger from a gin context
func GetLogger(c *gin.Context) *Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*Logger); ok {
			return l
		}
	}
	return Default("gin")
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
