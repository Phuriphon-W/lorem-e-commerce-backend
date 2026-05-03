package handler

import (
	"context"
	"errors"
	"testing"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/user/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock type for the UserRepository type
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUsers(ctx context.Context) ([]database.User, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]database.User), args.Error(1)
	}
	return nil, args.Error(1)
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

func TestUserHandlerImpl_GetUserById_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	handler := NewUserHandlerImpl(mockRepo)

	userID := uuid.New()
	expectedUser := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
	}

	mockRepo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)

	input := &dto.GetUserByIdInputDto{ID: userID}
	res, err := handler.GetUserById(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, userID, res.Body.ID)
	assert.Equal(t, "johndoe", res.Body.Username)
	mockRepo.AssertExpectations(t)
}

func TestUserHandlerImpl_GetUserById_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	handler := NewUserHandlerImpl(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("not found"))

	input := &dto.GetUserByIdInputDto{ID: userID}
	res, err := handler.GetUserById(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, res)
	mockRepo.AssertExpectations(t)
}

func TestUserHandlerImpl_GetMe_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	handler := NewUserHandlerImpl(mockRepo)

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

	mockRepo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)

	ctx := context.WithValue(context.Background(), "userID", userID.String())
	input := &dto.GetMeInputDto{}

	res, err := handler.GetMe(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, userID, res.Body.ID)
	assert.Equal(t, "john@example.com", res.Body.Email)
	mockRepo.AssertExpectations(t)
}

func TestUserHandlerImpl_UpdateMe_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	handler := NewUserHandlerImpl(mockRepo)

	userID := uuid.New()
	existingUser := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
	}

	mockRepo.On("GetUserByID", mock.Anything, userID).Return(existingUser, nil)
	mockRepo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
		return u.FirstName == "Jane" && u.LastName == "Smith"
	})).Return(nil)

	ctx := context.WithValue(context.Background(), "userID", userID.String())
	input := &dto.UpdateMeInputDto{
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

	res, err := handler.UpdateMe(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Profile Updated Successfully", res.Body.Message)
	mockRepo.AssertExpectations(t)
}
