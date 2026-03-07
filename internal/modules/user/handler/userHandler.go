package handler

import (
	"context"
	"lorem-backend/internal/modules/user/dto"
)

type UserHandler interface {
	CreateUser(ctx context.Context, input *dto.CreateUserInputDto) (*dto.CreateUserOutputDto, error)
	GetUserById(ctx context.Context, input *dto.GetUserByIdInputDto) (*dto.GetUserByIdOutputDto, error)
}
