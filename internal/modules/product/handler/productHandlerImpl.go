package handler

import (
	"context"
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
	}
}

func (p *productHandlerImpl) GetProducts(ctx context.Context, input dto.GetProductsInputDto) (*dto.GetProductsOutputDto, error) {

}
