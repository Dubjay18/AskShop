package main

import (
	"askshop/services/order-service/internal/domain"
	"askshop/services/order-service/internal/infrastructure/cartclient"
	orderhttp "askshop/services/order-service/internal/infrastructure/http"
	"askshop/services/order-service/internal/infrastructure/repository"
	"askshop/services/order-service/internal/service"
	"askshop/shared/contracts"
	"askshop/shared/db"
	"askshop/shared/env"
	"askshop/shared/events"
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var httpAddr = env.GetString("HTTP_ADDR", ":8085")

func main() {
	log.Println("Starting Order Service")

	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := db.LoadConfigFromEnv()
	gdb := db.MustConnect(cfg, db.WithAutoMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(&domain.Order{}, &domain.OrderItem{})
	}))

	orderRepo := repository.NewPGOrderRepository(gdb)
	cartCli := cartclient.New()
	publisher := events.NewPublisher()
	defer publisher.Close()

	orderService := service.NewOrderService(orderRepo, cartCli, publisher)

	// Notification stub: consumes the events order-service just published, proving
	// the RabbitMQ round trip end to end. A real notification service would live
	// on its own; this keeps Phase 1's wiring self-contained and demonstrable.
	consumeCtx, cancelConsume := context.WithCancel(context.Background())
	defer cancelConsume()
	go events.Consume(consumeCtx, "order-service.notify-stub", contracts.OrderEventPlaced, func(body []byte) {
		log.Printf("notify-stub: received %s: %s", contracts.OrderEventPlaced, string(body))
	})

	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"service": "order", "status": "ok"}) })
	orderhttp.NewOrderHandler(orderService).RegisterRoutes(r)

	log.Printf("Order Service listening on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatal(err)
	}
}
