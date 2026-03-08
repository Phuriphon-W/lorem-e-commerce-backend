package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type authPostgresRepository struct {
	db database.Database
}

func NewAuthPostgresRepository(db database.Database) AuthRepository {
	return &authPostgresRepository{
		db: db,
	}
}

func (a *authPostgresRepository) RegisterUser(ctx context.Context, user *database.User) (uuid.UUID, string, error) {
	result := gorm.WithResult()

	err := a.db.GetDb().Transaction(func(tx *gorm.DB) error {

		// Create User
		if err := gorm.G[database.User](tx, result).Create(ctx, user); err != nil {
			return err // This triggers a rollback
		}

		// Create Cart
		cart := &database.Cart{UserID: user.ID}
		if err := gorm.G[database.Cart](tx).Create(ctx, cart); err != nil {
			return err // This triggers a rollback
		}

		return nil // This triggers a commit
	})

	if err != nil {
		return uuid.Nil, "", err
	}

	return user.ID, user.Username, nil
}
