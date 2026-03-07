package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/category/dto"
	"lorem-backend/internal/modules/category/repository"
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
		return nil, fmt.Errorf("Failed to create category: %v", err)
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
		return nil, fmt.Errorf("Error retrieving category with ID: %v", input.ID)
	}

	res := &dto.GetCategoryByIdOutputDto{
		Body: dto.CategoryDto{
			ID:   category.ID,
			Name: category.Name,
		},
	}

	return res, nil
}
