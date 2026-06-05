package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/payment/dto"
	"lorem-backend/internal/modules/payment/gateway"
	productRepository "lorem-backend/internal/modules/product/repository"
	wsService "lorem-backend/internal/modules/websocket/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ────────────────────────────────────────────────────────────
// Mock Definitions
// ────────────────────────────────────────────────────────────

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) CreatePayment(ctx context.Context, payment *database.Payment) (uuid.UUID, error) {
	args := m.Called(ctx, payment)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPaymentRepository) GetUserPaymentByOrderID(ctx context.Context, orderID, userID uuid.UUID) (*database.Payment, error) {
	args := m.Called(ctx, orderID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPaymentRepository) GetUserPaymentsByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int64, orderBy, status string) ([]database.Payment, int64, error) {
	args := m.Called(ctx, userID, page, pageSize, orderBy, status)
	if args.Get(0) != nil {
		return args.Get(0).([]database.Payment), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockPaymentRepository) UpdatePaymentStatusByOrderID(ctx context.Context, orderID uuid.UUID, status string) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order *database.Order) (uuid.UUID, error) {
	args := m.Called(ctx, order)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockOrderRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int64, status string, orderBy string) ([]database.Order, int64, error) {
	args := m.Called(ctx, userID, page, pageSize, status, orderBy)
	if args.Get(0) != nil {
		return args.Get(0).([]database.Order), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockOrderRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*database.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status database.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderRepository) UpdateOrderSession(ctx context.Context, orderID uuid.UUID, sessionID, sessionURL string, expiresAt *time.Time) error {
	args := m.Called(ctx, orderID, sessionID, sessionURL, expiresAt)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) CreateProduct(ctx context.Context, product *database.Product) (uuid.UUID, error) {
	args := m.Called(ctx, product)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockProductRepository) GetProducts(ctx context.Context, page int64, pageSize int64, category, search, order string) ([]database.Product, int64, error) {
	args := m.Called(ctx, page, pageSize, category, search, order)
	if args.Get(0) != nil {
		return args.Get(0).([]database.Product), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, productID uuid.UUID) (*database.Product, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductRepository) GetProductsByIDs(ctx context.Context, productIDs []uuid.UUID) ([]database.Product, error) {
	args := m.Called(ctx, productIDs)
	if args.Get(0) != nil {
		return args.Get(0).([]database.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductRepository) GetProductStock(ctx context.Context, productId uuid.UUID) (uint, error) {
	args := m.Called(ctx, productId)
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockProductRepository) UpdateProductByID(ctx context.Context, productID uuid.UUID, updateData map[string]interface{}) error {
	args := m.Called(ctx, productID, updateData)
	return args.Error(0)
}

func (m *MockProductRepository) DeductProductStocks(ctx context.Context, deductions []productRepository.StockDeduction) error {
	args := m.Called(ctx, deductions)
	return args.Error(0)
}

func (m *MockProductRepository) AddProductStocks(ctx context.Context, additions []productRepository.StockDeduction) error {
	args := m.Called(ctx, additions)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreateCheckoutSession(orderID uuid.UUID, totalPrice float32, successURL, cancelURL string) (string, string, int64, error) {
	args := m.Called(orderID, totalPrice, successURL, cancelURL)
	return args.String(0), args.String(1), args.Get(2).(int64), args.Error(3)
}

func (m *MockPaymentGateway) ExtractOrderEventFromWebhook(payload []byte, c echo.Context) (string, string, error) {
	args := m.Called(payload, c)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockPaymentGateway) VerifySessionPayment(sessionID string) (bool, error) {
	args := m.Called(sessionID)
	return args.Bool(0), args.Error(1)
}

type MockWebsocketService struct {
	mock.Mock
}

func (m *MockWebsocketService) AddClient(userID uuid.UUID, conn *websocket.Conn) {
	m.Called(userID, conn)
}

func (m *MockWebsocketService) RemoveClient(userID uuid.UUID, conn *websocket.Conn) {
	m.Called(userID, conn)
}

func (m *MockWebsocketService) SendToUser(userID uuid.UUID, message wsService.WSPayload) {
	m.Called(userID, message)
}

func (m *MockWebsocketService) WebsocketHandler(c echo.Context) error {
	args := m.Called(c)
	return args.Error(0)
}

// ────────────────────────────────────────────────────────────
// Suite Setup
// ────────────────────────────────────────────────────────────

type PaymentHandlerTestSuite struct {
	suite.Suite
	mockPaymentRepo *MockPaymentRepository
	mockOrderRepo   *MockOrderRepository
	mockProductRepo *MockProductRepository
	mockGateway     *MockPaymentGateway
	mockWsService   *MockWebsocketService
	handler         PaymentHandler
	ctx             context.Context
}

func (s *PaymentHandlerTestSuite) SetupTest() {
	s.mockPaymentRepo = new(MockPaymentRepository)
	s.mockOrderRepo = new(MockOrderRepository)
	s.mockProductRepo = new(MockProductRepository)
	s.mockGateway = new(MockPaymentGateway)
	s.mockWsService = new(MockWebsocketService)

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

				expectedAdditions := []productRepository.StockDeduction{
					{ProductID: prodID1, Quantity: 2},
					{ProductID: prodID2, Quantity: 1},
				}
				s.mockProductRepo.On("AddProductStocks", mock.Anything, expectedAdditions).
					Return(nil).Once()

				s.mockOrderRepo.On("UpdateOrderStatus", mock.Anything, orderID, database.Failed).
					Return(nil).Once()

				expectedPayload := wsService.WSPayload{
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

				expectedAdditions := []productRepository.StockDeduction{
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
