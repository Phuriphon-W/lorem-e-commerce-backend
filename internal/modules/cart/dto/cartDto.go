package dto

import (
	catDto "lorem-backend/internal/modules/category/dto"

	"github.com/google/uuid"
)

type CartItemDto struct {
	ProductID   uuid.UUID          `json:"productId" doc:"Product ID"`
	Name        string             `json:"name" required:"true" minLength:"1" doc:"Item name"`
	Description string             `json:"description" maxLength:"500" doc:"Description"`
	Price       float32            `json:"price" required:"true" minimum:"0.01" doc:"Price"`
	ImageURL    string             `json:"image_url" required:"true" doc:"Image URL"`
	Category    catDto.CategoryDto `json:"category" required:"true" doc:"Item category"`
	Quantity    uint               `json:"quantity" required:"true" minimum:"1" doc:"Quantity in cart"`
	Available   uint               `json:"available" required:"true" doc:"Amount of available products in stock"`
}

// Get Cart
type (
	GetCartByUserIdInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"User ID"`
	}

	GetCartByUserIdOutputDtoBody struct {
		CartID    uuid.UUID     `json:"cartId" doc:"Cart ID"`
		CartItems []CartItemDto `json:"cartItems" doc:"Collection of cart items"`
	}

	GetCartByUserIdOutputDto struct {
		Body GetCartByUserIdOutputDtoBody
	}
)

// Create/Add to Cart
type (
	CreateCartItemInputDto struct {
		UserID uuid.UUID `path:"id" required:"true" doc:"User ID"`
		Body   struct {
			ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
			Quantity  uint      `json:"quantity" required:"true" minimum:"1" default:"1" doc:"Quantity to add"`
		}
	}

	CreateCartItemOutputDtoBody struct {
		CartItemID uuid.UUID `json:"cartItemId" doc:"Created or updated Cart Item ID"`
	}

	CreateCartItemOutputDto struct {
		Body CreateCartItemOutputDtoBody
	}
)

// Edit Cart Item
type (
	EditCartItemInputDto struct {
		UserID uuid.UUID `path:"id" required:"true" doc:"User ID"`
		Body   struct {
			ProductID uuid.UUID `json:"productId" required:"true" doc:"Product ID"`
			Quantity  uint      `json:"quantity" required:"true" minimum:"1" doc:"New exact quantity (must be at least 1)"`
		}
	}

	EditCartItemOutputDtoBody struct {
		Message string `json:"message" doc:"Response message of the request"`
	}

	EditCartItemOutputDto struct {
		Body EditCartItemOutputDtoBody
	}
)

// Delete Cart Items (POST Request)
type (
	DeleteCartItemsInputDto struct {
		UserID uuid.UUID `path:"id" required:"true" doc:"User ID"`
		Body   struct {
			ProductIDs []uuid.UUID `json:"productIds" required:"true" minItems:"1" doc:"List of Product IDs to remove"`
		}
	}

	DeleteCartItemsOutputDtoBody struct {
		Message string `json:"message" doc:"Response message of the request"`
	}

	DeleteCartItemsOutputDto struct {
		Body DeleteCartItemsOutputDtoBody
	}
)
