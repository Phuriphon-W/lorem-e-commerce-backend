package handler

import (
	"bytes"
	"context"
	"errors"
	orderRepo "lorem-backend/internal/modules/order/repository"
	"lorem-backend/internal/modules/payment/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/payment/dto"
	gateway "lorem-backend/internal/modules/payment/gateway"
	productRepo "lorem-backend/internal/modules/product/repository"
	service "lorem-backend/internal/modules/websocket/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ────────────────────────────────────────────────────────────
// Mock Definitions
// ────────────────────────────────────────────────────────────

// ────────────────────────────────────────────────────────────
// Suite Setup
// ────────────────────────────────────────────────────────────

type PaymentHandlerTestSuite struct {
	suite.Suite
	mockPaymentRepo *repository.MockPaymentRepository
	mockOrderRepo   *orderRepo.MockOrderRepository
	mockProductRepo *productRepo.MockProductRepository
	mockGateway     *gateway.MockPaymentGateway
	mockWsService   *service.MockWebsocketService
	handler         PaymentHandler
	ctx             context.Context
}

func (s *PaymentHandlerTestSuite) SetupTest() {
	s.mockPaymentRepo = repository.NewMockPaymentRepository(s.T())
	s.mockOrderRepo = orderRepo.NewMockOrderRepository(s.T())
	s.mockProductRepo = productRepo.NewMockProductRepository(s.T())
	s.mockGateway = gateway.NewMockPaymentGateway(s.T())
	s.mockWsService = service.NewMockWebsocketService(s.T())

	config.GlobalConfig = &config.Config{
		FrontendURL: "http://test-frontend.com",
	}

	s.handler = NewPaymentHandlerImpl(
		s.mockPaymentRepo,
		s.mockOrderRepo,
		s.mockProductRepo,
		s.mockGateway,
		s.mockWsService,
	)
	s.ctx = context.Background()
}

// ────────────────────────────────────────────────────────────
// CreateCheckoutSession Tests
// ────────────────────────────────────────────────────────────

func (s *PaymentHandlerTestSuite) TestCreateCheckoutSession() {
	userID := uuid.New()
	orderID := uuid.New()
	sessionID := "sess_123"
	checkoutURL := "http://checkout.stripe.com/pay"
	expiresAt := time.Now().Add(1 * time.Hour).Unix()
	expiresAtTime := time.Unix(expiresAt, 0)

	testCases := []struct {
		name       string
		input      *dto.CreateCheckoutInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.CreateCheckoutOutputDto)
	}{
		{
			name: "Success - new session created",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  150.00,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()

				successURL := "http://test-frontend.com/payment/success?session_id={CHECKOUT_SESSION_ID}"
				cancelURL := "http://test-frontend.com/payment/failure?session_id={CHECKOUT_SESSION_ID}"
				s.mockGateway.On("CreateCheckoutSession", orderID, float32(150.00), successURL, cancelURL).
					Return(sessionID, checkoutURL, expiresAt, nil).Once()

				s.mockOrderRepo.On("UpdateOrderSession", s.ctx, orderID, sessionID, checkoutURL, &expiresAtTime).
					Return(nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.CreateCheckoutOutputDto) {
				s.NotNil(out)
				s.Equal(checkoutURL, out.Body.CheckoutURL)
				s.Equal(expiresAt, out.Body.ExpiresAt)
			},
		},
		{
			name: "Success - returns existing valid session",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				sessID := "existing_sess"
				sessURL := "http://checkout.stripe.com/existing"
				futureExpires := time.Now().Add(30 * time.Minute)
				order := &database.Order{
					Base:                   database.Base{ID: orderID},
					UserID:                 userID,
					TotalPrice:             150.00,
					OrderStatus:            database.Pending,
					StripeSessionID:        &sessID,
					StripeSessionURL:       &sessURL,
					StripeSessionExpiresAt: &futureExpires,
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.CreateCheckoutOutputDto) {
				s.NotNil(out)
				s.Equal("http://checkout.stripe.com/existing", out.Body.CheckoutURL)
			},
		},
		{
			name: "Failure - order not found",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(nil, errors.New("not found")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
		{
			name: "Failure - order belongs to different user",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      uuid.New(), // different user
					TotalPrice:  150.00,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()
			},
			wantErr:   true,
			errStatus: 403,
		},
		{
			name: "Failure - order not in pending state",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  150.00,
					OrderStatus: database.Paid, // not pending
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()
			},
			wantErr:   true,
			errStatus: 400,
		},
		{
			name: "Failure - gateway CreateCheckoutSession error",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  150.00,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()

				successURL := "http://test-frontend.com/payment/success?session_id={CHECKOUT_SESSION_ID}"
				cancelURL := "http://test-frontend.com/payment/failure?session_id={CHECKOUT_SESSION_ID}"
				s.mockGateway.On("CreateCheckoutSession", orderID, float32(150.00), successURL, cancelURL).
					Return("", "", int64(0), errors.New("gateway error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
		{
			name: "Failure - UpdateOrderSession error",
			input: &dto.CreateCheckoutInputDto{
				Body: dto.CreateCheckoutInputDtoBody{
					UserID:  userID,
					OrderID: orderID,
				},
			},
			setupMocks: func() {
				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  150.00,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()

				successURL := "http://test-frontend.com/payment/success?session_id={CHECKOUT_SESSION_ID}"
				cancelURL := "http://test-frontend.com/payment/failure?session_id={CHECKOUT_SESSION_ID}"
				s.mockGateway.On("CreateCheckoutSession", orderID, float32(150.00), successURL, cancelURL).
					Return(sessionID, checkoutURL, expiresAt, nil).Once()

				s.mockOrderRepo.On("UpdateOrderSession", s.ctx, orderID, sessionID, checkoutURL, &expiresAtTime).
					Return(errors.New("db error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.ctx = context.WithValue(s.ctx, "userID", tc.input.Body.UserID.String())
			tc.setupMocks()

			out, err := s.handler.CreateCheckoutSession(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockOrderRepo.AssertExpectations(s.T())
			s.mockGateway.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// HandleStripeWebhook Tests
// ────────────────────────────────────────────────────────────

func (s *PaymentHandlerTestSuite) TestHandleStripeWebhook() {
	orderID := uuid.New()
	prodID1 := uuid.New()
	prodID2 := uuid.New()
	userID := uuid.New()

	orderItems := []database.OrderItem{
		{
			ProductID: prodID1,
			Quantity:  2,
		},
		{
			ProductID: prodID2,
			Quantity:  1,
		},
	}

	testCases := []struct {
		name           string
		ipAddress      string
		body           []byte
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:           "Forbidden - non-allowed IP",
			ipAddress:      "1.2.3.4",
			body:           []byte("some-payload"),
			setupMocks:     func() {},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Bad request - body read error (too large)",
			ipAddress:      "127.0.0.1",
			body:           []byte(strings.Repeat("A", 70000)),
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Bad request - gateway returns error",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return("", "", errors.New("invalid webhook payload")).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "OK - unhandled event type (ignored)",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return("", "", gateway.ErrUnhandledWebhookEvent).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Internal error - invalid order ID in metadata",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return("not-a-uuid", "success", nil).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "Internal error - order not found",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "success", nil).Once()

				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(nil, errors.New("order not found")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "OK - idempotency skip (order already non-pending)",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "success", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					OrderStatus: database.Paid, // already paid
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "OK - payment failed -> stock reverted + order marked failed + WS notification",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "failed", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					OrderStatus: database.Pending,
					OrderItems:  orderItems,
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()

				expectedAdditions := []productRepo.StockDeduction{
					{ProductID: prodID1, Quantity: 2},
					{ProductID: prodID2, Quantity: 1},
				}
				s.mockProductRepo.On("AddProductStocks", mock.Anything, expectedAdditions).
					Return(nil).Once()

				s.mockOrderRepo.On("UpdateOrderStatus", mock.Anything, orderID, database.Failed).
					Return(nil).Once()

				expectedPayload := service.WSPayload{
					Type: "ORDER_EXPIRED",
					Payload: map[string]string{
						"order_id": orderID.String(),
					},
				}
				s.mockWsService.On("SendToUser", userID, expectedPayload).
					Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Internal error - payment failed but UpdateOrderStatus fails",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "failed", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					OrderStatus: database.Pending,
					OrderItems:  orderItems,
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()

				expectedAdditions := []productRepo.StockDeduction{
					{ProductID: prodID1, Quantity: 2},
					{ProductID: prodID2, Quantity: 1},
				}
				s.mockProductRepo.On("AddProductStocks", mock.Anything, expectedAdditions).
					Return(nil).Once()

				s.mockOrderRepo.On("UpdateOrderStatus", mock.Anything, orderID, database.Failed).
					Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "OK - payment success -> payment record created + order marked paid",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "success", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  125.50,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()

				expectedPayment := &database.Payment{
					OrderID:       orderID,
					UserID:        userID,
					PaymentMethod: "card",
					PaymentAmount: 125.50,
					PaymentStatus: "paid",
				}
				s.mockPaymentRepo.On("CreatePayment", mock.Anything, expectedPayment).
					Return(uuid.New(), nil).Once()

				s.mockOrderRepo.On("UpdateOrderStatus", mock.Anything, orderID, database.Paid).
					Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Internal error - payment success but CreatePayment fails",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "success", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  125.50,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()

				expectedPayment := &database.Payment{
					OrderID:       orderID,
					UserID:        userID,
					PaymentMethod: "card",
					PaymentAmount: 125.50,
					PaymentStatus: "paid",
				}
				s.mockPaymentRepo.On("CreatePayment", mock.Anything, expectedPayment).
					Return(uuid.Nil, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "Internal error - payment success but UpdateOrderStatus fails",
			ipAddress: "127.0.0.1",
			body:      []byte("payload"),
			setupMocks: func() {
				s.mockGateway.On("ExtractOrderEventFromWebhook", []byte("payload"), mock.Anything).
					Return(orderID.String(), "success", nil).Once()

				order := &database.Order{
					Base:        database.Base{ID: orderID},
					UserID:      userID,
					TotalPrice:  125.50,
					OrderStatus: database.Pending,
				}
				s.mockOrderRepo.On("GetOrderByID", mock.Anything, orderID).
					Return(order, nil).Once()

				expectedPayment := &database.Payment{
					OrderID:       orderID,
					UserID:        userID,
					PaymentMethod: "card",
					PaymentAmount: 125.50,
					PaymentStatus: "paid",
				}
				s.mockPaymentRepo.On("CreatePayment", mock.Anything, expectedPayment).
					Return(uuid.New(), nil).Once()

				s.mockOrderRepo.On("UpdateOrderStatus", mock.Anything, orderID, database.Paid).
					Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tc.body))
			req.Header.Set("X-Real-IP", tc.ipAddress)
			req.RemoteAddr = tc.ipAddress + ":1234"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := s.handler.HandleStripeWebhook(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			s.Equal(tc.expectedStatus, rec.Code)

			s.mockGateway.AssertExpectations(s.T())
			s.mockOrderRepo.AssertExpectations(s.T())
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockPaymentRepo.AssertExpectations(s.T())
			s.mockWsService.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// VerifySession Tests
// ────────────────────────────────────────────────────────────

func (s *PaymentHandlerTestSuite) TestVerifySession() {
	sessionID := "sess_123"

	testCases := []struct {
		name       string
		input      *dto.VerifySessionInputDto
		setupMocks func()
		verify     func(*dto.VerifySessionOutputDto, error)
	}{
		{
			name: "Valid session",
			input: &dto.VerifySessionInputDto{
				SessionID: sessionID,
			},
			setupMocks: func() {
				s.mockGateway.On("VerifySessionPayment", sessionID).Return(true, nil).Once()
			},
			verify: func(out *dto.VerifySessionOutputDto, err error) {
				s.Require().NoError(err)
				s.NotNil(out)
				s.True(out.Body.Valid)
			},
		},
		{
			name: "Invalid session - not paid",
			input: &dto.VerifySessionInputDto{
				SessionID: sessionID,
			},
			setupMocks: func() {
				s.mockGateway.On("VerifySessionPayment", sessionID).Return(false, nil).Once()
			},
			verify: func(out *dto.VerifySessionOutputDto, err error) {
				s.Require().NoError(err)
				s.NotNil(out)
				s.False(out.Body.Valid)
			},
		},
		{
			name: "Gateway error",
			input: &dto.VerifySessionInputDto{
				SessionID: sessionID,
			},
			setupMocks: func() {
				s.mockGateway.On("VerifySessionPayment", sessionID).Return(false, errors.New("gateway error")).Once()
			},
			verify: func(out *dto.VerifySessionOutputDto, err error) {
				s.Require().NoError(err)
				s.NotNil(out)
				s.False(out.Body.Valid)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.VerifySession(s.ctx, tc.input)
			tc.verify(out, err)

			s.mockGateway.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// GetUserPaymentsByUserID Tests
// ────────────────────────────────────────────────────────────

func (s *PaymentHandlerTestSuite) TestGetUserPaymentsByUserID() {
	userID := uuid.New()
	paymentID1 := uuid.New()
	paymentID2 := uuid.New()
	orderID1 := uuid.New()
	orderID2 := uuid.New()

	testCases := []struct {
		name       string
		input      *dto.GetPaymentsByUserIdInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.GetPaymentsByUserIdOutputDto)
	}{
		{
			name: "Success - paid status + date_asc",
			input: &dto.GetPaymentsByUserIdInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   20,
				Status:     "paid",
				OrderBy:    "date_asc",
			},
			setupMocks: func() {
				payments := []database.Payment{
					{
						Base:          database.Base{ID: paymentID1, CreatedAt: time.Now()},
						OrderID:       orderID1,
						UserID:        userID,
						PaymentMethod: "card",
						PaymentAmount: 100.00,
						PaymentStatus: "paid",
					},
					{
						Base:          database.Base{ID: paymentID2, CreatedAt: time.Now()},
						OrderID:       orderID2,
						UserID:        userID,
						PaymentMethod: "card",
						PaymentAmount: 150.00,
						PaymentStatus: "paid",
					},
				}
				s.mockPaymentRepo.On("GetUserPaymentsByUserID", s.ctx, userID, int64(1), int64(20), "created_at ASC", "paid").
					Return(payments, int64(2), nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetPaymentsByUserIdOutputDto) {
				s.NotNil(out)
				s.Equal(int64(2), out.Body.Total)
				s.Len(out.Body.Payments, 2)
				s.Equal(paymentID1, out.Body.Payments[0].ID)
				s.Equal(paymentID2, out.Body.Payments[1].ID)
			},
		},
		{
			name: "Success - pending status + default order",
			input: &dto.GetPaymentsByUserIdInputDto{
				UserID:     userID,
				PageNumber: 2,
				PageSize:   5,
				Status:     "pending",
				OrderBy:    "",
			},
			setupMocks: func() {
				payments := []database.Payment{
					{
						Base:          database.Base{ID: paymentID1, CreatedAt: time.Now()},
						OrderID:       orderID1,
						UserID:        userID,
						PaymentMethod: "card",
						PaymentAmount: 100.00,
						PaymentStatus: "pending",
					},
				}
				s.mockPaymentRepo.On("GetUserPaymentsByUserID", s.ctx, userID, int64(2), int64(5), "created_at DESC", "pending").
					Return(payments, int64(12), nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetPaymentsByUserIdOutputDto) {
				s.NotNil(out)
				s.Equal(int64(12), out.Body.Total)
				s.Len(out.Body.Payments, 1)
				s.Equal(paymentID1, out.Body.Payments[0].ID)
			},
		},
		{
			name: "Success - unknown status defaults to no filter",
			input: &dto.GetPaymentsByUserIdInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
				Status:     "unknown",
				OrderBy:    "date_desc",
			},
			setupMocks: func() {
				s.mockPaymentRepo.On("GetUserPaymentsByUserID", s.ctx, userID, int64(1), int64(10), "created_at DESC", "").
					Return([]database.Payment{}, int64(0), nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetPaymentsByUserIdOutputDto) {
				s.NotNil(out)
				s.Equal(int64(0), out.Body.Total)
				s.Len(out.Body.Payments, 0)
			},
		},
		{
			name: "Failure - repo returns error",
			input: &dto.GetPaymentsByUserIdInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
				Status:     "paid",
				OrderBy:    "",
			},
			setupMocks: func() {
				s.mockPaymentRepo.On("GetUserPaymentsByUserID", s.ctx, userID, int64(1), int64(10), "created_at DESC", "paid").
					Return(nil, int64(0), errors.New("repo error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.GetUserPaymentsByUserID(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockPaymentRepo.AssertExpectations(s.T())
		})
	}
}

func TestPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentHandlerTestSuite))
}
