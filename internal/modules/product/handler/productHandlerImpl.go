package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	catDto "lorem-backend/internal/modules/category/dto"
	objectstorage "lorem-backend/internal/modules/objectStorage"
	"lorem-backend/internal/modules/product/dto"
	"lorem-backend/internal/modules/product/repository"
)

type productHandlerImpl struct {
	productRepository repository.ProductRepository
	s3Repository      objectstorage.ObjectStorage
}

func NewProductHandlerImpl(repo repository.ProductRepository, obj objectstorage.ObjectStorage) ProductHandler {
	return &productHandlerImpl{
		productRepository: repo,
		s3Repository:      obj,
	}
}

func (p *productHandlerImpl) CreateProduct(ctx context.Context, input *dto.CreateProductInputDto) (*dto.CreatedProductOutputDto, error) {
	formData := input.RawBody.Data()

	objKey, err := p.s3Repository.UploadFile(
		ctx,
		"product-images",
		formData.ImageFile,
		formData.ImageFile.Size,
		formData.ImageFile.ContentType,
		formData.ImageFile.Filename,
	)
	if err != nil {
		return nil, fmt.Errorf("Error Uploading Product Image to Object Storage: %v", err)
	}

	product := &database.Product{
		Name:        formData.Name,
		Description: formData.Description,
		Price:       formData.Price,
		Available:   formData.Available,
		ImageObjKey: objKey,
		CategoryID:  formData.CategoryId,
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
				ImageURL:    p.ImageObjKey, // TODO: Generate presigned URL
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
