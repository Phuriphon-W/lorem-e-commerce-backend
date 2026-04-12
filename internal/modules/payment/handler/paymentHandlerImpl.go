package handler

import (
	"context"
	"errors"
	"io"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	orderRepository "lorem-backend/internal/modules/order/repository"
	"lorem-backend/internal/modules/payment/dto"
	"lorem-backend/internal/modules/payment/gateway"
	"lorem-backend/internal/modules/payment/repository"
	"lorem-backend/internal/utils"
	"net/http"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type paymentHandlerImpl struct {
	paymentRepo    repository.PaymentRepository
	orderRepo      orderRepository.OrderRepository
	paymentGateway gateway.PaymentGateway
}

func NewPaymentHandlerImpl(
	payRepo repository.PaymentRepository,
	ordRepo orderRepository.OrderRepository,
	payGateway gateway.PaymentGateway,
) PaymentHandler {
	return &paymentHandlerImpl{
		paymentRepo:    payRepo,
		orderRepo:      ordRepo,
		paymentGateway: payGateway,
	}
}

func (h *paymentHandlerImpl) CreateCheckoutSession(ctx context.Context, input *dto.CreateCheckoutInputDto) (*dto.CreateCheckoutOutputDto, error) {
	// Get order to proceed payment
	order, err := h.orderRepo.GetOrderByID(ctx, input.Body.OrderID)
	if err != nil {
		return nil, huma.Error404NotFound("Order not found", err)
	}

	// Check if the order belongs to the user who made request
	if order.UserID != input.Body.UserID {
		return nil, huma.Error403Forbidden("You are not allowed to make payment for this order")
	}

	// Check order state
	if order.OrderStatus != database.Pending {
		return nil, huma.Error400BadRequest("Order is not in pending state")
	}

	// TODO: Edit to match actual frontend endpoint for checkout
	successURL := config.GlobalConfig.FrontendURL + "/checkout/success?sessionId={CHECKOUT_SESSION_ID}"
	cancelURL := config.GlobalConfig.FrontendURL + "/checkout/cancel"

	// Ask the Gateway to create the session
	checkoutURL, err := h.paymentGateway.CreateCheckoutSession(order, successURL, cancelURL)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initialize payment gateway", err)
	}

	// Check if the payment for this order already exist (pending)
	_, err = h.paymentRepo.GetUserPaymentByOrderID(ctx, input.Body.OrderID, input.Body.UserID)
	if err != nil {
		// Error cause by record does not exists, create a new one
		if errors.Is(err, gorm.ErrRecordNotFound) {
			payment := &database.Payment{
				OrderID:       order.ID,
				UserID:        order.UserID,
				PaymentMethod: "Card",
				PaymentAmount: float64(order.TotalPrice),
				PaymentStatus: "pending",
			}
			_, createErr := h.paymentRepo.CreatePayment(ctx, payment)
			if createErr != nil {
				return nil, huma.Error500InternalServerError("Error Creating Payment", createErr)
			}
		} else {
			// Other internal error
			return nil, huma.Error500InternalServerError("Error occurred while checking payment", err)
		}
	}
	// Note: If err == nil, the payment ALREADY exists. We do nothing,

	// Return the secure URL to the frontend
	return &dto.CreateCheckoutOutputDto{
		Body: dto.CreateCheckoutOutputDtoBody{
			CheckoutURL: checkoutURL,
		},
	}, nil
}

func (h *paymentHandlerImpl) HandleStripeWebhook(c echo.Context) error {
	req := c.Request()
	res := c.Response()
	ipFromStripeWebhook := c.RealIP()

	// Checks webhook coming from allowed IP
	if !slices.Contains(utils.AllowedStripeIPs[:], ipFromStripeWebhook) {
		return c.String(http.StatusForbidden, "IP not allowed")
	}

	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(res.Writer, req.Body, MaxBodyBytes)

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "Error reading request body")
	}

	// Ask the Gateway to verify and parse the webhook
	orderIDStr, err := h.paymentGateway.ExtractOrderIDFromWebhook(payload, c)
	if err != nil {
		if errors.Is(err, gateway.ErrUnhandledWebhookEvent) {
			// It's a valid webhook, but we don't care about this event type. Safely ignore.
			return c.NoContent(http.StatusOK)
		}
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Parse the returned ID
	parsedOrderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Invalid order ID in metadata")
	}

	// Update internal Database
	ctx := req.Context()
	err = h.paymentRepo.UpdatePaymentStatusByOrderID(ctx, parsedOrderID, "paid")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Database error")
	}

	err = h.orderRepo.UpdateOrderStatus(ctx, parsedOrderID, database.Paid)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Database error")
	}

	return c.NoContent(http.StatusOK)
}

func (h *paymentHandlerImpl) VerifySession(ctx context.Context, input *dto.VerifySessionInputDto) (*dto.VerifySessionOutputDto, error) {
	// Check if the provided sessionId is valid
	isValid, err := h.paymentGateway.VerifySessionPayment(input.SessionID)

	if err != nil || !isValid {
		// We return a 200 OK, but with valid: false so the frontend can handle it gracefully
		return &dto.VerifySessionOutputDto{
			Body: dto.VerifySessionOutputDtoBody{
				Valid: false,
			},
		}, nil
	}

	return &dto.VerifySessionOutputDto{
		Body: dto.VerifySessionOutputDtoBody{
			Valid: true,
		},
	}, nil
}
