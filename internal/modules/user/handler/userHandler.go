package handler

import (
	"context"
	"lorem-backend/internal/modules/user/dto"
)

type UserHandler interface {
	GetUserById(ctx context.Context, input *dto.GetUserByIdInputDto) (*dto.GetUserByIdOutputDto, error)
}
