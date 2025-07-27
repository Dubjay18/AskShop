package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the API Gateway
func SetupRoutes(router *gin.Engine) {
	// Add middleware
	router.Use(CORSMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())

	// Basic health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API v1 group
	//	v1 := router.Group("/api/v1")

	// Documentation endpoint
	router.GET("/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API documentation will be available here",
		})
	})

}
