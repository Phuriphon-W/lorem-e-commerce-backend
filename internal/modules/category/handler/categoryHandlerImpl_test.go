package handler

import (
	"context"
	"errors"
	"testing"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/category/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockCategoryRepository is an inline testify/mock implementation of repository.CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) CreateCategory(ctx context.Context, category *database.Category) (uuid.UUID, error) {
	args := m.Called(ctx, category)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockCategoryRepository) GetCategoryByID(ctx context.Context, catID uuid.UUID) (*database.Category, error) {
	args := m.Called(ctx, catID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.Category), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCategoryRepository) GetCategories(ctx context.Context) ([]database.Category, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]database.Category), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCategoryRepository) UpdateCategoryByID(ctx context.Context, catID uuid.UUID, name string) error {
	args := m.Called(ctx, catID, name)
	return args.Error(0)
}

func (m *MockCategoryRepository) DeleteCategoryByID(ctx context.Context, catID uuid.UUID) error {
	args := m.Called(ctx, catID)
	return args.Error(0)
}

// Suite Definition
type CategoryHandlerTestSuite struct {
	suite.Suite
	mockRepo *MockCategoryRepository
	handler  CategoryHandler
	ctx      context.Context
}

func (s *CategoryHandlerTestSuite) SetupTest() {
	s.mockRepo = new(MockCategoryRepository)
	s.handler = NewCategoryHandlerImpl(s.mockRepo)
	s.ctx = context.Background()
}

// ────────────────────────────────────────────────────────────
// TestCreateCategory
// ────────────────────────────────────────────────────────────

func (s *CategoryHandlerTestSuite) TestCreateCategory() {
	catID := uuid.New()

	testCases := []struct {
		name          string
		input         *dto.CreateCategoryInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.CreateCategoryOutputDto)
	}{
		{
			name: "Success - category created",
			input: &dto.CreateCategoryInputDto{
				Body: struct {
					Name string `json:"name" required:"true" minLength:"1" doc:"Category name" example:"Apparel"`
				}{Name: "Apparel"},
			},
			setupMock: func() {
				s.mockRepo.On("CreateCategory", mock.Anything, mock.MatchedBy(func(c *database.Category) bool {
					return c.Name == "Apparel"
				})).Return(catID, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.CreateCategoryOutputDto) {
				s.NotNil(res)
				s.Equal(catID, res.Body.ID)
			},
		},
		{
			name: "Failure - repository error",
			input: &dto.CreateCategoryInputDto{
				Body: struct {
					Name string `json:"name" required:"true" minLength:"1" doc:"Category name" example:"Apparel"`
				}{Name: "Apparel"},
			},
			setupMock: func() {
				s.mockRepo.On("CreateCategory", mock.Anything, mock.Anything).
					Return(uuid.Nil, errors.New("db error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.CreateCategoryOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.CreateCategory(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetCategoryById
// ────────────────────────────────────────────────────────────

func (s *CategoryHandlerTestSuite) TestGetCategoryById() {
	catID := uuid.New()
	mockCategory := &database.Category{
		Base: database.Base{ID: catID},
		Name: "Apparel",
	}

	testCases := []struct {
		name          string
		input         *dto.GetCategoryByIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.GetCategoryByIdOutputDto)
	}{
		{
			name:  "Success - category found",
			input: &dto.GetCategoryByIdInputDto{ID: catID},
			setupMock: func() {
				s.mockRepo.On("GetCategoryByID", mock.Anything, catID).Return(mockCategory, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCategoryByIdOutputDto) {
				s.NotNil(res)
				s.Equal(catID, res.Body.ID)
				s.Equal("Apparel", res.Body.Name)
			},
		},
		{
			name:  "Failure - category not found",
			input: &dto.GetCategoryByIdInputDto{ID: catID},
			setupMock: func() {
				s.mockRepo.On("GetCategoryByID", mock.Anything, catID).Return(nil, errors.New("not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetCategoryByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetCategoryById(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetCategories
// ────────────────────────────────────────────────────────────

func (s *CategoryHandlerTestSuite) TestGetCategories() {
	cat1ID := uuid.New()
	cat2ID := uuid.New()
	mockCategories := []database.Category{
		{Base: database.Base{ID: cat1ID}, Name: "Apparel"},
		{Base: database.Base{ID: cat2ID}, Name: "Electronics"},
	}

	testCases := []struct {
		name          string
		setupMock     func()
		expectedError bool
		verify        func(res *dto.GetCategoriesOutputDto)
	}{
		{
			name: "Success - returns all categories",
			setupMock: func() {
				s.mockRepo.On("GetCategories", mock.Anything).Return(mockCategories, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCategoriesOutputDto) {
				s.NotNil(res)
				s.Len(res.Body, 2)
				s.Equal(cat1ID, res.Body[0].ID)
				s.Equal("Apparel", res.Body[0].Name)
				s.Equal(cat2ID, res.Body[1].ID)
				s.Equal("Electronics", res.Body[1].Name)
			},
		},
		{
			name: "Success - returns empty list",
			setupMock: func() {
				s.mockRepo.On("GetCategories", mock.Anything).Return([]database.Category{}, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetCategoriesOutputDto) {
				s.NotNil(res)
				s.Len(res.Body, 0)
			},
		},
		{
			name: "Failure - repository error",
			setupMock: func() {
				s.mockRepo.On("GetCategories", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetCategoriesOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetCategories(s.ctx, nil)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestUpdateCategory
// ────────────────────────────────────────────────────────────

func (s *CategoryHandlerTestSuite) TestUpdateCategory() {
	catID := uuid.New()

	testCases := []struct {
		name          string
		input         *dto.UpdateCategoryByIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.UpdateCategoryByIdOutputDto)
	}{
		{
			name: "Success - category updated",
			input: &dto.UpdateCategoryByIdInputDto{
				ID: catID,
				Body: struct {
					Name string `json:"name" required:"true" minLength:"1" doc:"Category name" example:"Apparel"`
				}{Name: "Updated Apparel"},
			},
			setupMock: func() {
				s.mockRepo.On("UpdateCategoryByID", mock.Anything, catID, "Updated Apparel").Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.UpdateCategoryByIdOutputDto) {
				s.NotNil(res)
				s.Equal(catID, res.Body.ID)
				s.Equal("Updated Apparel", res.Body.Name)
			},
		},
		{
			name: "Failure - repository error (category not found)",
			input: &dto.UpdateCategoryByIdInputDto{
				ID: catID,
				Body: struct {
					Name string `json:"name" required:"true" minLength:"1" doc:"Category name" example:"Apparel"`
				}{Name: "Updated Apparel"},
			},
			setupMock: func() {
				s.mockRepo.On("UpdateCategoryByID", mock.Anything, catID, "Updated Apparel").
					Return(errors.New("record not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.UpdateCategoryByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.UpdateCategory(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestDeleteCategory
// ────────────────────────────────────────────────────────────

func (s *CategoryHandlerTestSuite) TestDeleteCategory() {
	catID := uuid.New()

	testCases := []struct {
		name          string
		input         *dto.DeleteCategoryByIdInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.DeleteCategoryByIdOutputDto)
	}{
		{
			name:  "Success - category deleted",
			input: &dto.DeleteCategoryByIdInputDto{ID: catID},
			setupMock: func() {
				s.mockRepo.On("DeleteCategoryByID", mock.Anything, catID).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.DeleteCategoryByIdOutputDto) {
				s.NotNil(res)
				s.Equal("Category deleted successfully", res.Body.Message)
			},
		},
		{
			name:  "Failure - repository error (category not found)",
			input: &dto.DeleteCategoryByIdInputDto{ID: catID},
			setupMock: func() {
				s.mockRepo.On("DeleteCategoryByID", mock.Anything, catID).
					Return(errors.New("record not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.DeleteCategoryByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.DeleteCategory(s.ctx, tc.input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

func TestCategoryHandlerSuite(t *testing.T) {
	suite.Run(t, new(CategoryHandlerTestSuite))
}
