package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/modules/user/dto"
	"lorem-backend/internal/modules/user/repository"
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
		return nil, fmt.Errorf("Failed to retrieve user with ID: %v", input.ID)
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
