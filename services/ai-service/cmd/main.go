package main

import (
	"askshop/services/ai-service/internal/cartclient"
	aihttp "askshop/services/ai-service/internal/http"
	"askshop/services/ai-service/internal/llm"
	"askshop/services/ai-service/internal/productclient"
	"askshop/services/ai-service/internal/service"
	"askshop/shared/env"
	"askshop/shared/events"
	"askshop/shared/health"
	"log"

	"github.com/gin-gonic/gin"
)

var httpAddr = env.GetString("HTTP_ADDR", ":8086")

func main() {
	log.Println("Starting AI Service")

	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	llmClient := llm.New()
	if !llmClient.Available {
		log.Println("GEMINI_API_KEY not set — AI endpoints will return 503 until it's configured")
	} else {
		log.Printf("AI Service using model %s", llmClient.Model)
	}

	aiService := service.NewAIService(llmClient, productclient.New(), cartclient.New())
	publisher := events.NewPublisher()
	defer publisher.Close()

	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"service": "ai", "status": "ok"}) })
	health.Register(r, "ai")
	aihttp.NewAIHandler(aiService, publisher).RegisterRoutes(r)

	log.Printf("AI Service listening on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatal(err)
	}
}
