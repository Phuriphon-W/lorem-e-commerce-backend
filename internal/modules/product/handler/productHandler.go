package handler

import (
	"context"
	"lorem-backend/internal/modules/product/dto"
)

type ProductHandler interface {
	CreateProduct(ctx context.Context, input *dto.CreateProductInputDto) (*dto.CreatedProductOutputDto, error)
	GetProducts(ctx context.Context, input *dto.GetProductsInputDto) (*dto.GetProductsOutputDto, error)
	GetProductById(ctx context.Context, input *dto.GetProductByIdInputDto) (*dto.GetProductByIdOutputDto, error)
}
