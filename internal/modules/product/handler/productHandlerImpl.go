package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	catDto "lorem-backend/internal/modules/category/dto"
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
	products, total, err := p.productRepository.GetProducts(ctx, input.PageNumber, input.PageSize)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve products: %v", err)
	}

	results := make([]dto.ProductResponse, len(products))
	for i, p := range products {
		results[i] = dto.ProductResponse{
			ID: p.ID,
			ProductDtoBase: dto.ProductDtoBase{
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
				Available:   p.Available,
				ImageURL:    p.ImageURL,
			},
			Category: catDto.CategoryDto{
				ID:   p.CategoryID,
				Name: p.Category.Name,
			},
		}
	}

	res := &dto.GetProductsOutputDto{
		Body: dto.GetProductsOutputDtoBody{
			Products: results,
			Total:    total,
		},
	}

	return res, nil
}
