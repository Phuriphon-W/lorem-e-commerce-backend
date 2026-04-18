package gateway

import (
	"encoding/json"
	"errors"
	"lorem-backend/internal/config"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/webhook"
)

var ErrUnhandledWebhookEvent = errors.New("unhandled webhook event")

type stripePaymentGateway struct {
	webhookSecret string
}

func NewStripePaymentGateway(secretKey string, webhookSecret string) PaymentGateway {
	// Initialize Stripe securely here
	stripe.Key = secretKey
	return &stripePaymentGateway{
		webhookSecret: webhookSecret,
	}
}

func (s *stripePaymentGateway) CreateCheckoutSession(orderID uuid.UUID, totalPrice float32, successURL, cancelURL string) (string, error) {
	amountInCents := int64(totalPrice * 100)

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Order #" + orderID.String()[:8]),
					},
					UnitAmount: stripe.Int64(amountInCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		ExpiresAt:  stripe.Int64(time.Now().Add(config.GlobalConfig.StripeSessionExpire).Unix()),
	}

	params.AddMetadata("order_id", orderID.String())

	stripeSession, err := session.New(params)
	if err != nil {
		return "", err
	}

	return stripeSession.URL, nil
}

func (s *stripePaymentGateway) ExtractOrderEventFromWebhook(payload []byte, c echo.Context) (string, string, error) {
	signatureHeader := c.Request().Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(
		payload,
		signatureHeader,
		s.webhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		return "", "", errors.New("invalid signature")
	}

	// Extract the session data
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return "", "", errors.New("error parsing webhook JSON")
	}

	orderIDStr := session.Metadata["order_id"]

	switch event.Type {
	case "checkout.session.completed":
		return orderIDStr, "success", nil

	case "checkout.session.expired", "checkout.session.async_payment_failed":
		return orderIDStr, "failed", nil

	default:
		// Safely ignore events we don't care about
		return "", "", ErrUnhandledWebhookEvent
	}
}

func (s *stripePaymentGateway) VerifySessionPayment(sessionID string) (bool, error) {
	// Ask Stripe for the session details
	stripeSession, err := session.Get(sessionID, nil)
	if err != nil {
		return false, err
	}

	// Check if the user actually paid
	if stripeSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		return true, nil
	}

	return false, nil
}
