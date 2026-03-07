package handlers

import (
	"context"
	"lorem-backend/internal/modules/user/dtos"
)

type UserHandler interface {
	CreateUser(ctx context.Context, input *dtos.CreateUserRequestDto) (*dtos.CreateUserResponseDto, error)
}
