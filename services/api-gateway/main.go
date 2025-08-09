package main

import (
	"askshop/shared/env"
	"log"

	"github.com/gin-gonic/gin"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")

	// Set Gin mode based on environment
	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a default gin router with logger and recovery middleware
	router := gin.Default()

	// Setup routes
	SetupRoutes(router)

	// Add a root route
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to AskShop API Gateway",
			"version": "1.0.0",
			"endpoints": []string{
				"/api/v1/auth/register - User registration",
				"/api/v1/auth/login - User login",
				"/api/v1/auth/profile - Get user profile (protected)",
				"/api/v1/products - List all products",
				"/api/v1/products/:id - Get product by ID",
				"/api/v1/users - List all users",
				"/api/v1/users/:id - Get user by ID",
				"/health - Health check",
				"/docs - API documentation",
			},
		})
	})

	// Start the server
	log.Printf("API Gateway listening on %s", httpAddr)
	log.Printf("REST API available at http://localhost%s/api/v1", httpAddr)
	if err := router.Run(httpAddr); err != nil {
		log.Printf("HTTP server error %v", err)
	}
}
