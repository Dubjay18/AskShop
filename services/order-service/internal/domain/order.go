package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCartEmpty     = errors.New("cart is empty")
	ErrCartInvalid   = errors.New("cart failed checkout validation")
	ErrOrderNotFound = errors.New("order not found")
)

// Order statuses. Kept as plain strings (validated via a DB check constraint)
// to match the convention used by Cart/Product in this codebase.
const (
	StatusPlaced    = "placed"
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
)

// Order represents a placed order, snapshotting cart contents at checkout time.
type Order struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID string    `json:"userId" gorm:"type:varchar(255);not null;index"`
	Status string    `json:"status" gorm:"type:varchar(16);not null;default:placed;check:status IN ('placed','confirmed','cancelled')"`

	Currency      string `json:"currency" gorm:"type:char(3);not null;default:USD"`
	SubtotalCents int64  `json:"subtotalCents" gorm:"not null;default:0"`
	TotalCents    int64  `json:"totalCents" gorm:"not null;default:0"`

	Items []OrderItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// OrderItem is a line item snapshot taken from the cart at checkout time.
type OrderItem struct {
	ID      uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrderID uuid.UUID `json:"orderId" gorm:"type:uuid;not null;index"`

	ProductID   uuid.UUID `json:"productId" gorm:"type:uuid;not null;index"`
	ProductSKU  string    `json:"productSku" gorm:"type:varchar(64);not null"`
	ProductName string    `json:"productName" gorm:"type:varchar(255);not null"`

	UnitPriceCents int64 `json:"unitPriceCents" gorm:"not null"`
	Quantity       int   `json:"quantity" gorm:"not null;check:quantity > 0"`

	CreatedAt time.Time `json:"createdAt"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (i *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// OrderRepository defines the persistence contract for orders.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) (*Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]*Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error
}

// OrderService defines the business logic for placing and reading orders.
type OrderService interface {
	PlaceOrder(ctx context.Context, userID string) (*Order, error)
	GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrdersForUser(ctx context.Context, userID string) ([]*Order, error)
}
