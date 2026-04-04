package dto

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	catDto "lorem-backend/internal/modules/category/dto"
)

type ProductDtoBase struct {
	Name        string  `json:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
	Description string  `json:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
	Price       float32 `json:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
	Available   uint    `json:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
	ImageURL    string  `json:"image_url" required:"true" doc:"Image URL" example:"https://example.com/images/shirt.jpg"`
}

type ProductFormDto struct {
	Name        string        `form:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
	Description string        `form:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
	Price       float32       `form:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
	Available   uint          `form:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
	ImageFile   huma.FormFile `form:"image_file" required:"true" doc:"Image file of the product"`
}

// Product Response (with ID)
type ProductResponse struct {
	ID uuid.UUID `json:"id" doc:"Product unique identifier" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductDtoBase
	Category catDto.CategoryDto `json:"category" doc:"Category of the product"`
}

// Create Product
type (
	CreateProductInputDtoBody struct {
		ProductFormDto
		CategoryId uuid.UUID `form:"categoryId" required:"true" doc:"ID of the product category" example:"fdc93985-b4fd-40d3-ad6c-3fb94c6ec8c7"`
	}

	CreateProductInputDto struct {
		RawBody huma.MultipartFormFiles[struct {
			Name        string        `form:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
			Description string        `form:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
			Price       float32       `form:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
			Available   uint          `form:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
			ImageFile   huma.FormFile `form:"image_file" required:"true" doc:"Image file of the product"`
			CategoryId  uuid.UUID     `form:"categoryId" required:"true" doc:"ID of the product category" example:"fdc93985-b4fd-40d3-ad6c-3fb94c6ec8c7"`
		}]
	}

	CreatedProductOutputDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Created Product ID"`
	}

	CreatedProductOutputDto struct {
		Body CreatedProductOutputDtoBody
	}
)

// Get Products
type (
	GetProductsInputDto struct {
		PageNumber uint64 `query:"pageNumber" default:"1" minimum:"1" doc:"Page number"`
		PageSize   uint64 `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
	}

	GetProductsOutputDtoBody struct {
		Products []ProductResponse `json:"products"`
		Total    int64             `json:"total"`
	}

	GetProductsOutputDto struct {
		Body GetProductsOutputDtoBody
	}
)

// Get Product By ID
type (
	GetProductByIdInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"Product ID"`
	}

	GetProductByIdOutputDto struct {
		Body ProductResponse
	}
)
