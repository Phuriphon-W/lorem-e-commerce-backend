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
	result := gorm.WithResult()

	err := gorm.G[database.Category](c.db.GetDb(), result).Create(ctx, category)
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
