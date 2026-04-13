package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetUsers(ctx context.Context) ([]database.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*database.User, error)
	UpdateUser(ctx context.Context, user *database.User) error
}
