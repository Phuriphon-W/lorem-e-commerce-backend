package handler

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/cart/dto"
	productRepo "lorem-backend/internal/modules/product/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ────────────────────────────────────────────────────────────
// Mock Implementations
// ────────────────────────────────────────────────────────────

// MockCartRepository is an inline testify/mock implementation of repository.CartRepository
type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) GetCartByUserId(ctx context.Context, userId uuid.UUID) (*database.Cart, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) != nil {
		return args.Get(0).(*database.Cart), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCartRepository) CreateCartItem(ctx context.Context, cartItem *database.CartItem) (uuid.UUID, error) {
	args := m.Called(ctx, cartItem)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockCartRepository) GetCartItem(ctx context.Context, cartId, productId uuid.UUID) (*database.CartItem, error) {
	args := m.Called(ctx, cartId, productId)
	if args.Get(0) != nil {
		return args.Get(0).(*database.CartItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCartRepository) EditCartItem(ctx context.Context, cartId uuid.UUID, productId uuid.UUID, quantity uint) error {
	args := m.Called(ctx, cartId, productId, quantity)
	return args.Error(0)
}

func (m *MockCartRepository) RemoveCartItems(ctx context.Context, cartId uuid.UUID, productIds []uuid.UUID) error {
	args := m.Called(ctx, cartId, productIds)
	return args.Error(0)
}

// MockProductRepository is an inline testify/mock implementation of productRepo.ProductRepository
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

// MockFileRepository is an inline testify/mock implementation of fileRepo.FileRepository
// (which embeds ObjectStorage and adds DB-backed metadata methods)
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

func (m *MockFileRepository) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	args := m.Called(ctx, objKey)
	return args.String(0), args.Error(1)
}

// ────────────────────────────────────────────────────────────
// Suite Definition
// ────────────────────────────────────────────────────────────

type CartHandlerTestSuite struct {
	suite.Suite
	mockCartRepo    *MockCartRepository
	mockProductRepo *MockProductRepository
	mockFileRepo    *MockFileRepository
	handler         CartHandler
	ctx             context.Context
}

func (s *CartHandlerTestSuite) SetupTest() {
	s.mockCartRepo = new(MockCartRepository)
	s.mockProductRepo = new(MockProductRepository)
	s.mockFileRepo = new(MockFileRepository)
	s.handler = NewCartHandler(s.mockCartRepo, s.mockFileRepo, s.mockProductRepo)
	s.ctx = context.Background()
}

// ────────────────────────────────────────────────────────────
// TestGetCartByUserId
// ────────────────────────────────────────────────────────────

func (s *CartHandlerTestSuite) TestGetCartByUserId() {
	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()
	catID := uuid.New()

	mockCart := &database.Cart{
		Base:   database.Base{ID: cartID},
		UserID: userID,
		CartItems: []database.CartItem{
			{
				Base:      database.Base{ID: uuid.New()},
				CartID:    cartID,
				ProductID: productID,
				Quantity:  2,
				Product: database.Product{
					Base:        database.Base{ID: productID},
					Name:        "Sweater",
					Description: "Warm sweater",
					Price:       49.99,
					ImageObjKey: "images/sweater.png",
					CategoryID:  catID,
					Category: database.Category{
						Base: database.Base{ID: catID},
						Name: "Apparel",
					},
				},
			},
		},
	}

	testCases := []struct {
		name          string
		input         *dto.GetCartByUserIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.GetCartByUserIdOutputDto)
	}{
		{
			name:  "Success - returns cart with items, presigned URL and stock",
			input: &dto.GetCartByUserIdInputDto{ID: userID},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("https://presigned.url/images/sweater.png", nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCartByUserIdOutputDto) {
				s.NotNil(res)
				s.Equal(cartID, res.Body.CartID)
				s.Len(res.Body.CartItems, 1)
				s.Equal(productID, res.Body.CartItems[0].ProductID)
				s.Equal("Sweater", res.Body.CartItems[0].Name)
				s.Equal(float32(49.99), res.Body.CartItems[0].Price)
				s.Equal(uint(2), res.Body.CartItems[0].Quantity)
				s.Equal(uint(10), res.Body.CartItems[0].Available)
				s.Equal("https://presigned.url/images/sweater.png", res.Body.CartItems[0].ImageURL)
				s.Equal("Apparel", res.Body.CartItems[0].Category.Name)
			},
		},
		{
			name:  "Success - presign URL error degrades gracefully (empty image URL)",
			input: &dto.GetCartByUserIdInputDto{ID: userID},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("", errors.New("s3 presign failed")).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(5), nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCartByUserIdOutputDto) {
				s.NotNil(res)
				s.Equal("", res.Body.CartItems[0].ImageURL) // falls back to empty string
				s.Equal(uint(5), res.Body.CartItems[0].Available)
			},
		},
		{
			name:  "Success - GetProductStock error degrades gracefully (Available = 0)",
			input: &dto.GetCartByUserIdInputDto{ID: userID},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("https://presigned.url/images/sweater.png", nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(0), errors.New("product not found")).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCartByUserIdOutputDto) {
				s.NotNil(res)
				s.Equal(uint(0), res.Body.CartItems[0].Available) // falls back to 0
			},
		},
		{
			name:  "Success - empty cart (no cart items)",
			input: &dto.GetCartByUserIdInputDto{ID: userID},
			setupMock: func() {
				emptyCart := &database.Cart{
					Base:      database.Base{ID: cartID},
					UserID:    userID,
					CartItems: []database.CartItem{},
				}
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(emptyCart, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCartByUserIdOutputDto) {
				s.NotNil(res)
				s.Equal(cartID, res.Body.CartID)
				s.Len(res.Body.CartItems, 0)
			},
		},
		{
			name:  "Failure - cart not found",
			input: &dto.GetCartByUserIdInputDto{ID: userID},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).
					Return(nil, errors.New("record not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetCartByUserIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetCartByUserId(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockCartRepo.AssertExpectations(s.T())
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestCreateCartItem
// ────────────────────────────────────────────────────────────

func (s *CartHandlerTestSuite) TestCreateCartItem() {
	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()
	cartItemID := uuid.New()

	mockCart := &database.Cart{
		Base:   database.Base{ID: cartID},
		UserID: userID,
	}

	testCases := []struct {
		name          string
		input         *dto.CreateCartItemInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.CreateCartItemOutputDto)
	}{
		{
			name: "Success - new item created (item not in cart)",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 2},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
				s.mockCartRepo.On("GetCartItem", mock.Anything, cartID, productID).Return(nil, errors.New("not found")).Once()
				s.mockCartRepo.On("CreateCartItem", mock.Anything, mock.MatchedBy(func(ci *database.CartItem) bool {
					return ci.CartID == cartID && ci.ProductID == productID && ci.Quantity == 2
				})).Return(cartItemID, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.NotNil(res)
				s.Equal(cartItemID, res.Body.CartItemID)
			},
		},
		{
			name: "Success - item already in cart (dedup: merges quantity via EditCartItem)",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 3},
			},
			setupMock: func() {
				existingItem := &database.CartItem{
					Base:      database.Base{ID: cartItemID},
					CartID:    cartID,
					ProductID: productID,
					Quantity:  2, // already 2 in cart
				}
				// First GetCartByUserId for CreateCartItem
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
				s.mockCartRepo.On("GetCartItem", mock.Anything, cartID, productID).Return(existingItem, nil).Once()
				// The handler calls EditCartItem internally, which calls GetCartByUserId again
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				// newTotalQuantity = 2 + 3 = 5
				s.mockCartRepo.On("EditCartItem", mock.Anything, cartID, productID, uint(5)).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.NotNil(res)
				s.Equal(cartItemID, res.Body.CartItemID)
			},
		},
		{
			name: "Failure - dedup exceeds stock",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 5},
			},
			setupMock: func() {
				existingItem := &database.CartItem{
					Base:      database.Base{ID: cartItemID},
					CartID:    cartID,
					ProductID: productID,
					Quantity:  8, // already 8, adding 5 = 13 > stock(10)
				}
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
				s.mockCartRepo.On("GetCartItem", mock.Anything, cartID, productID).Return(existingItem, nil).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - new item exceeds stock",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 20},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
				s.mockCartRepo.On("GetCartItem", mock.Anything, cartID, productID).Return(nil, errors.New("not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - cart not found",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 1},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).
					Return(nil, errors.New("cart not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - product not found (stock fetch error)",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 1},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).
					Return(uint(0), errors.New("product not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - repo CreateCartItem error",
			input: &dto.CreateCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
				}{ProductID: productID, Quantity: 2},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockProductRepo.On("GetProductStock", mock.Anything, productID).Return(uint(10), nil).Once()
				s.mockCartRepo.On("GetCartItem", mock.Anything, cartID, productID).Return(nil, errors.New("not found")).Once()
				s.mockCartRepo.On("CreateCartItem", mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("db insert error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCartItemOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.CreateCartItem(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockCartRepo.AssertExpectations(s.T())
			s.mockProductRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestEditCartItem
// ────────────────────────────────────────────────────────────

func (s *CartHandlerTestSuite) TestEditCartItem() {
	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()

	mockCart := &database.Cart{
		Base:   database.Base{ID: cartID},
		UserID: userID,
	}

	testCases := []struct {
		name          string
		input         *dto.EditCartItemInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.EditCartItemOutputDto)
	}{
		{
			name: "Success - cart item updated",
			input: &dto.EditCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" doc:"New exact quantity (must be at least 1)"`
				}{ProductID: productID, Quantity: 5},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockCartRepo.On("EditCartItem", mock.Anything, cartID, productID, uint(5)).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.EditCartItemOutputDto) {
				s.NotNil(res)
				s.Equal("Cart item updated successfully", res.Body.Message)
			},
		},
		{
			name: "Failure - cart not found",
			input: &dto.EditCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" doc:"New exact quantity (must be at least 1)"`
				}{ProductID: productID, Quantity: 5},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).
					Return(nil, errors.New("cart not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.EditCartItemOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - repo EditCartItem error",
			input: &dto.EditCartItemInputDto{
				UserID: userID,
				Body: struct {
					ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
					Quantity  uint      `json:"quantity" required:"true" minimum:"1" doc:"New exact quantity (must be at least 1)"`
				}{ProductID: productID, Quantity: 5},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockCartRepo.On("EditCartItem", mock.Anything, cartID, productID, uint(5)).
					Return(errors.New("db update error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.EditCartItemOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.EditCartItem(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockCartRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestDeleteCartItems
// ────────────────────────────────────────────────────────────

func (s *CartHandlerTestSuite) TestDeleteCartItems() {
	userID := uuid.New()
	cartID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	mockCart := &database.Cart{
		Base:   database.Base{ID: cartID},
		UserID: userID,
	}

	testCases := []struct {
		name          string
		input         *dto.DeleteCartItemsInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.DeleteCartItemsOutputDto)
	}{
		{
			name: "Success - cart items deleted",
			input: &dto.DeleteCartItemsInputDto{
				UserID: userID,
				Body: struct {
					ProductIDs []uuid.UUID `json:"productIds" required:"true" minItems:"1" doc:"List of Product IDs to remove"`
				}{ProductIDs: []uuid.UUID{productID1, productID2}},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockCartRepo.On("RemoveCartItems", mock.Anything, cartID, []uuid.UUID{productID1, productID2}).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.DeleteCartItemsOutputDto) {
				s.NotNil(res)
				s.Equal("Cart item(s) deleted successfully", res.Body.Message)
			},
		},
		{
			name: "Failure - cart not found",
			input: &dto.DeleteCartItemsInputDto{
				UserID: userID,
				Body: struct {
					ProductIDs []uuid.UUID `json:"productIds" required:"true" minItems:"1" doc:"List of Product IDs to remove"`
				}{ProductIDs: []uuid.UUID{productID1}},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).
					Return(nil, errors.New("cart not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.DeleteCartItemsOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - repo RemoveCartItems error",
			input: &dto.DeleteCartItemsInputDto{
				UserID: userID,
				Body: struct {
					ProductIDs []uuid.UUID `json:"productIds" required:"true" minItems:"1" doc:"List of Product IDs to remove"`
				}{ProductIDs: []uuid.UUID{productID1}},
			},
			setupMock: func() {
				s.mockCartRepo.On("GetCartByUserId", mock.Anything, userID).Return(mockCart, nil).Once()
				s.mockCartRepo.On("RemoveCartItems", mock.Anything, cartID, []uuid.UUID{productID1}).
					Return(errors.New("db delete error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.DeleteCartItemsOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.DeleteCartItems(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockCartRepo.AssertExpectations(s.T())
		})
	}
}

func TestCartHandlerSuite(t *testing.T) {
	suite.Run(t, new(CartHandlerTestSuite))
}
