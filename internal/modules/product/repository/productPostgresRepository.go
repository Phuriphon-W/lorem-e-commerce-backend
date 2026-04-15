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

func (r *productPostgresRepository) GetProducts(
	ctx context.Context,
	page int64,
	pageSize int64,
	category string,
	search string,
	order string,
) ([]database.Product, int64, error) {
	var products []database.Product
	var total int64
	query := r.db.GetDb().WithContext(ctx).Model(&database.Product{})

	// Filter by Category
	if category != "" {
		query = query.Joins("JOIN categories ON products.category_id = categories.id").
			Where("categories.name ILIKE ?", category)
	}

	// Filter by Search
	if search != "" {
		query = query.Where("products.name ILIKE ?", "%"+search+"%")
	}

	// Count Total Records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply Ordering
	if order != "" {
		query.Order(order)
	} else {
		// Default order in ascending created date
		query.Order("products.created_at DESC")
	}

	// Calculate Page Offset
	offset := (page - 1) * pageSize

	// Apply Pagination and fetch data
	err := query.WithContext(ctx).
		Limit(int(pageSize)).
		Offset(int(offset)).
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

func (r *productPostgresRepository) GetProductsByIDs(ctx context.Context, productIDs []uuid.UUID) ([]database.Product, error) {
	var products []database.Product

	err := r.db.GetDb().WithContext(ctx).
		Where("id IN ?", productIDs).
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	return products, nil
}
