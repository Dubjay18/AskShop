package main

import (
	"askshop/services/product-service/internal/domain"
	grpc_server "askshop/services/product-service/internal/infrastructure/grpc"
	producthttp "askshop/services/product-service/internal/infrastructure/http"
	"askshop/services/product-service/internal/infrastructure/repository"
	"askshop/services/product-service/internal/service"
	"askshop/shared/db"
	"askshop/shared/env"
	"askshop/shared/health"
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
	gdb, err := db.Connect(cfg, db.WithAutoMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(
			&domain.Product{},
			&domain.ProductImage{},
			&domain.Category{},
		)
	}))

	var productRepo repository.ProductRepository
	if err != nil {
		log.Printf("DB connection failed, falling back to in-memory repository: %v", err)
		productRepo = repository.NewInMemoryProductRepository()
	} else {
		log.Println("DB connected; using Postgres repository")
		productRepo = repository.NewPGProductRepository(gdb)
	}

	// Initialize service
	productService := service.NewProductService(productRepo)

	// gRPC server (for service-to-service calls, e.g. cart/order pricing lookups)
	grpcAddr := env.GetString("PRODUCT_SERVICE_GRPC_ADDR", ":9090")
	// If env contains a host like "product-service:9090", bind to local port only
	if !strings.HasPrefix(grpcAddr, ":") {
		if idx := strings.LastIndex(grpcAddr, ":"); idx != -1 && idx+1 < len(grpcAddr) {
			grpcAddr = ":" + grpcAddr[idx+1:]
		}
	}
	grpcSrv := grpc_server.NewGRPCServer(grpcAddr, productService)
	go grpcSrv.Start()

	// HTTP server
	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"service": "product", "status": "ok"}) })
	health.Register(r, "product")
	producthttp.NewProductHandler(productService).RegisterRoutes(r)

	if err := r.Run(httpAddr); err != nil {
		log.Fatal(err)
	}
}
