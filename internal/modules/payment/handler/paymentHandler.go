package handler

import (
	"context"
	"lorem-backend/internal/modules/payment/dto"

	"github.com/labstack/echo/v4"
)

type PaymentHandler interface {
	CreateCheckoutSession(ctx context.Context, input *dto.CreateCheckoutInputDto) (*dto.CreateCheckoutOutputDto, error)
	HandleStripeWebhook(c echo.Context) error
	GetUserPaymentsByUserID(ctx context.Context, input *dto.GetPaymentsByUserIdInputDto) (*dto.GetPaymentsByUserIdOutputDto, error)
}
