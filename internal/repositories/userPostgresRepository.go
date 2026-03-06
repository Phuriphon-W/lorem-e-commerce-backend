package repositories

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

	err := gorm.G[database.User](r.db.GetDb(), result).Create(ctx, user)
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

func (r *userPostgresRepository) GetUserByID(ctx context.Context, userID uint) (*database.User, error) {
	user, err := gorm.G[database.User](r.db.GetDb()).Where("id = ?", userID).First(ctx)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
