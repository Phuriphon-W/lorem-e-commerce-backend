package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/modules/user/dto"
	"lorem-backend/internal/modules/user/repository"
	"lorem-backend/internal/utils"

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

	var isAdmin bool
	if user.IsAdmin != nil {
		isAdmin = *user.IsAdmin
	}

	userAddress := dto.UserAddress{
		ZipCode:     user.ZipCode,
		Road:        user.Road,
		District:    user.District,
		SubDistrict: user.SubDistrict,
		HouseNumber: user.HouseNumber,
		Province:    user.Province,
	}

	res := &dto.GetUserByIdOutputDto{
		Body: dto.UserDto{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Telephone: user.Telephone,
			Address:   userAddress,
			IsAdmin:   isAdmin,
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

	var isAdmin bool
	if user.IsAdmin != nil {
		isAdmin = *user.IsAdmin
	}

	userAddress := dto.UserAddress{
		ZipCode:     user.ZipCode,
		Road:        user.Road,
		District:    user.District,
		SubDistrict: user.SubDistrict,
		HouseNumber: user.HouseNumber,
		Province:    user.Province,
	}

	res := &dto.GetMeOutputDto{
		Body: dto.UserDto{
			ID:        user.ID,
			Username:  user.Username,
			LastName:  user.LastName,
			FirstName: user.FirstName,
			Email:     user.Email,
			Telephone: user.Telephone,
			Address:   userAddress,
			IsAdmin:   isAdmin,
		},
	}

	return res, nil
}

func (u *userHandlerImpl) UpdateMe(ctx context.Context, input *dto.UpdateMeInputDto) (*dto.UpdateMeOutputDto, error) {
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

	// Fetch the existing user to ensure they exist and we don't overwrite other fields
	user, err := u.userRepository.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, huma.Error404NotFound("User not found", err)
	}

	// Update ONLY the fields allowed by the frontend
	// We use the helper to safely convert strings to pointers and handle empty inputs
	user.FirstName = input.Body.FirstName
	user.LastName = input.Body.LastName
	user.Telephone = utils.StringToPtr(input.Body.Telephone)
	user.HouseNumber = utils.PtrToNilIfEmpty(input.Body.Address.HouseNumber)
	user.Road = utils.PtrToNilIfEmpty(input.Body.Address.Road)
	user.District = utils.PtrToNilIfEmpty(input.Body.Address.District)
	user.SubDistrict = utils.PtrToNilIfEmpty(input.Body.Address.SubDistrict)
	user.Province = utils.PtrToNilIfEmpty(input.Body.Address.Province)
	user.ZipCode = utils.PtrToNilIfEmpty(input.Body.Address.ZipCode)

	// Save the updated user back to the database
	err = u.userRepository.UpdateUser(ctx, user)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update user profile", err)
	}

	// Return success response
	return &dto.UpdateMeOutputDto{
		Body: dto.UpdateMeOutputDtoBody{
			Message: "Profile Updated Successfully",
		},
	}, nil
}

func (u *userHandlerImpl) GetUsers(ctx context.Context, input *dto.GetUsersInputDto) (*dto.GetUsersOutputDto, error) {
	users, total, err := u.userRepository.GetUsers(ctx, input.PageNumber, input.PageSize, input.Search, input.Order)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve users", err)
	}

	userDtos := make([]dto.UserDto, len(users))
	for i, user := range users {
		var isAdmin bool
		if user.IsAdmin != nil {
			isAdmin = *user.IsAdmin
		}

		userAddress := dto.UserAddress{
			ZipCode:     user.ZipCode,
			Road:        user.Road,
			District:    user.District,
			SubDistrict: user.SubDistrict,
			HouseNumber: user.HouseNumber,
			Province:    user.Province,
		}

		userDtos[i] = dto.UserDto{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Telephone: user.Telephone,
			Address:   userAddress,
			IsAdmin:   isAdmin,
		}
	}

	return &dto.GetUsersOutputDto{
		Body: dto.GetUsersOutputDtoBody{
			Users: userDtos,
			Total: total,
		},
	}, nil
}

func (u *userHandlerImpl) GetUsersCount(ctx context.Context, input *struct{}) (*dto.GetUsersCountOutputDto, error) {
	count, err := u.userRepository.GetUsersCount(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve users count", err)
	}

	return &dto.GetUsersCountOutputDto{
		Body: struct {
			Count int64 `json:"count" doc:"Total number of users"`
		}{
			Count: count,
		},
	}, nil
}
