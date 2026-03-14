package handler

import (
	"context"
	"lorem-backend/internal/modules/user/dto"
)

type UserHandler interface {
	GetUserById(ctx context.Context, input *dto.GetUserByIdInputDto) (*dto.GetUserByIdOutputDto, error)
	GetMe(ctx context.Context, input *dto.GetMeInputDto) (*dto.GetMeOutputDto, error)
}
