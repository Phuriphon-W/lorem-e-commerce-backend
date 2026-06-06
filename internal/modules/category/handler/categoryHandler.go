package handler

import (
	"context"
	"lorem-backend/internal/modules/category/dto"
)

type CategoryHandler interface {
	CreateCategory(ctx context.Context, input *dto.CreateCategoryInputDto) (*dto.CreateCategoryOutputDto, error)
	GetCategoryById(ctx context.Context, input *dto.GetCategoryByIdInputDto) (*dto.GetCategoryByIdOutputDto, error)
	GetCategories(ctx context.Context, _ *struct{}) (*dto.GetCategoriesOutputDto, error)
	UpdateCategory(ctx context.Context, input *dto.UpdateCategoryByIdInputDto) (*dto.UpdateCategoryByIdOutputDto, error)
	DeleteCategory(ctx context.Context, input *dto.DeleteCategoryByIdInputDto) (*dto.DeleteCategoryByIdOutputDto, error)
	GetCategoriesCount(ctx context.Context, input *struct{}) (*dto.GetCategoriesCountOutputDto, error)
}
