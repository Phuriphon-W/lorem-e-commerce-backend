package handler

import (
	"context"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/category/dto"
	"lorem-backend/internal/modules/category/repository"

	"github.com/danielgtaylor/huma/v2"
)

type categoryHandlerImpl struct {
	categoryRepository repository.CategoryRepository
}

func NewCategoryHandlerImpl(repo repository.CategoryRepository) CategoryHandler {
	return &categoryHandlerImpl{
		categoryRepository: repo,
	}
}

func (c *categoryHandlerImpl) CreateCategory(ctx context.Context, input *dto.CreateCategoryInputDto) (*dto.CreateCategoryOutputDto, error) {
	category := &database.Category{
		Name: input.Body.Name,
	}

	categoryId, err := c.categoryRepository.CreateCategory(ctx, category)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create category", err)
	}

	res := &dto.CreateCategoryOutputDto{
		Body: dto.CreateCategoryOutputDtoBody{
			ID: categoryId,
		},
	}

	return res, nil
}

func (c *categoryHandlerImpl) GetCategoryById(ctx context.Context, input *dto.GetCategoryByIdInputDto) (*dto.GetCategoryByIdOutputDto, error) {
	category, err := c.categoryRepository.GetCategoryByID(ctx, input.ID)

	if err != nil {
		return nil, huma.Error404NotFound("Error retrieving category", err)
	}

	res := &dto.GetCategoryByIdOutputDto{
		Body: dto.CategoryDto{
			ID:   category.ID,
			Name: category.Name,
		},
	}

	return res, nil
}

func (c *categoryHandlerImpl) GetCategories(ctx context.Context, _ *struct{}) (*dto.GetCategoriesOutputDto, error) {
	categories, err := c.categoryRepository.GetCategories(ctx)

	if err != nil {
		return nil, huma.Error404NotFound("Failed to retrieve categories", err)
	}

	results := make([]dto.CategoryDto, len(categories))
	for i, c := range categories {
		results[i] = dto.CategoryDto{
			ID:   c.ID,
			Name: c.Name,
		}
	}

	res := &dto.GetCategoriesOutputDto{
		Body: results,
	}

	return res, nil
}
