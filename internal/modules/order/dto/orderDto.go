package dto

import (
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/product/dto"

	"github.com/google/uuid"
)

// Base Order Item DTO
type OrderItemResponse struct {
	ID              uuid.UUID          `json:"id" doc:"Order Item ID"`
	ProductID       uuid.UUID          `json:"productId"`
	Product         dto.ProductDtoBase `json:"product" doc:"Product Details"`
	PriceAtPurchase float32            `json:"priceAtPurchase" doc:"Price when purchased"`
	Quantity        uint               `json:"quantity" doc:"Quantity purchased"`
}

// Base Order Response
type OrderResponse struct {
	ID          uuid.UUID            `json:"id" doc:"Order ID"`
	UserID      uuid.UUID            `json:"userId" doc:"ID of the user who placed the order"`
	TotalPrice  float32              `json:"totalPrice" doc:"Total price of the order"`
	OrderStatus database.OrderStatus `json:"orderStatus" doc:"Current status of the order" example:"pending"`
	OrderItems  []OrderItemResponse  `json:"orderItems" doc:"List of items in the order"`
	CreatedAt   string               `json:"createdAt" doc:"Order creation date"`
}

// Create Order
type (
	OrderItemRequest struct {
		ProductID uuid.UUID `json:"productId" required:"true" doc:"ID of the product"`
		Quantity  uint      `json:"quantity" required:"true" minimum:"1" doc:"Quantity to order"`
	}

	CreateOrderInputDtoBody struct {
		UserID uuid.UUID          `json:"userId" required:"true" doc:"ID of the user placing the order"`
		Items  []OrderItemRequest `json:"items" required:"true" minItems:"1" doc:"Items to purchase"`
	}

	CreateOrderInputDto struct {
		Body CreateOrderInputDtoBody
	}

	CreatedOrderOutputDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Created Order ID"`
	}

	CreatedOrderOutputDto struct {
		Body CreatedOrderOutputDtoBody
	}
)

// Get Orders (List for User)
type (
	GetOrdersInputDto struct {
		UserID     uuid.UUID `path:"userId" required:"true" doc:"User ID"`
		PageNumber int64     `query:"pageNumber" default:"1" minimum:"1" doc:"Page number"`
		PageSize   int64     `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
		Status     string    `query:"status" doc:"Status of the Order"`
		Order      string    `query:"orderBy" doc:"Ordering of the Orders"`
	}

	GetOrdersOutputDtoBody struct {
		Orders []OrderResponse `json:"orders"`
		Total  int64           `json:"total"`
	}

	GetOrdersOutputDto struct {
		Body GetOrdersOutputDtoBody
	}
)

// Get Order By ID
type (
	GetOrderByIdInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"Order ID"`
	}

	GetOrderByIdOutputDto struct {
		Body OrderResponse
	}
)

// Update Order Status
type (
	UpdateOrderStatusInputDtoBody struct {
		Status database.OrderStatus `json:"status" required:"true" doc:"New status"`
	}

	UpdateOrderStatusInputDto struct {
		ID   uuid.UUID `path:"id" required:"true" doc:"Order ID"`
		Body UpdateOrderStatusInputDtoBody
	}

	UpdateOrderStatusOutputDtoBody struct {
		Success bool `json:"success" doc:"True if update was successful"`
	}

	UpdateOrderStatusOutputDto struct {
		Body UpdateOrderStatusOutputDtoBody
	}
)
