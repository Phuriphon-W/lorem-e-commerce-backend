package handler

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"reflect"
	"testing"
	"unsafe"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/product/dto"
	"lorem-backend/internal/modules/product/repository"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Dummy multipart.File implementation for testing
type dummyFile struct {
	*bytes.Reader
}

func (d dummyFile) Close() error {
	return nil
}

// MockProductRepository is a mock of repository.ProductRepository
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

func (m *MockProductRepository) DeductProductStocks(ctx context.Context, deductions []repository.StockDeduction) error {
	args := m.Called(ctx, deductions)
	return args.Error(0)
}

func (m *MockProductRepository) AddProductStocks(ctx context.Context, additions []repository.StockDeduction) error {
	args := m.Called(ctx, additions)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

// MockFileRepository is a mock of file.FileRepository
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

// Suite Definition
type ProductHandlerTestSuite struct {
	suite.Suite
	mockProductRepo *MockProductRepository
	mockFileRepo    *MockFileRepository
	handler         ProductHandler
	ctx             context.Context
}

func (s *ProductHandlerTestSuite) SetupTest() {
	s.mockProductRepo = new(MockProductRepository)
	s.mockFileRepo = new(MockFileRepository)
	s.handler = NewProductHandlerImpl(s.mockProductRepo, s.mockFileRepo)
	s.ctx = context.Background()
}

func createCreateProductInput(
	name string,
	description string,
	price float32,
	available uint,
	categoryID uuid.UUID,
) *dto.CreateProductInputDto {
	type productMultipartFormFields = struct {
		Name        string        `form:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
		Description string        `form:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
		Price       float32       `form:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
		Available   uint          `form:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
		ImageFile   huma.FormFile `form:"image_file" required:"true" doc:"Image file of the product"`
		CategoryId  uuid.UUID     `form:"categoryId" required:"true" doc:"ID of the product category" example:"fdc93985-b4fd-40d3-ad6c-3fb94c6ec8c7"`
	}

	formFile := huma.FormFile{
		File:        dummyFile{Reader: bytes.NewReader([]byte("fake-image"))},
		ContentType: "image/png",
		IsSet:       true,
		Size:        10,
		Filename:    "test.png",
	}

	val := &productMultipartFormFields{
		Name:        name,
		Description: description,
		Price:       price,
		Available:   available,
		ImageFile:   formFile,
		CategoryId:  categoryID,
	}

	var res dto.CreateProductInputDto
	v := reflect.ValueOf(&res.RawBody).Elem()
	f := v.FieldByName("data")
	ptr := unsafe.Pointer(f.UnsafeAddr())
	reflect.NewAt(f.Type(), ptr).Elem().Set(reflect.ValueOf(val))

	return &res
}

func (s *ProductHandlerTestSuite) TestCreateProduct() {
	catID := uuid.New()
	productID := uuid.New()
	input := createCreateProductInput("Awesome T-Shirt", "A nice shirt", 29.99, 100, catID)

	testCases := []struct {
		name          string
		setupMock     func()
		expectedError bool
		verify        func(res *dto.CreatedProductOutputDto)
	}{
		{
			name: "Success - creates product",
			setupMock: func() {
				s.mockFileRepo.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, int64(10), "image/png").
					Return("product-images/test-key.png", nil).Once()

				s.mockProductRepo.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p *database.Product) bool {
					return p.Name == "Awesome T-Shirt" && p.Price == 29.99 && p.CategoryID == catID && p.ImageObjKey == "product-images/test-key.png"
				})).Return(productID, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.CreatedProductOutputDto) {
				s.NotNil(res)
				s.Equal(productID, res.Body.ID)
			},
		},
		{
			name: "Failure - image upload error",
			setupMock: func() {
				s.mockFileRepo.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, int64(10), "image/png").
					Return("", errors.New("s3 upload failed")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreatedProductOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - DB create error",
			setupMock: func() {
				s.mockFileRepo.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, int64(10), "image/png").
					Return("product-images/test-key.png", nil).Once()

				s.mockProductRepo.On("CreateProduct", mock.Anything, mock.Anything).
					Return(uuid.Nil, errors.New("db error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreatedProductOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.CreateProduct(s.ctx, input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

func (s *ProductHandlerTestSuite) TestGetProducts() {
	catID := uuid.New()
	prodID := uuid.New()
	mockProducts := []database.Product{
		{
			Base: database.Base{
				ID: prodID,
			},
			Name:        "Sweater",
			Description: "Warm sweater",
			Price:       49.99,
			Available:   50,
			ImageObjKey: "images/sweater.png",
			CategoryID:  catID,
			Category: database.Category{
				Base: database.Base{
					ID: catID,
				},
				Name: "Apparel",
			},
		},
	}

	testCases := []struct {
		name          string
		input         *dto.GetProductsInputDto
		setupMock     func(input *dto.GetProductsInputDto)
		expectedError bool
		verify        func(res *dto.GetProductsOutputDto)
	}{
		{
			name: "Success - standard query & pre-signed URLs generated",
			input: &dto.GetProductsInputDto{
				PageNumber: 1,
				PageSize:   10,
				Category:   "Apparel",
				Search:     "Sweater",
				Order:      "price_low",
			},
			setupMock: func(input *dto.GetProductsInputDto) {
				s.mockProductRepo.On("GetProducts", mock.Anything, int64(1), int64(10), "Apparel", "Sweater", "products.price ASC").
					Return(mockProducts, int64(1), nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("https://presigned.url/images/sweater.png", nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetProductsOutputDto) {
				s.NotNil(res)
				s.Len(res.Body.Products, 1)
				s.Equal(prodID, res.Body.Products[0].ID)
				s.Equal("https://presigned.url/images/sweater.png", res.Body.Products[0].ImageURL)
				s.Equal("Apparel", res.Body.Products[0].Category.Name)
			},
		},
		{
			name: "Success - sorting mappings (price_high, name_asc, name_desc, date_asc, default)",
			input: &dto.GetProductsInputDto{
				PageNumber: 1,
				PageSize:   10,
				Order:      "price_high",
			},
			setupMock: func(input *dto.GetProductsInputDto) {
				s.mockProductRepo.On("GetProducts", mock.Anything, int64(1), int64(10), "", "", "products.price DESC").
					Return(mockProducts, int64(1), nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("https://presigned.url/images/sweater.png", nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetProductsOutputDto) {
				s.NotNil(res)
			},
		},
		{
			name: "Success - other order mappings date_asc & default",
			input: &dto.GetProductsInputDto{
				PageNumber: 1,
				PageSize:   10,
				Order:      "date_asc",
			},
			setupMock: func(input *dto.GetProductsInputDto) {
				s.mockProductRepo.On("GetProducts", mock.Anything, int64(1), int64(10), "", "", "products.created_at ASC").
					Return(mockProducts, int64(1), nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("", errors.New("presign url failed")).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetProductsOutputDto) {
				s.NotNil(res)
				s.Equal("", res.Body.Products[0].ImageURL) // gracefully falls back to empty string on presign error
			},
		},
		{
			name: "Failure - product repo error",
			input: &dto.GetProductsInputDto{
				PageNumber: 1,
				PageSize:   10,
			},
			setupMock: func(input *dto.GetProductsInputDto) {
				s.mockProductRepo.On("GetProducts", mock.Anything, int64(1), int64(10), "", "", "products.created_at DESC").
					Return(nil, int64(0), errors.New("db query failed")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetProductsOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(tc.input)

			res, err := s.handler.GetProducts(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

func (s *ProductHandlerTestSuite) TestGetProductById() {
	catID := uuid.New()
	prodID := uuid.New()
	mockProduct := &database.Product{
		Base: database.Base{
			ID: prodID,
		},
		Name:        "Sweater",
		Description: "Warm sweater",
		Price:       49.99,
		Available:   50,
		ImageObjKey: "images/sweater.png",
		CategoryID:  catID,
		Category: database.Category{
			Base: database.Base{
				ID: catID,
			},
			Name: "Apparel",
		},
	}

	testCases := []struct {
		name          string
		input         *dto.GetProductByIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.GetProductByIdOutputDto)
	}{
		{
			name:  "Success - product found and URL pre-signed",
			input: &dto.GetProductByIdInputDto{ID: prodID},
			setupMock: func() {
				s.mockProductRepo.On("GetProductByID", mock.Anything, prodID).Return(mockProduct, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("https://presigned.url/images/sweater.png", nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetProductByIdOutputDto) {
				s.NotNil(res)
				s.Equal(prodID, res.Body.ID)
				s.Equal("https://presigned.url/images/sweater.png", res.Body.ImageURL)
				s.Equal("Sweater", res.Body.Name)
			},
		},
		{
			name:  "Success - URL pre-sign error returns product with empty image URL",
			input: &dto.GetProductByIdInputDto{ID: prodID},
			setupMock: func() {
				s.mockProductRepo.On("GetProductByID", mock.Anything, prodID).Return(mockProduct, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", mock.Anything, "images/sweater.png").
					Return("", errors.New("failed presign")).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetProductByIdOutputDto) {
				s.NotNil(res)
				s.Equal("", res.Body.ImageURL)
			},
		},
		{
			name:  "Failure - product not found in database",
			input: &dto.GetProductByIdInputDto{ID: prodID},
			setupMock: func() {
				s.mockProductRepo.On("GetProductByID", mock.Anything, prodID).Return(nil, errors.New("not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetProductByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetProductById(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockProductRepo.AssertExpectations(s.T())
			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

func (s *ProductHandlerTestSuite) TestDeleteProductById() {
	prodID := uuid.New()

	testCases := []struct {
		name          string
		input         *dto.DeleteProductByIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.DeleteProductByIdOutputDto)
	}{
		{
			name:  "Success - product deleted successfully",
			input: &dto.DeleteProductByIdInputDto{ID: prodID},
			setupMock: func() {
				s.mockProductRepo.On("DeleteProductByID", mock.Anything, prodID).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.DeleteProductByIdOutputDto) {
				s.NotNil(res)
				s.Equal("Product deleted successfully", res.Body.Message)
			},
		},
		{
			name:  "Failure - product delete database error",
			input: &dto.DeleteProductByIdInputDto{ID: prodID},
			setupMock: func() {
				s.mockProductRepo.On("DeleteProductByID", mock.Anything, prodID).Return(errors.New("db error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.DeleteProductByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.DeleteProductById(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockProductRepo.AssertExpectations(s.T())
		})
	}
}

func TestProductHandlerSuite(t *testing.T) {
	suite.Run(t, new(ProductHandlerTestSuite))
}
