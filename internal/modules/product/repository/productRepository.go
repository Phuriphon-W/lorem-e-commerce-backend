package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, product *database.Product) (uuid.UUID, error)
	GetProducts(ctx context.Context, page uint64, pageSize uint64, category, search, order string) ([]database.Product, int64, error)
	GetProductByID(ctx context.Context, productID uuid.UUID) (*database.Product, error)
}
