package dtos

import "time"

type ProductSchema struct {
	Name        string  `json:"name" required:"true" minLength:"1" doc:"Product name" example:"Shirt"`
	Description string  `json:"description" maxLength:"500" doc:"Description" example:"A comfortable cotton shirt."`
	Category    string  `json:"category" required:"true" doc:"Category" example:"Clothing"`
	Price       float64 `json:"price" required:"true" minimum:"0.01" doc:"Price" example:"19.99"`
	Available   int     `json:"available" required:"true" minimum:"0" doc:"Available stock quantity" example:"100"`
	ImageURL    string  `json:"image_url" required:"true" doc:"Image URL" example:"https://example.com/images/shirt.jpg"`
}

type AddProductInput struct {
	Body ProductSchema
}

type AddProductOutput struct {
	Body struct {
		ID string `json:"id" doc:"The product ID" example:"123e4567-e89b-12d3-a456-426614174000"`
		ProductSchema
		CreatedAt time.Time `json:"created_at" doc:"Creation timestamp" example:"2024-01-01T12:00:00Z"`
	}
}

type UpdateProductInput struct {
	ID   string        `path:"id" doc:"The product ID to update"` // From URL
	Body ProductSchema // From JSON
}

// 4. Output
type ProductIDOutput struct {
	Body struct {
		ID string `json:"id" doc:"The product ID" example:"123e4567-e89b-12d3-a456-426614174000"`
	}
}
