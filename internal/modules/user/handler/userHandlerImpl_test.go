package handler

import (
	"context"
	"errors"
	"testing"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/user/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockUserRepository is a mock type for the UserRepository type
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUsers(ctx context.Context, page, pageSize int64, search, order string) ([]database.User, int64, error) {
	args := m.Called(ctx, page, pageSize, search, order)
	if args.Get(0) != nil {
		return args.Get(0).([]database.User), int64(args.Int(1)), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*database.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *database.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUsersCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return int64(args.Int(0)), args.Error(1)
}

type UserHandlerTestSuite struct {
	suite.Suite
	mockRepo *MockUserRepository
	handler  UserHandler
	ctx      context.Context
}

func (s *UserHandlerTestSuite) SetupTest() {
	s.mockRepo = new(MockUserRepository)
	s.handler = NewUserHandlerImpl(s.mockRepo)
	s.ctx = context.Background()
}

func (s *UserHandlerTestSuite) TestGetUserById() {
	userID := uuid.New()
	expectedUser := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
	}

	testCases := []struct {
		name          string
		setupMock     func()
		input         *dto.GetUserByIdInputDto
		expectedError bool
		verify        func(res *dto.GetUserByIdOutputDto)
	}{
		{
			name: "Success - returns user when ID exists",
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil).Once()
			},
			input:         &dto.GetUserByIdInputDto{ID: userID},
			expectedError: false,
			verify: func(res *dto.GetUserByIdOutputDto) {
				s.Equal(userID, res.Body.ID)
				s.Equal("johndoe", res.Body.Username)
			},
		},
		{
			name: "Failure - User Not Found",
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("not found")).Once()
			},
			input:         &dto.GetUserByIdInputDto{ID: userID},
			expectedError: true,
			verify: func(res *dto.GetUserByIdOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetUserById(s.ctx, tc.input)

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

func (s *UserHandlerTestSuite) TestGetMe() {
	userID := uuid.New()
	expectedUser := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}

	testCases := []struct {
		name          string
		ctx           context.Context
		setupMock     func()
		expectedError bool
		verify        func(res *dto.GetMeOutputDto)
	}{
		{
			name: "Success - returns current user details",
			ctx:  context.WithValue(context.Background(), "userID", userID.String()),
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetMeOutputDto) {
				s.NotNil(res)
				s.Equal(userID, res.Body.ID)
				s.Equal("john@example.com", res.Body.Email)
				s.Equal("johndoe", res.Body.Username)
			},
		},
		{
			name:          "Failure - missing userID in context",
			ctx:           context.Background(),
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.GetMeOutputDto) {
				s.Nil(res)
			},
		},
		{
			name:          "Failure - invalid UUID string in context",
			ctx:           context.WithValue(context.Background(), "userID", "invalid-uuid"),
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.GetMeOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - user not found in repository",
			ctx:  context.WithValue(context.Background(), "userID", userID.String()),
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("user not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetMeOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetMe(tc.ctx, &dto.GetMeInputDto{})

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

func (s *UserHandlerTestSuite) TestUpdateMe() {
	userID := uuid.New()
	existingUser := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
	}

	updateInput := &dto.UpdateMeInputDto{
		Body: struct {
			FirstName string          `json:"firstName"`
			LastName  string          `json:"lastName"`
			Telephone string          `json:"telephone"`
			Address   dto.UserAddress `json:"address"`
		}{
			FirstName: "Jane",
			LastName:  "Smith",
			Address:   dto.UserAddress{},
		},
	}

	testCases := []struct {
		name          string
		ctx           context.Context
		input         *dto.UpdateMeInputDto
		setupMock     func()
		expectedError bool
		verify        func(res *dto.UpdateMeOutputDto)
	}{
		{
			name:  "Success - updates user profile",
			ctx:   context.WithValue(context.Background(), "userID", userID.String()),
			input: updateInput,
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(existingUser, nil).Once()
				s.mockRepo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
					return u.FirstName == "Jane" && u.LastName == "Smith"
				})).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.UpdateMeOutputDto) {
				s.NotNil(res)
				s.Equal("Profile Updated Successfully", res.Body.Message)
			},
		},
		{
			name:          "Failure - missing userID in context",
			ctx:           context.Background(),
			input:         updateInput,
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.UpdateMeOutputDto) {
				s.Nil(res)
			},
		},
		{
			name:          "Failure - invalid UUID string in context",
			ctx:           context.WithValue(context.Background(), "userID", "invalid-uuid"),
			input:         updateInput,
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.UpdateMeOutputDto) {
				s.Nil(res)
			},
		},
		{
			name:  "Failure - user not found in repository (GetUserByID returns error)",
			ctx:   context.WithValue(context.Background(), "userID", userID.String()),
			input: updateInput,
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("user not found")).Once()
			},
			expectedError: true,
			verify: func(res *dto.UpdateMeOutputDto) {
				s.Nil(res)
			},
		},
		{
			name:  "Failure - database error on update (UpdateUser returns error)",
			ctx:   context.WithValue(context.Background(), "userID", userID.String()),
			input: updateInput,
			setupMock: func() {
				s.mockRepo.On("GetUserByID", mock.Anything, userID).Return(existingUser, nil).Once()
				s.mockRepo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
					return u.FirstName == "Jane" && u.LastName == "Smith"
				})).Return(errors.New("db error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.UpdateMeOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.UpdateMe(tc.ctx, tc.input)

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

func (s *UserHandlerTestSuite) TestGetUsers() {
	usersList := []database.User{
		{
			Username:  "user1",
			FirstName: "First1",
			LastName:  "Last1",
			Email:     "user1@example.com",
		},
		{
			Username:  "user2",
			FirstName: "First2",
			LastName:  "Last2",
			Email:     "user2@example.com",
		},
	}

	testCases := []struct {
		name          string
		setupMock     func()
		input         *dto.GetUsersInputDto
		expectedError bool
		verify        func(res *dto.GetUsersOutputDto)
	}{
		{
			name: "Success - returns users list and total count",
			input: &dto.GetUsersInputDto{
				PageNumber: 1,
				PageSize:   10,
			},
			setupMock: func() {
				s.mockRepo.On("GetUsers", mock.Anything, int64(1), int64(10), mock.Anything, mock.Anything).Return(usersList, 2, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetUsersOutputDto) {
				s.NotNil(res)
				s.Equal(int64(2), res.Body.Total)
				s.Len(res.Body.Users, 2)
				s.Equal("user1", res.Body.Users[0].Username)
				s.Equal("user2", res.Body.Users[1].Username)
			},
		},
		{
			name: "Success - with keyword search and order",
			input: &dto.GetUsersInputDto{
				PageNumber: 1,
				PageSize:   10,
				Search:     "First1",
				Order:      "first_name ASC",
			},
			setupMock: func() {
				s.mockRepo.On("GetUsers", mock.Anything, int64(1), int64(10), "First1", "first_name ASC").Return([]database.User{usersList[0]}, 1, nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.GetUsersOutputDto) {
				s.NotNil(res)
				s.Equal(int64(1), res.Body.Total)
				s.Len(res.Body.Users, 1)
				s.Equal("user1", res.Body.Users[0].Username)
			},
		},
		{
			name: "Failure - repository error",
			input: &dto.GetUsersInputDto{
				PageNumber: 1,
				PageSize:   10,
			},
			setupMock: func() {
				s.mockRepo.On("GetUsers", mock.Anything, int64(1), int64(10), mock.Anything, mock.Anything).Return(nil, 0, errors.New("repository error")).Once()
			},
			expectedError: true,
			verify: func(res *dto.GetUsersOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			res, err := s.handler.GetUsers(s.ctx, tc.input)

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

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}

func (s *UserHandlerTestSuite) TestGetUsersCount_Success() {
	s.mockRepo.On("GetUsersCount", mock.Anything).Return(15, nil).Once()

	res, err := s.handler.GetUsersCount(s.ctx, &struct{}{})
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int64(15), res.Body.Count)
}

func (s *UserHandlerTestSuite) TestGetUsersCount_Error() {
	s.mockRepo.On("GetUsersCount", mock.Anything).Return(0, errors.New("db error")).Once()

	res, err := s.handler.GetUsersCount(s.ctx, &struct{}{})
	s.Error(err)
	s.Nil(res)
}
