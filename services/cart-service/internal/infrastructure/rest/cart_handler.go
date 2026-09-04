package rest

import (
	"askshop/services/cart-service/internal/domain"
	productclient "askshop/services/cart-service/internal/infrastructure/grpc"
	"askshop/shared/contracts"
	"askshop/shared/response"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func mapDomainError(err error) (int, string, string, interface{}) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, contracts.CodeInternalServerError, "Cart not found", nil
	case errors.Is(err, domain.ErrCartExpired):
		return http.StatusGone, contracts.CodeInternalServerError, "Cart has expired", nil
	case errors.Is(err, domain.ErrItemNotFound):
		return http.StatusNotFound, contracts.CodeInternalServerError, "Item not found", nil
	case errors.Is(err, productclient.ErrProductUnavailable):
		return http.StatusConflict, contracts.CodeInternalServerError, err.Error(), nil
	case errors.Is(err, domain.ErrInsufficientStock):
		return http.StatusConflict, contracts.CodeInternalServerError, err.Error(), nil
	default:
		return http.StatusInternalServerError, contracts.CodeInternalServerError, "Internal server error", err.Error()
	}
}

type CartHandler struct {
	cartService domain.CartService
}

func NewCartHandler(svc domain.CartService) *CartHandler {
	return &CartHandler{cartService: svc}
}

// RegisterRoutes wires cart endpoints under /api/v1/cart. userID is read from the
// "userId" context key, set by an upstream auth middleware.
func (h *CartHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	cart := api.Group("/cart")
	{
		cart.GET("", h.GetCart)
		cart.GET("/summary", h.GetCartSummary)
		cart.POST("/items", h.AddItem)
		cart.PUT("/items/:itemId", h.UpdateItemQuantity)
		cart.DELETE("/items/:itemId", h.RemoveItem)
		cart.DELETE("", h.ClearCart)
		cart.POST("/checkout/validate", h.ValidateCheckout)
		cart.POST("/checkout/complete", h.CompleteCheckout)
		// Internal/admin endpoint (no auth gate yet): used by ai-service to
		// generate cart-abandonment nudges. Not exposed through the gateway.
		cart.GET("/admin/abandoned", h.ListAbandonedCarts)
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

// requireUserID rejects the request with 400 if no user identity is present,
// writing the error response itself. Returns ("", false) in that case so the
// caller can just `if uid, ok := requireUserID(c); !ok { return }`.
// Without this, every caller who omits X-User-ID collides on the same
// empty-string cart.
func requireUserID(c *gin.Context) (string, bool) {
	uid := userID(c)
	if uid == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "X-User-ID header is required", nil)
		return "", false
	}
	return uid, true
}

func (h *CartHandler) GetCart(c *gin.Context) {
	uid := userID(c)
	cart, err := h.cartService.GetOrCreateCart(c, uid, c.Query("sessionId"))
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, cart, nil, "")
}

func (h *CartHandler) GetCartSummary(c *gin.Context) {
	uid := userID(c)
	summary, err := h.cartService.GetCartSummary(c, uid)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, summary, nil, "")
}

func (h *CartHandler) AddItem(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	var req domain.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	cart, err := h.cartService.AddItem(c, uid, req.ProductID, req.Quantity, req.Variations)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, cart, nil, "Item added to cart")
}

func (h *CartHandler) UpdateItemQuantity(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid item id", nil)
		return
	}

	var req domain.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	cart, err := h.cartService.UpdateItemQuantity(c, uid, itemID, req.Quantity)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, cart, nil, "Cart item updated")
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid item id", nil)
		return
	}

	if err := h.cartService.RemoveItem(c, uid, itemID); err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, nil, nil, "Item removed from cart")
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.cartService.ClearCart(c, uid); err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, nil, nil, "Cart cleared")
}

// CompleteCheckout marks the caller's cart as converted. It's called by
// order-service once an order has been created from the cart's contents.
func (h *CartHandler) CompleteCheckout(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.cartService.ConvertCart(c, uid); err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, nil, nil, "Cart marked as converted")
}

// ListAbandonedCarts returns active carts untouched for at least sinceMinutes
// (default 60). Used by ai-service to generate re-engagement nudges.
func (h *CartHandler) ListAbandonedCarts(c *gin.Context) {
	minutes, err := strconv.Atoi(c.DefaultQuery("sinceMinutes", "60"))
	if err != nil || minutes <= 0 {
		minutes = 60
	}
	since := time.Now().Add(-time.Duration(minutes) * time.Minute)

	carts, err := h.cartService.GetAbandonedCarts(c, since)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, carts, nil, "")
}

func (h *CartHandler) ValidateCheckout(c *gin.Context) {
	uid, ok := requireUserID(c)
	if !ok {
		return
	}
	result, err := h.cartService.ValidateCartForCheckout(c, uid)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, result, nil, "")
}
