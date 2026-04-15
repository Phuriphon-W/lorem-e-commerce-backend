package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *database.Order) (uuid.UUID, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int64, status string, orderBy string) ([]database.Order, int64, error)
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (*database.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status database.OrderStatus) error
}
