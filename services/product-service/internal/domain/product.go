package domain

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProductInterface interface {
	GetProductByID(ctx context.Context)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	CreateProduct(ctx context.Context, product *Product) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) (*Product, error)
	DeleteProduct(ctx context.Context, productID uuid.UUID) error
}

// ---------- Core Models ----------

type Product struct {
	// Identity
	ID   uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"` // Postgres: enable pgcrypto or use uuid-ossp
	Name string    `json:"name" gorm:"type:varchar(160);not null;index"`
	Slug string    `json:"slug" gorm:"type:varchar(180);not null;uniqueIndex"`
	SKU  string    `json:"sku" gorm:"type:varchar(64);uniqueIndex"`

	// Merch data
	Description string                      `json:"description" gorm:"type:text"`
	Currency    string                      `json:"currency" gorm:"type:char(3);not null;default:USD"`
	Status      string                      `json:"status" gorm:"type:varchar(16);not null;default:draft;check:status IN ('draft','active','archived')"`
	Tags        datatypes.JSONSlice[string] `json:"tags" gorm:"type:jsonb;default:'[]'"` // e.g., ["new","sale"]

	// Pricing & inventory
	PriceCents    int64 `json:"priceCents" gorm:"not null;default:0"` // unit price in minor currency units (e.g. cents)
	StockQuantity int   `json:"stockQuantity" gorm:"not null;default:0"`

	// Associations
	Images     []ProductImage `json:"images" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Categories []Category     `json:"categories" gorm:"many2many:product_categories;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Housekeeping
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ProductImage keeps order for galleries.
type ProductImage struct {
	ID        uuid.UUID         `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProductID uuid.UUID         `json:"productId" gorm:"type:uuid;not null;index"`
	URL       string            `json:"url" gorm:"type:text;not null"`
	Alt       string            `json:"alt" gorm:"type:varchar(180);not null;default:''"`
	Position  int               `json:"position" gorm:"not null;default:0;index"`
	Metadata  datatypes.JSONMap `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Category is minimal; you can expand as needed.
type Category struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string         `json:"name" gorm:"type:varchar(120);not null;uniqueIndex"`
	Slug      string         `json:"slug" gorm:"type:varchar(140);not null;uniqueIndex"`
	ParentID  *uuid.UUID     `json:"parentId" gorm:"type:uuid;index"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Children []Category `json:"-" gorm:"foreignKey:ParentID"`
}

// ---------- Hooks & Scopes ----------

// BeforeCreate ensures UUID/Slug are set for Product, Brand, Category.
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Slug == "" {
		p.Slug = slugify(p.Name)
	}
	return nil
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Slug == "" {
		c.Slug = slugify(c.Name)
	}
	return nil
}

// Scopes for common filters.
func ScopeActive(db *gorm.DB) *gorm.DB {
	return db.Where("status = ?", "active")
}

func ScopeSearch(q string) func(*gorm.DB) *gorm.DB {
	q = strings.TrimSpace(q)
	if q == "" {
		return func(db *gorm.DB) *gorm.DB { return db }
	}
	like := "%" + strings.ToLower(q) + "%"
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ? OR LOWER(sku) LIKE ?", like, like, like)
	}
}

// ---------- Helpers ----------

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return uuid.NewString()
	}
	return s
}
