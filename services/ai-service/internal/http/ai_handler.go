package http

import (
	"askshop/services/ai-service/internal/service"
	"askshop/shared/contracts"
	"askshop/shared/events"
	"askshop/shared/response"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	service   *service.AIService
	publisher *events.Publisher
}

func NewAIHandler(svc *service.AIService, publisher *events.Publisher) *AIHandler {
	return &AIHandler{service: svc, publisher: publisher}
}

func (h *AIHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/ai")
	{
		api.POST("/chat", h.Chat)
		api.POST("/products/:id/explain", h.ExplainProduct)
		api.POST("/cart-nudges", h.GenerateCartNudges)
	}
}

func mapError(err error) (int, string, string) {
	if errors.Is(err, service.ErrLLMUnavailable) {
		return http.StatusServiceUnavailable, contracts.CodeInternalServerError, err.Error()
	}
	return http.StatusInternalServerError, contracts.CodeInternalServerError, err.Error()
}

type chatRequest struct {
	Message string                `json:"message" binding:"required"`
	History []service.ChatMessage `json:"history"`
}

// Chat handles POST /api/v1/ai/chat: the conversational shopping assistant.
func (h *AIHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Chat(c.Request.Context(), req.History, req.Message)
	if err != nil {
		status, code, msg := mapError(err)
		response.Error(c, status, code, msg, nil)
		return
	}
	response.Success(c, http.StatusOK, result, nil, "")
}

// ExplainProduct handles POST /api/v1/ai/products/:id/explain.
func (h *AIHandler) ExplainProduct(c *gin.Context) {
	productID := c.Param("id")

	explanation, err := h.service.ExplainProduct(c.Request.Context(), productID)
	if err != nil {
		status, code, msg := mapError(err)
		response.Error(c, status, code, msg, nil)
		return
	}

	if pubErr := h.publisher.Publish(c.Request.Context(), contracts.AICmdExplainProduct, gin.H{"productId": productID}); pubErr != nil {
		// Non-fatal: the explanation already succeeded.
		_ = pubErr
	}

	response.Success(c, http.StatusOK, gin.H{"explanation": explanation}, nil, "")
}

// GenerateCartNudges handles POST /api/v1/ai/cart-nudges: a batch job
// (intended for cron/manual trigger) that generates re-engagement messages
// for abandoned carts.
func (h *AIHandler) GenerateCartNudges(c *gin.Context) {
	minutes, err := strconv.Atoi(c.DefaultQuery("sinceMinutes", "60"))
	if err != nil || minutes <= 0 {
		minutes = 60
	}

	nudges, err := h.service.GenerateAbandonedCartNudges(c.Request.Context(), minutes)
	if err != nil {
		status, code, msg := mapError(err)
		response.Error(c, status, code, msg, nil)
		return
	}
	response.Success(c, http.StatusOK, nudges, nil, "")
}
