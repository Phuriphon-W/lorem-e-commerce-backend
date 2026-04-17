package gateway

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PaymentGateway interface {
	CreateCheckoutSession(orderID uuid.UUID, totalPrice float32, successURL, cancelURL string) (string, error)
	ExtractOrderEventFromWebhook(payload []byte, c echo.Context) (string, string, error)
	VerifySessionPayment(sessionID string) (bool, error)
}
