package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/modules/user/dto"
	"lorem-backend/internal/modules/user/repository"

	"github.com/danielgtaylor/huma/v2"
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
		return nil, huma.Error404NotFound("User Not Found", fmt.Errorf("Failed to retrieve user with ID: %v, Error: %v", input.ID, err))
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
		return nil, huma.Error400BadRequest("invalid or missing user ID in context")
	}

	// Parse string to uuid.UUID type
	parsedID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid UUID format", err)
	}

	user, err := u.userRepository.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, huma.Error404NotFound("Failed to get current user data", err)
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
