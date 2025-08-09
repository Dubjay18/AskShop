package response

import "github.com/gin-gonic/gin"

// ErrorDetail represents the error object returned in failed responses
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Standard represents the unified JSON response structure
// success responses: { success:true, data:..., meta:..., message:"" }
// error responses:   { success:false, error:{code,message,details}, trace_id:"" }
type Standard struct {
	Success bool         `json:"success"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    interface{}  `json:"meta,omitempty"`
	Message string       `json:"message,omitempty"`
	TraceID string       `json:"trace_id,omitempty"`
}

// Success sends a success JSON response
func Success(c *gin.Context, status int, data interface{}, meta interface{}, message string) {
	traceID, _ := c.Get("request_id")
	c.JSON(status, Standard{
		Success: true,
		Data:    data,
		Meta:    meta,
		Message: message,
		TraceID: toString(traceID),
	})
}

// Error sends an error JSON response
func Error(c *gin.Context, status int, code, message string, details interface{}) {
	traceID, _ := c.Get("request_id")
	c.JSON(status, Standard{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		TraceID: toString(traceID),
	})
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
