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

func (r *userPostgresRepository) GetUsers(ctx context.Context, page, pageSize int64, search, order string) ([]database.User, int64, error) {
	var users []database.User
	var total int64

	query := r.db.GetDb().WithContext(ctx).Model(&database.User{})

	// Filter by Search
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("username ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// Count Total Records after search filters
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply Ordering
	if order != "" {
		query = query.Order(order)
	} else {
		// Default order: created_at DESC
		query = query.Order("created_at DESC")
	}

	offset := (page - 1) * pageSize
	err := query.Limit(int(pageSize)).Offset(int(offset)).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
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

func (r *userPostgresRepository) GetUsersCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.GetDb().WithContext(ctx).Model(&database.User{}).Count(&count).Error
	return count, err
}
