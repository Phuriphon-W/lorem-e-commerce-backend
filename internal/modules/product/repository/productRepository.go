package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, product *database.Product) (uuid.UUID, error)
	GetProducts(ctx context.Context, page int64, pageSize int64, category, search, order string) ([]database.Product, int64, error)
	GetProductByID(ctx context.Context, productID uuid.UUID) (*database.Product, error)
	GetProductsByIDs(ctx context.Context, productIDs []uuid.UUID) ([]database.Product, error)
	GetProductStock(ctx context.Context, productId uuid.UUID) (uint, error)
	UpdateProductByID(ctx context.Context, productID uuid.UUID, updateData map[string]interface{}) error
	DeductProductStocks(ctx context.Context, deductions []StockDeduction) error
	AddProductStocks(ctx context.Context, additions []StockDeduction) error
}
