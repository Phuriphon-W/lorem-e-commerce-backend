package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type productPostgresRepository struct {
	db database.Database
}

func NewProductPostgresRepository(db database.Database) ProductRepository {
	return &productPostgresRepository{
		db: db,
	}
}

func (r *productPostgresRepository) CreateProduct(ctx context.Context, product *database.Product) (uuid.UUID, error) {
	result := gorm.WithResult()

	err := gorm.G[database.Product](r.db.GetDb(), result).Create(ctx, product)
	if err != nil {
		return uuid.Nil, err
	}

	return product.ID, nil
}

func (r *productPostgresRepository) GetProducts(ctx context.Context, page uint64, pageSize uint64) ([]database.Product, int64, error) {
	var products []database.Product
	var total int64
	db := r.db.GetDb()

	// Count total records
	if err := db.Model(&database.Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate Offset
	offset := (page - 1) * pageSize

	// Fetch paginated data
	err := db.WithContext(ctx).
		Limit(int(pageSize)).
		Offset(int(offset)).
		Order("created_at DESC"). // Ordered by DESC by default
		Preload("Category").
		Find(&products).Error

	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *productPostgresRepository) GetProductByID(ctx context.Context, productID uuid.UUID) (*database.Product, error) {
	product, err := gorm.G[database.Product](r.db.GetDb()).
		Where("id = ?", productID).
		Preload("Category", nil).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &product, nil
}
