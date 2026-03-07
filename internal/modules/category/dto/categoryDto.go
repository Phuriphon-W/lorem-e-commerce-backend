package dto

import "github.com/google/uuid"

// Category DTO
type CategoryDto struct {
	ID   uuid.UUID `json:"id" doc:"Category unique identifier" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name string    `json:"name" minLength:"1" doc:"Category name" example:"Apparel"`
}

// Create Category
type (
	CreateCategoryInputDto struct {
		Body struct {
			Name string `json:"name" required:"true" minLength:"1" doc:"Category name" example:"Apparel"`
		}
	}

	CreateCategoryOutputDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Category unique identifier" example:"550e8400-e29b-41d4-a716-446655440000"`
	}

	CreateCategoryOutputDto struct {
		Body CreateCategoryOutputDtoBody
	}
)

// Get Category By ID
type (
	GetCategoryByIdInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"Category ID"`
	}

	GetCategoryByIdOutputDto struct {
		Body CategoryDto
	}
)
