package repository

import (
	"context"
	"fmt"
	"log"
	"lorem-backend/internal/database"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	var existing database.Product
	err := r.db.GetDb().WithContext(ctx).Unscoped().Where("name = ?", product.Name).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			err = r.db.GetDb().WithContext(ctx).Unscoped().
				Model(&database.Product{}).
				Where("id = ?", existing.ID).
				Updates(map[string]interface{}{
					"deleted_at":  nil,
					"name":        product.Name,
					"description": product.Description,
					"price":       product.Price,
					"available":   product.Available,
					"obj_key":     product.ImageObjKey,
					"category_id": product.CategoryID,
				}).Error
			if err != nil {
				return uuid.Nil, err
			}
			product.ID = existing.ID
			return existing.ID, nil
		}
	}

	result := gorm.WithResult()

	err = gorm.G[database.Product](r.db.GetDb(), result).Create(ctx, product)
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
		query = query.Order(order)
	} else {
		// Default order in ascending created date
		query = query.Order("products.created_at DESC")
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

	err := database.GetDB(ctx, r.db.GetDb()).
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
	// Sort deductions by ProductID ascending to prevent deadlocks under concurrent load
	sort.Slice(deductions, func(i, j int) bool {
		return deductions[i].ProductID.String() < deductions[j].ProductID.String()
	})

	db := database.GetDB(ctx, r.db.GetDb())

	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range deductions {
			// Perform the safe mathematical deduction and check stock in one atomic query
			result := tx.Model(&database.Product{}).
				Where("id = ? AND available >= ?", item.ProductID, item.Quantity).
				Update("available", gorm.Expr("available - ?", item.Quantity))

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return fmt.Errorf("insufficient stock for product %s: requested %d", item.ProductID, item.Quantity)
			}
		}

		return nil
	})
}

func (r *productPostgresRepository) AddProductStocks(ctx context.Context, additions []StockDeduction) error {
	// Sort additions by ProductID ascending to prevent deadlocks under concurrent load
	sort.Slice(additions, func(i, j int) bool {
		return additions[i].ProductID.String() < additions[j].ProductID.String()
	})

	db := database.GetDB(ctx, r.db.GetDb())

	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range additions {
			var count int64
			err := tx.Unscoped().Model(&database.Product{}).Where("id = ?", item.ProductID).Count(&count).Error
			if err != nil {
				return err
			}
			if count == 0 {
				log.Printf("Warning: product %s was hard-deleted, skipping stock rollback", item.ProductID)
				continue
			}

			// Increment the stock safely (using Unscoped to find soft-deleted products too)
			err = tx.Unscoped().Model(&database.Product{}).
				Where("id = ?", item.ProductID).
				Update("available", gorm.Expr("available + ?", item.Quantity)).Error

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *productPostgresRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	return r.db.GetDb().WithContext(ctx).
		Where("id = ?", productID).
		Delete(&database.Product{}).Error
}

func (r *productPostgresRepository) GetProductsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.GetDb().WithContext(ctx).Model(&database.Product{}).Count(&count).Error
	return count, err
}
