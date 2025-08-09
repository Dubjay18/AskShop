package main

import (
	"askshop/services/user-service/internal/domain"
	"askshop/services/user-service/internal/infrastructure/http"
	"askshop/services/user-service/internal/infrastructure/repository"
	"askshop/services/user-service/internal/service"
	"askshop/shared/auth"
	"askshop/shared/db"
	"askshop/shared/env"
	"log"

	"github.com/gin-gonic/gin"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8084")
)

func main() {
	log.Println("Starting User Service")

	// Set Gin mode based on environment
	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	cfg := db.LoadConfigFromEnv()
	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Printf("DB connection failed: %v", err)
	}

	// Initialize repository
	var userRepo domain.UserRepository
	if err == nil {
		userRepo = repository.NewUserRepository(dbConn)
	} else {
		log.Println("Running without database connection")
	}

	// Initialize service (only if we have a repository)
	var userService service.UserServiceInterface
	var userHandler *http.UserHandler
	if userRepo != nil {
		userService = service.NewUserService(userRepo)
		userHandler = http.NewUserHandler(userService)
	}

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// JWT manager shared
	jwtManager := auth.NewManager(auth.LoadConfig())

	// Register routes
	if userHandler != nil {
		userHandler.RegisterRoutes(router)
	}

	// Additional specific routes for different lookup methods
	router.GET("/api/users/by-id/:id", userHandler.GetUserById)
	router.GET("/api/users/by-email", userHandler.GetUserByEmail)

	// Example protected group (future write operations)
	protected := router.Group("/api/users/admin")
	protected.Use(auth.AuthMiddleware(jwtManager))
	protected.GET("/me", func(c *gin.Context) {
		if claims, ok := auth.GetClaims(c); ok {
			c.JSON(200, gin.H{"claims": claims})
		} else {
			c.JSON(500, gin.H{"error": "claims missing"})
		}
	})

	// Add a root route
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to AskShop User Service",
			"version": "1.0.0",
			"endpoints": []string{
				"/api/users/:identifier - Get user by ID or email",
				"/api/users/by-id/:id - Get user by ID only",
				"/api/users/by-email?email=... - Get user by email only",
				"POST /api/users - Create user",
			},
		})
	})

	// Start the server
	log.Printf("User Service listening on %s", httpAddr)
	if err := router.Run(httpAddr); err != nil {
		log.Printf("HTTP server error %v", err)
	}
}
