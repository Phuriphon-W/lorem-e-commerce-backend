package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/modules/user/dto"
	"lorem-backend/internal/modules/user/repository"

	"github.com/google/uuid"
)

type userHandlerImpl struct {
	userRepository repository.UserRepository
}

func NewUserHandlerImpl(
	userRepository repository.UserRepository,
) UserHandler {
	return &userHandlerImpl{
		userRepository: userRepository,
	}
}

func (u *userHandlerImpl) GetUserById(ctx context.Context, input *dto.GetUserByIdInputDto) (*dto.GetUserByIdOutputDto, error) {
	user, err := u.userRepository.GetUserByID(ctx, input.ID)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user with ID: %v\nError: %v", input.ID, err)
	}

	res := &dto.GetUserByIdOutputDto{
		Body: dto.UserDto{
			ID:        user.ID,
			Username:  user.Username,
			LastName:  user.LastName,
			FirstName: user.FirstName,
		},
	}

	return res, nil
}

func (u *userHandlerImpl) GetMe(ctx context.Context, input *dto.GetMeInputDto) (*dto.GetMeOutputDto, error) {
	// Get the value from context
	val := ctx.Value("userID")

	userIDStr, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing user ID in context")
	}

	// Parse string to uuid.UUID type
	parsedID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	user, err := u.userRepository.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user data: %v", err)
	}

	res := &dto.GetMeOutputDto{
		Body: dto.UserDto{
			ID:        user.ID,
			Username:  user.Username,
			LastName:  user.LastName,
			FirstName: user.FirstName,
		},
	}

	return res, nil
}
