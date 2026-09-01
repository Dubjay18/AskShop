package domain

import "context"

type ProductService interface {
	GetProducts(ctx context.Context) ([]*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	GetProductsByCategory(ctx context.Context, category string) ([]*Product, error)
	// Pagination
	GetProductsPage(ctx context.Context, page, pageSize int) ([]*Product, int64, error)
	GetProductsByCategoryPage(ctx context.Context, category string, page, pageSize int) ([]*Product, int64, error)
}
