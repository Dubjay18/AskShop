package service

import (
	"askshop/services/product-service/internal/domain"
	"askshop/services/product-service/internal/infrastructure/repository"
	"context"
)

type ProductServiceImpl struct {
	repo    repository.ProductRepository
	manager *domain.ProductManager
}

// NewProductService wires repository with business manager and returns the domain service
func NewProductService(repo repository.ProductRepository) domain.ProductService {
	return &ProductServiceImpl{
		repo:    repo,
		manager: domain.NewProductManager(),
	}
}

func (s *ProductServiceImpl) GetProducts(ctx context.Context) ([]*domain.Product, error) {
	return s.repo.GetAllProducts(ctx)
}

func (s *ProductServiceImpl) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return s.repo.GetProductByID(ctx, id)
}

func (s *ProductServiceImpl) GetProductsByCategory(ctx context.Context, category string) ([]*domain.Product, error) {
	return s.repo.GetProductByCategory(ctx, category)
}

// Pagination helpers and methods
func clampPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (s *ProductServiceImpl) GetProductsPage(ctx context.Context, page, pageSize int) ([]*domain.Product, int64, error) {
	page, pageSize = clampPagination(page, pageSize)
	offset := (page - 1) * pageSize
	return s.repo.GetAllProductsPage(ctx, pageSize, offset)
}

func (s *ProductServiceImpl) GetProductsByCategoryPage(ctx context.Context, category string, page, pageSize int) ([]*domain.Product, int64, error) {
	page, pageSize = clampPagination(page, pageSize)
	offset := (page - 1) * pageSize
	return s.repo.GetProductByCategoryPage(ctx, category, pageSize, offset)
}
