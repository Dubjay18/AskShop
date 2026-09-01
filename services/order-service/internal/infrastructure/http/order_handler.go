package http

import (
	"askshop/services/order-service/internal/domain"
	"askshop/shared/contracts"
	"askshop/shared/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderHandler struct {
	service domain.OrderService
}

func NewOrderHandler(service domain.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	orders := api.Group("/orders")
	{
		orders.POST("", h.PlaceOrder)
		orders.GET("", h.ListOrders)
		orders.GET("/:id", h.GetOrder)
	}
}

func userID(c *gin.Context) string {
	if v, ok := c.Get("userId"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return c.GetHeader("X-User-ID")
}

func mapDomainError(err error) (int, string, string, interface{}) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, domain.ErrOrderNotFound):
		return http.StatusNotFound, contracts.CodeInternalServerError, "Order not found", nil
	case errors.Is(err, domain.ErrCartEmpty), errors.Is(err, domain.ErrCartInvalid):
		return http.StatusUnprocessableEntity, contracts.CodeInternalServerError, err.Error(), nil
	default:
		return http.StatusInternalServerError, contracts.CodeInternalServerError, "Internal server error", err.Error()
	}
}

// PlaceOrder handles POST /api/v1/orders: creates an order from the caller's cart.
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	uid := userID(c)
	if uid == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "X-User-ID header is required", nil)
		return
	}

	order, err := h.service.PlaceOrder(c.Request.Context(), uid)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusCreated, order, nil, "Order placed successfully")
}

// GetOrder handles GET /api/v1/orders/:id.
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid order id", nil)
		return
	}

	order, err := h.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, order, nil, "")
}

// ListOrders handles GET /api/v1/orders for the calling user.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	uid := userID(c)
	if uid == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "X-User-ID header is required", nil)
		return
	}

	orders, err := h.service.GetOrdersForUser(c.Request.Context(), uid)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, orders, nil, "")
}
