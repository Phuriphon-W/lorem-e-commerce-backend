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

func (r *userPostgresRepository) UpdateUser(ctx context.Context, user *database.User) error {
	err := r.db.GetDb().WithContext(ctx).Save(user).Error

	if err != nil {
		return err
	}

	return nil
}
