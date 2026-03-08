package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type AuthRepository interface {
	RegisterUser(ctx context.Context, user *database.User) (uuid.UUID, string, error)
	// SignIn(ctx context.Context)
	// SignOut(ctx context.Context)
}
