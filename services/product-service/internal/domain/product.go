package domain

import "context"

type ProductInterface interface {
	GetProductByID(ctx context.Context)
}
