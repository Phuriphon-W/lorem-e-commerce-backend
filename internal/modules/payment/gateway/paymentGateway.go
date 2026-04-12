package gateway

import (
	"lorem-backend/internal/database"

	"github.com/labstack/echo/v4"
)

type PaymentGateway interface {
	CreateCheckoutSession(order *database.Order, successURL, cancelURL string) (string, error)
	ExtractOrderIDFromWebhook(payload []byte, c echo.Context) (string, error)
	VerifySessionPayment(sessionID string) (bool, error)
}
