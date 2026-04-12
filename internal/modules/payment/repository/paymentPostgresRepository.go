package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type paymentPostgresRepository struct {
	db database.Database
}

func NewPaymentPostgresRepository(db database.Database) PaymentRepository {
	return &paymentPostgresRepository{db: db}
}

func (r *paymentPostgresRepository) CreatePayment(ctx context.Context, payment *database.Payment) (uuid.UUID, error) {
	result := gorm.WithResult()
	err := gorm.G[database.Payment](r.db.GetDb(), result).Create(ctx, payment)
	if err != nil {
		return uuid.Nil, err
	}
	return payment.ID, nil
}

func (r *paymentPostgresRepository) UpdatePaymentStatusByOrderID(ctx context.Context, orderID uuid.UUID, status string) error {
	return r.db.GetDb().WithContext(ctx).
		Model(&database.Payment{}).
		Where("order_id = ?", orderID).
		Update("payment_status", status).Error
}

func (r *paymentPostgresRepository) GetUserPaymentByOrderID(ctx context.Context, orderID, userID uuid.UUID) (*database.Payment, error) {
	payment, err := gorm.G[database.Payment](r.db.GetDb()).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &payment, nil
}
