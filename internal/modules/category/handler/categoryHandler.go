package handler

import (
	"context"
	"lorem-backend/internal/modules/category/dto"
)

type CategoryHandler interface {
	CreateCategory(ctx context.Context, input *dto.CreateCategoryInputDto) (*dto.CreateCategoryOutputDto, error)
	GetCategoryById(ctx context.Context, input *dto.GetCategoryByIdInputDto) (*dto.GetCategoryByIdOutputDto, error)
}
