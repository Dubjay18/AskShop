package main

import (
    "askshop/services/product-service/internal/domain"
    grpc_server "askshop/services/product-service/internal/infrastructure/grpc"
    "askshop/services/product-service/internal/infrastructure/repository"
    "askshop/services/product-service/internal/service"
    "askshop/shared/db"
    "askshop/shared/env"
    "log"
    "strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8082")
)

func main() {
	log.Println("Starting Product Service")

	// Set Gin mode based on environment
	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize repository: connect DB and prepare migrations
	cfg := db.LoadConfigFromEnv()
	_, err := db.Connect(cfg, db.WithAutoMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(
			&domain.Product{},
			&domain.ProductImage{},
			&domain.Category{},
		)
	}))
	if err != nil {
		log.Printf("DB connection failed: %v", err)
	}

    // Set up repository (in-memory until DB repo is implemented)
    var productRepo repository.ProductRepository = repository.NewInMemoryProductRepository()
    if err != nil {
        log.Println("Running with in-memory repository (DB unavailable)")
    } else {
        log.Println("DB connected; using in-memory repository until PG repo is added")
    }

	// Initialize service
	productService := service.NewProductService(productRepo)

	// TODO Initialize gRPC server
    grpcAddr := env.GetString("PRODUCT_SERVICE_GRPC_ADDR", ":9090")
    // If env contains a host like "product-service:9090", bind to local port only
    if strings.HasPrefix(grpcAddr, ":") == false {
        if idx := strings.LastIndex(grpcAddr, ":"); idx != -1 && idx+1 < len(grpcAddr) {
            grpcAddr = ":" + grpcAddr[idx+1:]
        }
    }
    grpcSrv := grpc_server.NewGRPCServer(grpcAddr, productService)
    go grpcSrv.Start()
	// TODO Initialize HTTP handler

	// TODO Create Gin router

	//TODO  Add CORS middleware

	// TODO Register routes

	// TODO Add a root route

    // Start a minimal HTTP server to keep the process alive
    r := gin.Default()
    r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"service": "product", "status": "ok"}) })
    if err := r.Run(httpAddr); err != nil {
        log.Fatal(err)
    }
}
