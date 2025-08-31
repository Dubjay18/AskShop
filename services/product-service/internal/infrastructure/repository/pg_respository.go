package repository

import (
    "askshop/services/product-service/internal/domain"
    "context"
)

//ProductRepository defines the data access interface for product operations

type ProductRepository interface {
    GetProductByID(ctx context.Context, productID string) (*domain.Product, error)
    GetProductByCategory(ctx context.Context, category string) ([]*domain.Product, error)
    GetAllProducts(ctx context.Context) ([]*domain.Product, error)
    // Paginated variants return items and total count
    GetAllProductsPage(ctx context.Context, limit, offset int) ([]*domain.Product, int64, error)
    GetProductByCategoryPage(ctx context.Context, category string, limit, offset int) ([]*domain.Product, int64, error)
}
