package handler

import (
	"bytes"
	"context"
	"errors"
	fileRepo "lorem-backend/internal/modules/file/repository"
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

// Suite Definition
type ProductHandlerTestSuite struct {
	suite.Suite
	mockProductRepo *repository.MockProductRepository
	mockFileRepo    *fileRepo.MockFileRepository
	handler         ProductHandler
	ctx             context.Context
}

func (s *ProductHandlerTestSuite) SetupTest() {
	s.mockProductRepo = repository.NewMockProductRepository(s.T())
	s.mockFileRepo = fileRepo.NewMockFileRepository(s.T())
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
		Description string        `form:"description" required:"true" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
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

func (s *ProductHandlerTestSuite) TestGetProductsCount_Success() {
	s.mockProductRepo.On("GetProductsCount", mock.Anything).Return(int64(150), nil).Once()

	res, err := s.handler.GetProductsCount(s.ctx, &struct{}{})
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int64(150), res.Body.Count)
}

func (s *ProductHandlerTestSuite) TestGetProductsCount_Error() {
	s.mockProductRepo.On("GetProductsCount", mock.Anything).Return(int64(0), errors.New("db error")).Once()

	res, err := s.handler.GetProductsCount(s.ctx, &struct{}{})
	s.Error(err)
	s.Nil(res)
}
