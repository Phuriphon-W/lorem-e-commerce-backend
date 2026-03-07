package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, category *database.Category) (uuid.UUID, error)
	GetCategoryByID(ctx context.Context, catID uuid.UUID) (*database.Category, error)
}
