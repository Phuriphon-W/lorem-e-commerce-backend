package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetUsers(ctx context.Context, page, pageSize int64, search, order string) ([]database.User, int64, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*database.User, error)
	UpdateUser(ctx context.Context, user *database.User) error
	GetUsersCount(ctx context.Context) (int64, error)
}
