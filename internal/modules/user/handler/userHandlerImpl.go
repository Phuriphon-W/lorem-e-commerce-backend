package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/user/dto"
	"lorem-backend/internal/modules/user/repository"
	"lorem-backend/internal/utils"
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

func (u *userHandlerImpl) CreateUser(ctx context.Context, input *dto.CreateUserInputDto) (*dto.CreateUserOutputDto, error) {
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

	userID, err := u.userRepository.CreateUser(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("Failed to create user: %v", err)
	}

	res := &dto.CreateUserOutputDto{
		Body: dto.CreateUserOutputDtoBody{
			ID: userID,
		},
	}
	return res, nil
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
