package handler

import (
	"context"
	"errors"
	"io"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	orderRepository "lorem-backend/internal/modules/order/repository"
	productRepository "lorem-backend/internal/modules/product/repository"

	"lorem-backend/internal/modules/payment/dto"
	"lorem-backend/internal/modules/payment/gateway"
	"lorem-backend/internal/modules/payment/repository"
	wsService "lorem-backend/internal/modules/websocket/service"
	"lorem-backend/internal/utils"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type paymentHandlerImpl struct {
	paymentRepo    repository.PaymentRepository
	orderRepo      orderRepository.OrderRepository
	productRepo    productRepository.ProductRepository
	paymentGateway gateway.PaymentGateway
	wsService      wsService.WebsocketService
}

func NewPaymentHandlerImpl(
	payRepo repository.PaymentRepository,
	ordRepo orderRepository.OrderRepository,
	productRepo productRepository.ProductRepository,
	payGateway gateway.PaymentGateway,
	wsService wsService.WebsocketService,
) PaymentHandler {
	return &paymentHandlerImpl{
		paymentRepo:    payRepo,
		orderRepo:      ordRepo,
		productRepo:    productRepo,
		paymentGateway: payGateway,
		wsService:      wsService,
	}
}

func (h *paymentHandlerImpl) CreateCheckoutSession(ctx context.Context, input *dto.CreateCheckoutInputDto) (*dto.CreateCheckoutOutputDto, error) {
	// Ownership verification
	authenticatedUserIDStr, ok := ctx.Value("userID").(string)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}
	isAdmin, _ := ctx.Value("isAdmin").(bool)
	if !isAdmin && authenticatedUserIDStr != input.Body.UserID.String() {
		return nil, huma.Error403Forbidden("Forbidden: You do not own this resource")
	}

	// Fetch the existing order
	order, err := h.orderRepo.GetOrderByID(ctx, input.Body.OrderID)
	if err != nil {
		return nil, huma.Error404NotFound("Order not found")
	}

	// Verify the order belongs to the user and is still pending
	if order.UserID != input.Body.UserID {
		return nil, huma.Error403Forbidden("This order does not belong to you")
	}

	if order.OrderStatus != database.Pending {
		return nil, huma.Error400BadRequest("Order is not in a payable state")
	}

	// Check if existing session is valid
	if order.StripeSessionID != nil && order.StripeSessionExpiresAt != nil && order.StripeSessionURL != nil && order.StripeSessionExpiresAt.After(time.Now()) {
		return &dto.CreateCheckoutOutputDto{
			Body: dto.CreateCheckoutOutputDtoBody{
				CheckoutURL: *order.StripeSessionURL,
				ExpiresAt:   order.StripeSessionExpiresAt.Unix(),
			},
		}, nil
	}

	successURL := config.GlobalConfig.FrontendURL + "/payment/success?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := config.GlobalConfig.FrontendURL + "/payment/failure?session_id={CHECKOUT_SESSION_ID}"

	// Ask the Gateway to create the session
	sessionID, checkoutURL, expiresAt, err := h.paymentGateway.CreateCheckoutSession(order.ID, order.TotalPrice, successURL, cancelURL)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initialize payment gateway", err)
	}

	expiresAtTime := time.Unix(expiresAt, 0)
	err = h.orderRepo.UpdateOrderSession(ctx, order.ID, sessionID, checkoutURL, &expiresAtTime)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to save payment session", err)
	}

	// Return the secure URL to the frontend
	return &dto.CreateCheckoutOutputDto{
		Body: dto.CreateCheckoutOutputDtoBody{
			CheckoutURL: checkoutURL,
			ExpiresAt:   expiresAt,
		},
	}, nil
}

func (h *paymentHandlerImpl) HandleStripeWebhook(c echo.Context) error {
	req := c.Request()
	res := c.Response()
	ipFromStripeWebhook := c.RealIP()

	// Checks webhook coming from allowed IP
	parsedIP := net.ParseIP(ipFromStripeWebhook)
	if !slices.Contains(utils.AllowedStripeIPs[:], ipFromStripeWebhook) && (parsedIP == nil || (!parsedIP.IsLoopback() && !parsedIP.IsPrivate())) {
		return c.JSON(http.StatusForbidden, utils.CreateErrorResponse(http.StatusForbidden, "IP Not Allowed"))
	}

	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(res.Writer, req.Body, MaxBodyBytes)

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.CreateErrorResponse(http.StatusForbidden, "Error reading request body"))
	}

	// Ask the Gateway to verify and parse the webhook
	orderIDStr, paymentStatus, err := h.paymentGateway.ExtractOrderEventFromWebhook(payload, c)
	if err != nil {
		// Error is due to ignored event. Skip it
		if errors.Is(err, gateway.ErrUnhandledWebhookEvent) {
			return c.NoContent(http.StatusOK)
		}
		return c.JSON(http.StatusBadRequest, utils.CreateErrorResponse(http.StatusBadRequest, err.Error()))
	}

	// Parse the returned ID
	parsedOrderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse(http.StatusInternalServerError, "Invalid order ID in metadata"))
	}

	ctx := req.Context()

	// Fetch the Order to get user info, total price, and items
	order, err := h.orderRepo.GetOrderByID(ctx, parsedOrderID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse(http.StatusInternalServerError, "Failed to retrieve order details"))
	}

	// IDEMPOTENCY CHECK: If Stripe sends this webhook twice, don't process it again
	if order.OrderStatus != database.Pending {
		return c.NoContent(http.StatusOK)
	}

	// Stripe failed
	if paymentStatus == "failed" {
		// Revert stock
		additions := make([]productRepository.StockDeduction, len(order.OrderItems))
		for i, item := range order.OrderItems {
			additions[i] = productRepository.StockDeduction{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
		}

		_ = h.productRepo.AddProductStocks(ctx, additions)

		// Mark order as failed
		err = h.orderRepo.UpdateOrderStatus(ctx, parsedOrderID, database.Failed)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse(http.StatusInternalServerError, "Failed to update order to failed status"))
		}

		h.wsService.SendToUser(order.UserID, wsService.WSPayload{
			Type: "ORDER_EXPIRED",
			Payload: map[string]string{
				"order_id": parsedOrderID.String(),
			},
		})

		return c.NoContent(http.StatusOK)
	}

	// Stripe succeed
	payment := &database.Payment{
		OrderID:       order.ID,
		UserID:        order.UserID,
		PaymentMethod: "card",
		PaymentAmount: float64(order.TotalPrice),
		PaymentStatus: "paid",
	}

	// Create payment
	_, createErr := h.paymentRepo.CreatePayment(ctx, payment)
	if createErr != nil {
		return c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse(http.StatusInternalServerError, "Error creating payment record"))
	}

	// Mark as paid
	err = h.orderRepo.UpdateOrderStatus(ctx, parsedOrderID, database.Paid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse(http.StatusInternalServerError, "Database error updating order to shipping"))
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

func (h *paymentHandlerImpl) GetUserPaymentsByUserID(ctx context.Context, input *dto.GetPaymentsByUserIdInputDto) (*dto.GetPaymentsByUserIdOutputDto, error) {
	var statusFilter string

	switch input.Status {
	case "pending":
		statusFilter = "pending"
	case "paid":
		statusFilter = "paid"
	default:
		statusFilter = ""
	}

	var queryOrder string

	if input.OrderBy == "date_asc" {
		queryOrder = "created_at ASC"
	} else {
		queryOrder = "created_at DESC"
	}

	userPayments, total, err := h.paymentRepo.GetUserPaymentsByUserID(
		ctx,
		input.UserID,
		input.PageNumber,
		input.PageSize,
		queryOrder,
		statusFilter,
	)

	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve payment records", err)
	}

	payments := make([]dto.PaymentDto, len(userPayments))
	for i, payment := range userPayments {
		payments[i] = dto.PaymentDto{
			ID:            payment.ID,
			OrderID:       payment.OrderID,
			PaymentMethod: payment.PaymentMethod,
			PaymentAmount: payment.PaymentAmount,
			PaymentStatus: payment.PaymentStatus,
			CreatedAt:     payment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &dto.GetPaymentsByUserIdOutputDto{
		Body: dto.GetPaymentsByUserIdOutputDtoBody{
			Payments: payments,
			Total:    total,
		},
	}, nil
}
