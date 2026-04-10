package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type AuthRepository interface {
	RegisterUser(ctx context.Context, user *database.User) (uuid.UUID, string, error)
	GetUserByEmail(ctx context.Context, email string) (*struct {
		ID           uuid.UUID
		Username     string
		PasswordHash string
	}, error)
	GetUserByUsername(ctx context.Context, username string) (*struct {
		ID       uuid.UUID
		Username string
	}, error)
	// SignOut(ctx context.Context)
}
