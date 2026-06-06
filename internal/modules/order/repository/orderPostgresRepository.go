package repository

import (
	"context"
	"errors"
	"lorem-backend/internal/database"

	"time"

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
	if len(order.OrderItems) == 0 {
		return uuid.Nil, errors.New("cannot create an order without items")
	}

	result := gorm.WithResult()

	err := gorm.G[database.Order](r.db.GetDb(), result).Create(ctx, order)
	if err != nil {
		return uuid.Nil, err
	}

	return order.ID, nil
}

func (r *orderPostgresRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int64, status string, orderBy string) ([]database.Order, int64, error) {
	var orders []database.Order
	var total int64
	query := r.db.GetDb().WithContext(ctx).Model(&database.Order{}).Where("user_id = ?", userID)

	// Filter By Order Status
	if status != "" {
		query = query.Where("order_status = ?", status)
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	// Count Total Records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate Page Offset
	offset := (page - 1) * pageSize

	// Fetch orders with items preloaded
	err := query.
		Limit(int(pageSize)).
		Offset(int(offset)).
		Preload("OrderItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("OrderItems.Product", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Find(&orders).Error

	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *orderPostgresRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*database.Order, error) {
	var order database.Order
	err := r.db.GetDb().WithContext(ctx).
		Where("id = ?", orderID).
		Preload("OrderItems").
		Preload("OrderItems.Product", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		First(&order).Error

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

func (r *orderPostgresRepository) UpdateOrderSession(ctx context.Context, orderID uuid.UUID, sessionID, sessionURL string, expiresAt *time.Time) error {
	return r.db.GetDb().WithContext(ctx).
		Model(&database.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"stripe_session_id":         sessionID,
			"stripe_session_url":        sessionURL,
			"stripe_session_expires_at": expiresAt,
		}).Error
}

func (r *orderPostgresRepository) GetOrdersCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.GetDb().WithContext(ctx).Model(&database.Order{}).Count(&count).Error
	return count, err
}
