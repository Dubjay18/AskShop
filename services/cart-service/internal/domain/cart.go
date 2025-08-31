package domain

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// CartInterface defines the contract for cart operations
type CartInterface interface {
	GetCartByUserID(ctx *gin.Context, userID string) (*Cart, error)
	AddItemToCart(ctx *gin.Context, userID string, item *CartItem) (*Cart, error)
	UpdateCartItem(ctx *gin.Context, userID string, itemID uuid.UUID, quantity int) (*Cart, error)
	RemoveItemFromCart(ctx *gin.Context, userID string, itemID uuid.UUID) error
	ClearCart(ctx *gin.Context, userID string) error
	GetCartTotal(ctx *gin.Context, userID string) (*CartSummary, error)
}

// ---------- Core Models ----------

// Cart represents a user's shopping cart
type Cart struct {
	// Identity
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID string    `json:"userId" gorm:"type:varchar(255);not null;uniqueIndex;index"`

	// Cart metadata
	Status   string                      `json:"status" gorm:"type:varchar(20);not null;default:active;check:status IN ('active','abandoned','converted','expired')"`
	Currency string                      `json:"currency" gorm:"type:char(3);not null;default:USD"`
	Notes    string                      `json:"notes" gorm:"type:text"`
	Metadata datatypes.JSONMap           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	Tags     datatypes.JSONSlice[string] `json:"tags" gorm:"type:jsonb;default:'[]'"` // e.g., ["wishlist","saved_for_later"]

	// Associations
	Items []CartItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Session and tracking
	SessionID string `json:"sessionId" gorm:"type:varchar(255);index"` // For anonymous carts
	IPAddress string `json:"ipAddress" gorm:"type:varchar(45)"`        // For analytics

	// Timestamps
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`

	// Cart expiry (useful for abandoned cart recovery)
	ExpiresAt *time.Time `json:"expiresAt" gorm:"index"`
}

// CartItem represents an individual item in the cart
type CartItem struct {
	// Identity
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CartID uuid.UUID `json:"cartId" gorm:"type:uuid;not null;index"`

	// Product reference
	ProductID   uuid.UUID `json:"productId" gorm:"type:uuid;not null;index"`
	ProductSKU  string    `json:"productSku" gorm:"type:varchar(64);not null"`
	ProductName string    `json:"productName" gorm:"type:varchar(255);not null"` // Snapshot for order history

	// Pricing (snapshot at time of adding to cart)
	UnitPrice    float64 `json:"unitPrice" gorm:"type:decimal(10,2);not null"`
	Currency     string  `json:"currency" gorm:"type:char(3);not null;default:USD"`
	DiscountRate float64 `json:"discountRate" gorm:"type:decimal(5,4);default:0"` // e.g., 0.1 for 10% discount

	// Quantity and variations
	Quantity int `json:"quantity" gorm:"not null;default:1;check:quantity > 0"`

	// Product variations/options (e.g., size, color)
	Variations datatypes.JSONMap `json:"variations" gorm:"type:jsonb;default:'{}'"` // {"size": "L", "color": "blue"}

	// Item metadata
	Notes    string            `json:"notes" gorm:"type:text"`
	Metadata datatypes.JSONMap `json:"metadata" gorm:"type:jsonb;default:'{}'"` // Custom data

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Calculated fields (not stored in DB)
	Subtotal      float64 `json:"subtotal" gorm:"-"`      // quantity * unitPrice
	DiscountTotal float64 `json:"discountTotal" gorm:"-"` // calculated discount amount
	FinalPrice    float64 `json:"finalPrice" gorm:"-"`    // subtotal - discountTotal
}

// CartSummary provides cart totals and summary information
type CartSummary struct {
	CartID        uuid.UUID `json:"cartId"`
	UserID        string    `json:"userId"`
	ItemCount     int       `json:"itemCount"`
	TotalQuantity int       `json:"totalQuantity"`
	Subtotal      float64   `json:"subtotal"`
	DiscountTotal float64   `json:"discountTotal"`
	TaxTotal      float64   `json:"taxTotal"`
	ShippingTotal float64   `json:"shippingTotal"`
	GrandTotal    float64   `json:"grandTotal"`
	Currency      string    `json:"currency"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SavedItem represents items saved for later (wishlist)
type SavedItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    string    `json:"userId" gorm:"type:varchar(255);not null;index"`
	ProductID uuid.UUID `json:"productId" gorm:"type:uuid;not null;index"`

	// Snapshot data
	ProductName string                      `json:"productName" gorm:"type:varchar(255);not null"`
	ProductSKU  string                      `json:"productSku" gorm:"type:varchar(64);not null"`
	UnitPrice   float64                     `json:"unitPrice" gorm:"type:decimal(10,2);not null"`
	Currency    string                      `json:"currency" gorm:"type:char(3);not null;default:USD"`
	Variations  datatypes.JSONMap           `json:"variations" gorm:"type:jsonb;default:'{}'"`
	Tags        datatypes.JSONSlice[string] `json:"tags" gorm:"type:jsonb;default:'[]'"` // e.g., ["wishlist","compare"]

	// Metadata
	Notes    string            `json:"notes" gorm:"type:text"`
	Metadata datatypes.JSONMap `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	Priority int               `json:"priority" gorm:"default:0"` // For ordering wishlist items

	// Timestamps
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`

	// Unique constraint to prevent duplicates
	_ struct{} `gorm:"uniqueIndex:idx_user_product,unique"`
}

// ---------- Hooks & Methods ----------

// BeforeCreate ensures UUID is set and sets default expiry
func (c *Cart) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.ExpiresAt == nil {
		// Set default expiry to 30 days from now
		expiry := time.Now().AddDate(0, 0, 30)
		c.ExpiresAt = &expiry
	}
	return nil
}

// BeforeCreate ensures UUID is set for cart items
func (ci *CartItem) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == uuid.Nil {
		ci.ID = uuid.New()
	}
	return nil
}

// BeforeCreate ensures UUID is set for saved items
func (si *SavedItem) BeforeCreate(tx *gorm.DB) error {
	if si.ID == uuid.Nil {
		si.ID = uuid.New()
	}
	return nil
}

// CalculateTotals calculates derived fields for cart item
func (ci *CartItem) CalculateTotals() {
	ci.Subtotal = ci.UnitPrice * float64(ci.Quantity)
	ci.DiscountTotal = ci.Subtotal * ci.DiscountRate
	ci.FinalPrice = ci.Subtotal - ci.DiscountTotal
}

// CalculateSummary calculates cart summary from items
func (c *Cart) CalculateSummary() CartSummary {
	summary := CartSummary{
		CartID:   c.ID,
		UserID:   c.UserID,
		Currency: c.Currency,
	}

	for _, item := range c.Items {
		item.CalculateTotals()
		summary.ItemCount++
		summary.TotalQuantity += item.Quantity
		summary.Subtotal += item.Subtotal
		summary.DiscountTotal += item.DiscountTotal
	}

	// Calculate tax (this would typically come from a tax service)
	summary.TaxTotal = summary.Subtotal * 0.08 // 8% tax example

	// Calculate shipping (this would typically come from a shipping service)
	if summary.Subtotal > 100 {
		summary.ShippingTotal = 0 // Free shipping over $100
	} else {
		summary.ShippingTotal = 9.99
	}

	summary.GrandTotal = summary.Subtotal - summary.DiscountTotal + summary.TaxTotal + summary.ShippingTotal
	summary.UpdatedAt = c.UpdatedAt

	return summary
}

// IsExpired checks if the cart has expired
func (c *Cart) IsExpired() bool {
	return c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt)
}

// IsEmpty checks if the cart has no items
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

// GetItemByProductID finds a cart item by product ID and variations
func (c *Cart) GetItemByProductID(productID uuid.UUID, variations datatypes.JSONMap) *CartItem {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			// Simple comparison - you might want more sophisticated variation matching
			if len(variations) == 0 && len(c.Items[i].Variations) == 0 {
				return &c.Items[i]
			}
			// TODO: Implement proper variation matching logic
		}
	}
	return nil
}

// ---------- Scopes ----------

// ScopeActive filters for active carts
func ScopeActive(db *gorm.DB) *gorm.DB {
	return db.Where("status = ?", "active")
}

// ScopeByUser filters carts by user ID
func ScopeByUser(userID string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}

// ScopeNotExpired filters out expired carts
func ScopeNotExpired(db *gorm.DB) *gorm.DB {
	return db.Where("expires_at IS NULL OR expires_at > ?", time.Now())
}

// ScopeBySession filters carts by session ID (for anonymous carts)
func ScopeBySession(sessionID string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("session_id = ?", sessionID)
	}
}
