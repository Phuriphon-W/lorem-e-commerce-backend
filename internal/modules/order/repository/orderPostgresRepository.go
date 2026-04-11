package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type orderPostgresRepository struct {
	db database.Database
}

func NewOrderPostgresRepository(db database.Database) OrderRepository {
	return &orderPostgresRepository{
		db: db,
	}
}

func (r *orderPostgresRepository) CreateOrder(ctx context.Context, order *database.Order) (uuid.UUID, error) {
	result := gorm.WithResult()

	err := gorm.G[database.Order](r.db.GetDb(), result).Create(ctx, order)
	if err != nil {
		return uuid.Nil, err
	}

	return order.ID, nil
}

func (r *orderPostgresRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID, page, pageSize uint64) ([]database.Order, int64, error) {
	var orders []database.Order
	var total int64
	query := r.db.GetDb().WithContext(ctx).Model(&database.Order{}).Where("user_id = ?", userID)

	// Count Total Records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate Page Offset
	offset := (page - 1) * pageSize

	// Fetch orders with items preloaded
	err := query.
		Order("created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Preload("OrderItems", func(db gorm.PreloadBuilder) error {
			db.Order("created_at ASC")
			return nil
		}).
		Preload("OrderItems.Product", nil).
		Find(&orders).Error

	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *orderPostgresRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*database.Order, error) {
	order, err := gorm.G[database.Order](r.db.GetDb()).
		Where("id = ?", orderID).
		Preload("OrderItems", nil).
		Preload("OrderItems.Product", nil).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderPostgresRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status database.OrderStatus) error {
	return r.db.GetDb().WithContext(ctx).
		Model(&database.Order{}).
		Where("id = ?", orderID).
		Update("order_status", status).Error
}
