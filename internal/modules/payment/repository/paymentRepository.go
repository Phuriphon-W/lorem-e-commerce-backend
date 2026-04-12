package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *database.Payment) (uuid.UUID, error)
	GetUserPaymentByOrderID(ctx context.Context, orderID, userID uuid.UUID) (*database.Payment, error)
	UpdatePaymentStatusByOrderID(ctx context.Context, orderID uuid.UUID, status string) error
}
