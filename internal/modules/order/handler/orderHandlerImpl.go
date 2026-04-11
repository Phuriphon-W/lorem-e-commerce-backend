package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/order/dto"
	"lorem-backend/internal/modules/order/repository"
	productDto "lorem-backend/internal/modules/product/dto"
	productRepo "lorem-backend/internal/modules/product/repository"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type orderHandlerImpl struct {
	orderRepository   repository.OrderRepository
	productRepository productRepo.ProductRepository
}

func NewOrderHandlerImpl(orderRepo repository.OrderRepository, prodRepo productRepo.ProductRepository) OrderHandler {
	return &orderHandlerImpl{
		orderRepository:   orderRepo,
		productRepository: prodRepo,
	}
}

func (h *orderHandlerImpl) CreateOrder(ctx context.Context, input *dto.CreateOrderInputDto) (*dto.CreatedOrderOutputDto, error) {
	var productIDs []uuid.UUID
	itemMap := make(map[uuid.UUID]uint)

	// Group quantities and collect unique IDs
	for _, item := range input.Body.Items {
		// If the ID isn't in the map yet, add it to our slice for the SQL query
		if _, exists := itemMap[item.ProductID]; !exists {
			productIDs = append(productIDs, item.ProductID)
		}
		// Accumulate quantity (protects against duplicate items in the request array)
		itemMap[item.ProductID] += item.Quantity
	}

	products, err := h.productRepository.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to verify products", err)
	}

	// Verify all requested products exist
	if len(products) != len(productIDs) {
		return nil, huma.Error404NotFound("One or more products could not be found")
	}

	var totalPrice float32
	var orderItems []database.OrderItem

	for _, product := range products {
		requestedQty := itemMap[product.ID]

		// Check stock safely
		if product.Available < requestedQty {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Insufficient stock for product: %s", product.Name))
		}

		// Calculate total
		priceAtPurchase := product.Price
		totalPrice += priceAtPurchase * float32(requestedQty)

		// Build order item
		orderItems = append(orderItems, database.OrderItem{
			ProductID:       product.ID,
			PriceAtPurchase: priceAtPurchase,
			Quantity:        requestedQty,
		})
	}

	// Build and save the final order
	order := &database.Order{
		UserID:      input.Body.UserID,
		TotalPrice:  totalPrice,
		OrderStatus: database.Pending,
		OrderItems:  orderItems,
	}

	oid, err := h.orderRepository.CreateOrder(ctx, order)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create order", err)
	}

	return &dto.CreatedOrderOutputDto{
		Body: dto.CreatedOrderOutputDtoBody{
			ID: oid,
		},
	}, nil
}

func (h *orderHandlerImpl) GetOrders(ctx context.Context, input *dto.GetOrdersInputDto) (*dto.GetOrdersOutputDto, error) {
	orders, total, err := h.orderRepository.GetOrdersByUserID(ctx, input.UserID, input.PageNumber, input.PageSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve orders", err)
	}

	results := make([]dto.OrderResponse, len(orders))
	for i, ord := range orders {
		results[i] = mapOrderToResponse(ord)
	}

	return &dto.GetOrdersOutputDto{
		Body: dto.GetOrdersOutputDtoBody{
			Orders: results,
			Total:  total,
		},
	}, nil
}

func (h *orderHandlerImpl) GetOrderById(ctx context.Context, input *dto.GetOrderByIdInputDto) (*dto.GetOrderByIdOutputDto, error) {
	order, err := h.orderRepository.GetOrderByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Order not found", err)
	}

	return &dto.GetOrderByIdOutputDto{
		Body: mapOrderToResponse(*order),
	}, nil
}

func (h *orderHandlerImpl) UpdateOrderStatus(ctx context.Context, input *dto.UpdateOrderStatusInputDto) (*dto.UpdateOrderStatusOutputDto, error) {
	err := h.orderRepository.UpdateOrderStatus(ctx, input.ID, input.Body.Status)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update order status", err)
	}

	return &dto.UpdateOrderStatusOutputDto{
		Body: dto.UpdateOrderStatusOutputDtoBody{
			Success: true,
		},
	}, nil
}

// Helper function to keep handler clean
func mapOrderToResponse(ord database.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(ord.OrderItems))
	for j, item := range ord.OrderItems {
		items[j] = dto.OrderItemResponse{
			ID:              item.ID,
			ProductID:       item.ProductID,
			PriceAtPurchase: item.PriceAtPurchase,
			Quantity:        item.Quantity,
			Product: productDto.ProductDtoBase{
				Name:        item.Product.Name,
				Description: item.Product.Description,
				Price:       item.Product.Price,
				Available:   item.Product.Available,
				ImageURL:    "", // Note: Generate presigned URL here if needed, similar to GetProducts
			},
		}
	}

	return dto.OrderResponse{
		ID:          ord.ID,
		UserID:      ord.UserID,
		TotalPrice:  ord.TotalPrice,
		OrderStatus: ord.OrderStatus,
		OrderItems:  items,
		CreatedAt:   ord.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
