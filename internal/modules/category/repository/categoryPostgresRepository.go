package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type categoryPostgresRepository struct {
	db database.Database
}

func NewCategoryPostgresRepository(db database.Database) CategoryRepository {
	return &categoryPostgresRepository{
		db: db,
	}
}

func (c *categoryPostgresRepository) CreateCategory(ctx context.Context, category *database.Category) (uuid.UUID, error) {
	var existing database.Category
	err := c.db.GetDb().WithContext(ctx).Unscoped().Where("name = ?", category.Name).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			err = c.db.GetDb().WithContext(ctx).Unscoped().
				Model(&database.Category{}).
				Where("id = ?", existing.ID).
				Updates(map[string]interface{}{
					"deleted_at": nil,
					"name":       category.Name,
				}).Error
			if err != nil {
				return uuid.Nil, err
			}
			category.ID = existing.ID
			return existing.ID, nil
		}
	}

	result := gorm.WithResult()

	err = gorm.G[database.Category](c.db.GetDb(), result).Create(ctx, category)
	if err != nil {
		return uuid.Nil, err
	}

	return category.ID, nil
}

func (c *categoryPostgresRepository) GetCategoryByID(ctx context.Context, catID uuid.UUID) (*database.Category, error) {
	category, err := gorm.G[database.Category](c.db.GetDb()).Where("id = ?", catID).First(ctx)

	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (c *categoryPostgresRepository) GetCategories(ctx context.Context) ([]database.Category, error) {
	categories, err := gorm.G[database.Category](c.db.GetDb()).Find(ctx)

	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (c *categoryPostgresRepository) UpdateCategoryByID(ctx context.Context, catID uuid.UUID, name string) error {
	return c.db.GetDb().WithContext(ctx).
		Model(&database.Category{}).
		Where("id = ?", catID).
		Updates(map[string]interface{}{"name": name}).Error
}

func (c *categoryPostgresRepository) DeleteCategoryByID(ctx context.Context, catID uuid.UUID) error {
	return c.db.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Cascade soft delete to all products belonging to this category
		if err := tx.Where("category_id = ?", catID).Delete(&database.Product{}).Error; err != nil {
			return err
		}

		// Delete the category itself
		if err := tx.Where("id = ?", catID).Delete(&database.Category{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (c *categoryPostgresRepository) GetCategoriesCount(ctx context.Context) (int64, error) {
	var count int64
	err := c.db.GetDb().WithContext(ctx).Model(&database.Category{}).Count(&count).Error
	return count, err
}
