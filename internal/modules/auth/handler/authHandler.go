package handler

import (
	"context"
	"lorem-backend/internal/modules/auth/dto"
)

type AuthHandler interface {
	RegisterUser(ctx context.Context, input *dto.RegisterUserInputDto) (*dto.RegisterUserOutputDto, error)
}
