package rest

import (
	"askshop/services/cart-service/internal/domain"
	"askshop/shared/contracts"
	"askshop/shared/response"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CartResponse struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	Status   string `json:"status" gorm:"type:varchar(20);not null;default:active;check:status IN ('active','abandoned','converted','expired')"`
	Currency string `json:"currency" gorm:"type:char(3);not null;default:USD"`
	Notes    string `json:"notes" gorm:"type:text"`

	// Associations
	Items []domain.CartItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Session and tracking
	SessionID string `json:"sessionId" gorm:"type:varchar(255);index"` // For anonymous carts
	IPAddress string `json:"ipAddress" gorm:"type:varchar(45)"`        // For analytics

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Cart expiry (useful for abandoned cart recovery)
	ExpiresAt *time.Time `json:"expiresAt" gorm:"index"`
}

func mapDomainError(err error) (int, string, string, interface{}) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, contracts.CodeUserNotFound, gorm.ErrRecordNotFound.Error(), nil
	default:
		return http.StatusInternalServerError, contracts.CodeInternalServerError, "Internal server error", err.Error()
	}
}

type CartHandler struct {
	cartService domain.CartInterface
}

func NewCartHandler(svc domain.CartInterface) *CartHandler {
	return &CartHandler{
		cartService: svc,
	}
}

func (h *CartHandler) GetCart(ctx *gin.Context) {
	userID := ctx.GetString("userId")
	cartResp, err := h.cartService.GetCartByUserID(ctx, userID)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(ctx, status, code, msg, details)
		return
	}
	response.Success(ctx, http.StatusOK, cartResp, nil, "")
}

// func (h *CartHandler) AddItemToCart(ctx *gin.Context) {

// }
