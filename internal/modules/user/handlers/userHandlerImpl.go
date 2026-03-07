package handlers

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/user/dtos"
	"lorem-backend/internal/modules/user/repositories"
	"lorem-backend/internal/utils"
)

type userHandlerImpl struct {
	userRepository repositories.UserRepository
}

func NewUserHandlerImpl(
	userRepository repositories.UserRepository,
) UserHandler {
	return &userHandlerImpl{
		userRepository: userRepository,
	}
}

func (u *userHandlerImpl) CreateUser(ctx context.Context, input *dtos.CreateUserRequestDto) (*dtos.CreateUserResponseDto, error) {
	hashed, err := utils.HashPassword(input.Body.Password)
	if err != nil {
		return nil, fmt.Errorf("Failed to hash password: %v", err)
	}

	data := &database.User{
		Username:     input.Body.Username,
		FirstName:    input.Body.FirstName,
		LastName:     input.Body.LastName,
		Email:        input.Body.Email,
		PasswordHash: hashed,
	}

	userID, err := u.userRepository.CreateUser(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("Failed to create user: %v", err)
	}

	res := &dtos.CreateUserResponseDto{
		Body: dtos.CreateUserResponseDtoBody{
			ID: userID,
		},
	}
	return res, nil
}
