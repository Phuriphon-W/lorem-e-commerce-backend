package handler

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"lorem-backend/internal/database"
	fileRepo "lorem-backend/internal/modules/file/repository"
	"lorem-backend/internal/modules/order/dto"
	productRepo "lorem-backend/internal/modules/product/repository"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ────────────────────────────────────────────────────────────
// Mock Implementations
// ────────────────────────────────────────────────────────────

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

func (m *MockOrderRepository) GetOrdersCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return int64(args.Int(0)), args.Error(1)
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

func (m *MockProductRepository) GetProductsCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) DeductProductStocks(ctx context.Context, deductions []productRepo.StockDeduction) error {
	args := m.Called(ctx, deductions)
	return args.Error(0)
}

func (m *MockProductRepository) AddProductStocks(ctx context.Context, additions []productRepo.StockDeduction) error {
	args := m.Called(ctx, additions)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

type MockFileRepository struct {
	mock.Mock
}

func (m *MockFileRepository) CreateFileMeta(ctx context.Context, fileMeta *database.File) (uuid.UUID, error) {
	args := m.Called(ctx, fileMeta)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockFileRepository) GetFileMetaByID(ctx context.Context, fileID uuid.UUID) (*database.File, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.File), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileRepository) GetAllFilesMetadata(ctx context.Context, page int64, pageSize int64) ([]database.File, int64, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) != nil {
		return args.Get(0).([]database.File), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockFileRepository) UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error) {
	args := m.Called(ctx, objKey, file, size, contentType)
	return args.String(0), args.Error(1)
}

var _ fileRepo.FileRepository = (*MockFileRepository)(nil)

func (m *MockFileRepository) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	args := m.Called(ctx, objKey)
	return args.String(0), args.Error(1)
}

// ────────────────────────────────────────────────────────────
// Test Suite Setup
// ────────────────────────────────────────────────────────────

type OrderHandlerTestSuite struct {
	suite.Suite
	mockOrderRepo   *MockOrderRepository
	mockProductRepo *MockProductRepository
	mockFileRepo    *MockFileRepository
	handler         OrderHandler
	ctx             context.Context
}

func (s *OrderHandlerTestSuite) SetupTest() {
	s.mockOrderRepo = new(MockOrderRepository)
	s.mockProductRepo = new(MockProductRepository)
	s.mockFileRepo = new(MockFileRepository)
	s.handler = NewOrderHandlerImpl(s.mockOrderRepo, s.mockProductRepo, s.mockFileRepo)
	s.ctx = context.Background()
}

// ────────────────────────────────────────────────────────────
// CreateOrder Tests
// ────────────────────────────────────────────────────────────

func (s *OrderHandlerTestSuite) TestCreateOrder() {
	userID := uuid.New()
	prodID1 := uuid.New()
	prodID2 := uuid.New()
	orderID := uuid.New()

	products := []database.Product{
		{
			Base:        database.Base{ID: prodID1},
			Name:        "Product 1",
			Description: "Desc 1",
			Price:       10.50,
			Available:   20,
			ImageObjKey: "key-1",
		},
		{
			Base:        database.Base{ID: prodID2},
			Name:        "Product 2",
			Description: "Desc 2",
			Price:       20.00,
			Available:   5,
			ImageObjKey: "key-2",
		},
	}

	testCases := []struct {
		name       string
		input      *dto.CreateOrderInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.CreatedOrderOutputDto)
	}{
		{
			name: "Success - order created and stock deducted",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items: []dto.OrderItemRequest{
						{ProductID: prodID1, Quantity: 2},
						{ProductID: prodID2, Quantity: 1},
					},
				},
			},
			setupMocks: func() {
				s.mockProductRepo.On("GetProductsByIDs", s.ctx, mock.MatchedBy(func(ids []uuid.UUID) bool {
					return len(ids) == 2 && ((ids[0] == prodID1 && ids[1] == prodID2) || (ids[0] == prodID2 && ids[1] == prodID1))
				})).Return(products, nil).Once()

				s.mockProductRepo.On("DeductProductStocks", s.ctx, mock.MatchedBy(func(d []productRepo.StockDeduction) bool {
					return len(d) == 2
				})).Return(nil).Once()

				s.mockOrderRepo.On("CreateOrder", s.ctx, mock.MatchedBy(func(o *database.Order) bool {
					return o.UserID == userID && o.TotalPrice == 41.00 && len(o.OrderItems) == 2
				})).Return(orderID, nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.CreatedOrderOutputDto) {
				s.NotNil(out)
				s.Equal(orderID, out.Body.ID)
			},
		},
		{
			name: "Failure - GetProductsByIDs returns database error",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items: []dto.OrderItemRequest{
						{ProductID: prodID1, Quantity: 2},
					},
				},
			},
			setupMocks: func() {
				s.mockProductRepo.On("GetProductsByIDs", s.ctx, []uuid.UUID{prodID1}).
					Return(nil, errors.New("db error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
		{
			name: "Failure - product not found",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items: []dto.OrderItemRequest{
						{ProductID: prodID1, Quantity: 2},
						{ProductID: prodID2, Quantity: 1},
					},
				},
			},
			setupMocks: func() {
				// Only return product 1, simulating product 2 was not found
				s.mockProductRepo.On("GetProductsByIDs", s.ctx, mock.MatchedBy(func(ids []uuid.UUID) bool {
					return len(ids) == 2
				})).Return([]database.Product{products[0]}, nil).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
		{
			name: "Failure - DeductProductStocks returns error (insufficient stock)",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items: []dto.OrderItemRequest{
						{ProductID: prodID1, Quantity: 2},
						{ProductID: prodID2, Quantity: 1},
					},
				},
			},
			setupMocks: func() {
				s.mockProductRepo.On("GetProductsByIDs", s.ctx, mock.MatchedBy(func(ids []uuid.UUID) bool {
					return len(ids) == 2
				})).Return(products, nil).Once()

				s.mockProductRepo.On("DeductProductStocks", s.ctx, mock.MatchedBy(func(d []productRepo.StockDeduction) bool {
					return len(d) == 2
				})).Return(errors.New("insufficient stock")).Once()
			},
			wantErr:   true,
			errStatus: 400,
		},
		{
			name: "Failure - CreateOrder returns error (triggers stock rollback)",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items: []dto.OrderItemRequest{
						{ProductID: prodID1, Quantity: 2},
						{ProductID: prodID2, Quantity: 1},
					},
				},
			},
			setupMocks: func() {
				s.mockProductRepo.On("GetProductsByIDs", s.ctx, mock.MatchedBy(func(ids []uuid.UUID) bool {
					return len(ids) == 2
				})).Return(products, nil).Once()

				s.mockProductRepo.On("DeductProductStocks", s.ctx, mock.MatchedBy(func(d []productRepo.StockDeduction) bool {
					return len(d) == 2
				})).Return(nil).Once()

				s.mockOrderRepo.On("CreateOrder", s.ctx, mock.Anything).
					Return(uuid.Nil, errors.New("db insert fail")).Once()

				// Stock should be rolled back/reverted
				s.mockProductRepo.On("AddProductStocks", s.ctx, mock.MatchedBy(func(add []productRepo.StockDeduction) bool {
					return len(add) == 2
				})).Return(nil).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
		{
			name: "Failure - empty items list",
			input: &dto.CreateOrderInputDto{
				Body: dto.CreateOrderInputDtoBody{
					UserID: userID,
					Items:  []dto.OrderItemRequest{},
				},
			},
			setupMocks: func() {},
			wantErr:    true,
			errStatus:  400,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // Reset mock state
			s.ctx = context.WithValue(s.ctx, "userID", tc.input.Body.UserID.String())
			tc.setupMocks()

			out, err := s.handler.CreateOrder(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected error to be huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}
			s.mockOrderRepo.AssertExpectations(s.T())
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// GetOrders Tests
// ────────────────────────────────────────────────────────────

func (s *OrderHandlerTestSuite) TestGetOrders() {
	userID := uuid.New()
	orderID1 := uuid.New()
	prodID := uuid.New()
	orderItemID := uuid.New()

	orders := []database.Order{
		{
			Base:        database.Base{ID: orderID1, CreatedAt: time.Now()},
			UserID:      userID,
			TotalPrice:  100.00,
			OrderStatus: database.Pending,
			OrderItems: []database.OrderItem{
				{
					Base:            database.Base{ID: orderItemID},
					ProductID:       prodID,
					PriceAtPurchase: 50.00,
					Quantity:        2,
					Product: database.Product{
						Base:        database.Base{ID: prodID},
						Name:        "Sweater",
						ImageObjKey: "sweater.png",
					},
				},
			},
		},
	}

	testCases := []struct {
		name       string
		input      *dto.GetOrdersInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.GetOrdersOutputDto)
	}{
		{
			name: "Success - retrieve orders with date_asc",
			input: &dto.GetOrdersInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
				Status:     "pending",
				Order:      "date_asc",
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrdersByUserID", s.ctx, userID, int64(1), int64(10), "pending", "created_at ASC").
					Return(orders, int64(1), nil).Once()

				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "sweater.png").
					Return("https://presigned.url/sweater.png", nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetOrdersOutputDto) {
				s.NotNil(out)
				s.Equal(int64(1), out.Body.Total)
				s.Len(out.Body.Orders, 1)
				s.Equal(orderID1, out.Body.Orders[0].ID)
				s.Len(out.Body.Orders[0].OrderItems, 1)
				s.Equal("https://presigned.url/sweater.png", out.Body.Orders[0].OrderItems[0].Product.ImageURL)
			},
		},
		{
			name: "Success - retrieve orders with default date_desc",
			input: &dto.GetOrdersInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
				Status:     "",
				Order:      "",
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrdersByUserID", s.ctx, userID, int64(1), int64(10), "", "created_at DESC").
					Return(orders, int64(1), nil).Once()

				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "sweater.png").
					Return("https://presigned.url/sweater.png", nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetOrdersOutputDto) {
				s.NotNil(out)
				s.Equal(int64(1), out.Body.Total)
			},
		},
		{
			name: "Success - retrieve orders but presign url fails (sets URL empty, does not fail order query)",
			input: &dto.GetOrdersInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
				Status:     "",
				Order:      "",
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrdersByUserID", s.ctx, userID, int64(1), int64(10), "", "created_at DESC").
					Return(orders, int64(1), nil).Once()

				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "sweater.png").
					Return("", errors.New("presign failed")).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetOrdersOutputDto) {
				s.NotNil(out)
				s.Len(out.Body.Orders, 1)
				s.Equal("", out.Body.Orders[0].OrderItems[0].Product.ImageURL)
			},
		},
		{
			name: "Failure - GetOrdersByUserID returns database error",
			input: &dto.GetOrdersInputDto{
				UserID:     userID,
				PageNumber: 1,
				PageSize:   10,
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrdersByUserID", s.ctx, userID, int64(1), int64(10), "", "created_at DESC").
					Return(nil, int64(0), errors.New("db query error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.GetOrders(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr))
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}
			s.mockOrderRepo.AssertExpectations(s.T())
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// GetOrderById Tests
// ────────────────────────────────────────────────────────────

func (s *OrderHandlerTestSuite) TestGetOrderById() {
	orderID := uuid.New()
	userID := uuid.New()
	prodID := uuid.New()

	order := &database.Order{
		Base:        database.Base{ID: orderID, CreatedAt: time.Now()},
		UserID:      userID,
		TotalPrice:  75.50,
		OrderStatus: database.Paid,
		OrderItems: []database.OrderItem{
			{
				ProductID:       prodID,
				PriceAtPurchase: 75.50,
				Quantity:        1,
				Product: database.Product{
					Base:        database.Base{ID: prodID},
					Name:        "Shoes",
					ImageObjKey: "shoes.jpg",
				},
			},
		},
	}

	testCases := []struct {
		name       string
		input      *dto.GetOrderByIdInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.GetOrderByIdOutputDto)
	}{
		{
			name: "Success - returns order by ID",
			input: &dto.GetOrderByIdInputDto{
				ID: orderID,
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(order, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "shoes.jpg").Return("http://url/shoes.jpg", nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetOrderByIdOutputDto) {
				s.NotNil(out)
				s.Equal(orderID, out.Body.ID)
				s.Equal(userID, out.Body.UserID)
				s.Equal(float32(75.50), out.Body.TotalPrice)
				s.Equal("http://url/shoes.jpg", out.Body.OrderItems[0].Product.ImageURL)
			},
		},
		{
			name: "Failure - order not found",
			input: &dto.GetOrderByIdInputDto{
				ID: orderID,
			},
			setupMocks: func() {
				s.mockOrderRepo.On("GetOrderByID", s.ctx, orderID).Return(nil, errors.New("not found")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.ctx = context.WithValue(s.ctx, "userID", userID.String())
			tc.setupMocks()

			out, err := s.handler.GetOrderById(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr))
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}
			s.mockOrderRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// UpdateOrderStatus Tests
// ────────────────────────────────────────────────────────────

func (s *OrderHandlerTestSuite) TestUpdateOrderStatus() {
	orderID := uuid.New()

	testCases := []struct {
		name       string
		input      *dto.UpdateOrderStatusInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.UpdateOrderStatusOutputDto)
	}{
		{
			name: "Success - update status to completed",
			input: &dto.UpdateOrderStatusInputDto{
				ID: orderID,
				Body: dto.UpdateOrderStatusInputDtoBody{
					Status: database.Completed,
				},
			},
			setupMocks: func() {
				s.mockOrderRepo.On("UpdateOrderStatus", s.ctx, orderID, database.Completed).Return(nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.UpdateOrderStatusOutputDto) {
				s.NotNil(out)
				s.True(out.Body.Success)
			},
		},
		{
			name: "Failure - repo returns error",
			input: &dto.UpdateOrderStatusInputDto{
				ID: orderID,
				Body: dto.UpdateOrderStatusInputDtoBody{
					Status: database.Failed,
				},
			},
			setupMocks: func() {
				s.mockOrderRepo.On("UpdateOrderStatus", s.ctx, orderID, database.Failed).Return(errors.New("db error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.UpdateOrderStatus(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr))
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}
			s.mockOrderRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Suite Runner
// ────────────────────────────────────────────────────────────

func TestOrderHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlerTestSuite))
}

func (s *OrderHandlerTestSuite) TestGetOrdersCount_Success() {
	s.mockOrderRepo.On("GetOrdersCount", mock.Anything).Return(85, nil).Once()

	res, err := s.handler.GetOrdersCount(s.ctx, &struct{}{})
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int64(85), res.Body.Count)
}

func (s *OrderHandlerTestSuite) TestGetOrdersCount_Error() {
	s.mockOrderRepo.On("GetOrdersCount", mock.Anything).Return(0, errors.New("db error")).Once()

	res, err := s.handler.GetOrdersCount(s.ctx, &struct{}{})
	s.Error(err)
	s.Nil(res)
}
