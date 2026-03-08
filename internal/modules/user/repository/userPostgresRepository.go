package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userPostgresRepository struct {
	db database.Database
}

func NewUserPostgresRepository(db database.Database) UserRepository {
	return &userPostgresRepository{
		db: db,
	}
}

func (r *userPostgresRepository) CreateUser(ctx context.Context, user *database.User) (uuid.UUID, error) {
	result := gorm.WithResult()

	err := r.db.GetDb().Transaction(func(tx *gorm.DB) error {

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
		return uuid.Nil, err
	}

	return user.ID, nil
}

func (r *userPostgresRepository) GetUsers(ctx context.Context) ([]database.User, error) {
	users, err := gorm.G[database.User](r.db.GetDb()).Find(ctx)

	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userPostgresRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*database.User, error) {
	user, err := gorm.G[database.User](r.db.GetDb()).Where("id = ?", userID).First(ctx)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
