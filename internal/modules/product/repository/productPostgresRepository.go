package repository

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type productPostgresRepository struct {
	db database.Database
}

type StockDeduction struct {
	ProductID uuid.UUID
	Quantity  uint
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

func (r *productPostgresRepository) GetProductStock(ctx context.Context, productId uuid.UUID) (uint, error) {
	var product database.Product
	err := r.db.GetDb().WithContext(ctx).
		Select("available").
		Where("id = ?", productId).
		First(&product).Error

	if err != nil {
		return 0, err
	}

	return product.Available, nil
}

func (r *productPostgresRepository) UpdateProductByID(ctx context.Context, productID uuid.UUID, updateData map[string]interface{}) error {
	return r.db.GetDb().WithContext(ctx).
		Model(&database.Product{}).
		Where("id = ?", productID).
		Updates(updateData).Error
}

func (r *productPostgresRepository) DeductProductStocks(ctx context.Context, deductions []StockDeduction) error {
	return r.db.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range deductions {
			var product database.Product
			// Lock the row and check stock
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", item.ProductID).First(&product).Error; err != nil {
				return err
			}

			if product.Available < item.Quantity {
				return fmt.Errorf("insufficient stock for product %s: requested %d, available %d", item.ProductID, item.Quantity, product.Available)
			}

			// Perform the safe mathematical deduction
			err := tx.Model(&database.Product{}).
				Where("id = ?", item.ProductID).
				Update("available", gorm.Expr("available - ?", item.Quantity)).Error

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *productPostgresRepository) AddProductStocks(ctx context.Context, additions []StockDeduction) error {
	return r.db.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range additions {
			// Increment the stock safely
			err := tx.Model(&database.Product{}).
				Where("id = ?", item.ProductID).
				Update("available", gorm.Expr("available + ?", item.Quantity)).Error

			if err != nil {
				return err
			}
		}

		return nil
	})
}
