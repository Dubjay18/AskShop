package domain

import (
	"context"

	"github.com/google/uuid"
)

type ProductService interface {
	GetProducts(ctx context.Context) ([]*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	GetProductsByCategory(ctx context.Context, category string) ([]*Product, error)
	// Pagination
	GetProductsPage(ctx context.Context, page, pageSize int) ([]*Product, int64, error)
	GetProductsByCategoryPage(ctx context.Context, category string, page, pageSize int) ([]*Product, int64, error)
	SearchProducts(ctx context.Context, query string, page, pageSize int) ([]*Product, int64, error)
	// Write operations
	CreateProduct(ctx context.Context, product *Product) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) (*Product, error)
	DeleteProduct(ctx context.Context, productID uuid.UUID) error
}
