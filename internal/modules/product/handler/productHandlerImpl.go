package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	catDto "lorem-backend/internal/modules/category/dto"
	file "lorem-backend/internal/modules/file/repository"
	"lorem-backend/internal/modules/product/dto"
	"lorem-backend/internal/modules/product/repository"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type productHandlerImpl struct {
	productRepository repository.ProductRepository
	fileRepository    file.FileRepository
}

func NewProductHandlerImpl(repo repository.ProductRepository, fileRepo file.FileRepository) ProductHandler {
	return &productHandlerImpl{
		productRepository: repo,
		fileRepository:    fileRepo,
	}
}

func (p *productHandlerImpl) CreateProduct(ctx context.Context, input *dto.CreateProductInputDto) (*dto.CreatedProductOutputDto, error) {
	formData := input.RawBody.Data()
	putKey := fmt.Sprintf("product-images/%v-%v", time.Now().Unix(), formData.ImageFile.Filename)

	objKey, err := p.fileRepository.UploadFile(
		ctx,
		putKey,
		formData.ImageFile,
		formData.ImageFile.Size,
		formData.ImageFile.ContentType,
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Error Uploading Product Image to Object Storage", err)
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
		return nil, huma.Error500InternalServerError("Failed to create product", err)
	}

	res := &dto.CreatedProductOutputDto{
		Body: dto.CreatedProductOutputDtoBody{
			ID: pid,
		},
	}

	return res, nil
}

func (p *productHandlerImpl) GetProducts(ctx context.Context, input *dto.GetProductsInputDto) (*dto.GetProductsOutputDto, error) {
	products, total, err := p.productRepository.GetProducts(
		ctx,
		input.PageNumber,
		input.PageSize,
		input.Category,
		input.Search,
		input.Order,
	)
	if err != nil {
		return nil, huma.Error404NotFound("Failed to retrieve products", err)
	}

	results := make([]dto.ProductResponse, len(products))

	// Use a WaitGroup to run S3 calls in parallel
	var wg sync.WaitGroup

	for i, prod := range products {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Generate URL (Parallel)
			imgUrl, err := p.fileRepository.GeneratePresignUrl(ctx, prod.ImageObjKey)
			if err != nil {
				fmt.Printf("Error generating URL for %s: %v\n", prod.ID, err)
				imgUrl = ""
			}

			// Map to DTO
			results[i] = dto.ProductResponse{
				ID: prod.ID,
				ProductDtoBase: dto.ProductDtoBase{
					Name:        prod.Name,
					Description: prod.Description,
					Price:       prod.Price,
					Available:   prod.Available,
					ImageURL:    imgUrl,
				},
				Category: catDto.CategoryDto{
					ID:   prod.CategoryID,
					Name: prod.Category.Name,
				},
			}
		}()
	}

	wg.Wait() // Wait for all S3 calls to finish

	return &dto.GetProductsOutputDto{
		Body: dto.GetProductsOutputDtoBody{
			Products: results,
			Total:    total,
		},
	}, nil
}

func (p *productHandlerImpl) GetProductById(ctx context.Context, input *dto.GetProductByIdInputDto) (*dto.GetProductByIdOutputDto, error) {
	product, err := p.productRepository.GetProductByID(ctx, input.ID)

	if err != nil {
		return nil, huma.Error404NotFound("Error retrieving product", err)
	}

	imgUrl, err := p.fileRepository.GeneratePresignUrl(ctx, product.ImageObjKey)
	if err != nil {
		fmt.Printf("Error generating URL for %s: %v\n", product.ID, err)
		imgUrl = ""
	}

	res := &dto.GetProductByIdOutputDto{
		Body: dto.ProductResponse{
			ID: product.ID,
			ProductDtoBase: dto.ProductDtoBase{
				Name:        product.Name,
				Description: product.Description,
				Price:       product.Price,
				Available:   product.Available,
				ImageURL:    imgUrl,
			},
			Category: catDto.CategoryDto{
				ID:   product.CategoryID,
				Name: product.Category.Name,
			},
		},
	}

	return res, nil
}
