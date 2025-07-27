package main

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WriteJSON is a utility function to maintain backward compatibility
// with code that might still be using the old HTTP standard library
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// RespondJSON is a helper function for returning JSON responses with Gin
func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// RespondError is a helper function for returning error responses with Gin
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}
