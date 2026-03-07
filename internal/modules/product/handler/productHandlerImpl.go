package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/product/dto"
	"lorem-backend/internal/modules/product/repository"
)

type productHandlerImpl struct {
	productRepository repository.ProductRepository
}

func NewProductHandlerImpl(repo repository.ProductRepository) ProductHandler {
	return &productHandlerImpl{
		productRepository: repo,
	}
}

func (p *productHandlerImpl) CreateProduct(ctx context.Context, input *dto.CreateProductInputDto) (*dto.CreatedProductOutputDto, error) {
	product := &database.Product{
		Name:        input.Body.Name,
		Description: input.Body.Description,
		Price:       input.Body.Price,
		Available:   input.Body.Available,
		ImageURL:    input.Body.ImageURL,
		CategoryID:  input.Body.CategoryId,
	}

	pid, err := p.productRepository.CreateProduct(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("Failed to create product: %v", err)
	}

	res := &dto.CreatedProductOutputDto{
		Body: dto.CreatedProductOutputDtoBody{
			ID: pid,
		},
	}

	return res, nil
}

func (p *productHandlerImpl) GetProducts(ctx context.Context, input *dto.GetProductsInputDto) (*dto.GetProductsOutputDto, error) {
	return nil, nil
}
