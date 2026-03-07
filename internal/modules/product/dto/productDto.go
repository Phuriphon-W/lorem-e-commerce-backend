package dto

import "github.com/google/uuid"

// Product DTO
type ProductDto struct {
	Name        string  `json:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
	Description *string `json:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
	Category    string  `json:"category" required:"true" doc:"Category" example:"Clothing"`
	Price       float32 `json:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
	Available   uint    `json:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
	ImageURL    string  `json:"image_url" required:"true" doc:"Image URL" example:"https://example.com/images/shirt.jpg"`
}

// Product Response (with ID)
type ProductResponse struct {
	ID uuid.UUID `json:"id" doc:"Product unique identifier" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductDto
}

// Create Product
type (
	CreateProductInputDto struct {
		Body ProductDto
	}

	CreatedProductOutputDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Created Product ID"`
	}

	CreatedProductOutputDto struct {
		Body CreatedProductOutputDtoBody
	}
)

// Get Product
type (
	GetProductsInputDto struct {
		Body struct {
			PageNumber uint64 `query:"pageNumber" default:"1" minimum:"1" doc:"Page number"`
			PageSize   uint64 `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
		}
	}

	GetProductsOutputDtoBody struct {
		Products []ProductResponse
		Total    uint64
	}

	GetProductsOutputDto struct {
		Body GetProductsOutputDtoBody
	}
)
