package main

import (
	"askshop/services/cart-service/internal/domain"
	productclient "askshop/services/cart-service/internal/infrastructure/grpc"
	"askshop/services/cart-service/internal/infrastructure/repository"
	"askshop/services/cart-service/internal/infrastructure/rest"
	"askshop/services/cart-service/internal/service"
	"askshop/shared/db"
	"askshop/shared/env"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var httpAddr = env.GetString("HTTP_ADDR", ":8083")

func main() {
	log.Println("Starting Cart Service")

	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := db.LoadConfigFromEnv()
	gdb := db.MustConnect(cfg, db.WithAutoMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(
			&domain.Cart{},
			&domain.CartItem{},
			&domain.SavedItem{},
		)
	}))

	cartRepo := repository.NewPGCartRepository(gdb)
	productAddr := env.GetString("PRODUCT_SERVICE_GRPC_ADDR", "product-service:9090")
	productClient := productclient.NewGRPCClient(productAddr)

	cartService := service.NewCartService(cartRepo, domain.DefaultCartConfig(), productClient)

	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"service": "cart", "status": "ok"}) })
	rest.NewCartHandler(cartService).RegisterRoutes(r)

	log.Printf("Cart Service listening on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatal(err)
	}
}
