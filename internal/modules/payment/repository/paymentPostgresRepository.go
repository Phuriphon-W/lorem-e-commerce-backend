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

func (r *paymentPostgresRepository) GetUserPaymentsByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int64, orderBy, status string) ([]database.Payment, int64, error) {
	var payments []database.Payment
	var total int64

	query := r.db.GetDb().WithContext(ctx).Model(&database.Payment{}).Where("user_id = ?", userID)

	// Filter By Status (Paid or Pending)
	if status != "" {
		query = query.Where("payment_status = ?", status)
	}

	// Count Total Records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply Ordering
	if orderBy == "" {
		query = query.Order("created_at DESC") // Newest first by default
	} else {
		query = query.Order(orderBy)
	}

	offSet := (page - 1) * pageSize

	err := query.
		Limit(int(pageSize)).
		Offset(int(offSet)).
		Find(&payments).Error

	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}
