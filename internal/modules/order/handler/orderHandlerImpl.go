package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	fileRepo "lorem-backend/internal/modules/file/repository"
	"lorem-backend/internal/modules/order/dto"
	"lorem-backend/internal/modules/order/repository"
	productDto "lorem-backend/internal/modules/product/dto"
	productRepo "lorem-backend/internal/modules/product/repository"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type orderHandlerImpl struct {
	orderRepository   repository.OrderRepository
	productRepository productRepo.ProductRepository
	fileRepository    fileRepo.FileRepository
}

func NewOrderHandlerImpl(orderRepo repository.OrderRepository, prodRepo productRepo.ProductRepository, fileRepo fileRepo.FileRepository) OrderHandler {
	return &orderHandlerImpl{
		orderRepository:   orderRepo,
		productRepository: prodRepo,
		fileRepository:    fileRepo,
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
	var deductions []productRepo.StockDeduction

	for _, product := range products {
		requestedQty := itemMap[product.ID]

		// Calculate total
		priceAtPurchase := product.Price
		totalPrice += priceAtPurchase * float32(requestedQty)

		// Build order item
		orderItems = append(orderItems, database.OrderItem{
			ProductID:       product.ID,
			PriceAtPurchase: priceAtPurchase,
			Quantity:        requestedQty,
		})

		// Build deduction
		deductions = append(deductions, productRepo.StockDeduction{
			ProductID: product.ID,
			Quantity:  requestedQty,
		})
	}

	// Deduct stock
	if err := h.productRepository.DeductProductStocks(ctx, deductions); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
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
		// Revert stock if order creation fails
		_ = h.productRepository.AddProductStocks(ctx, deductions)
		return nil, huma.Error500InternalServerError("Failed to create order", err)
	}

	return &dto.CreatedOrderOutputDto{
		Body: dto.CreatedOrderOutputDtoBody{
			ID: oid,
		},
	}, nil
}

func (h *orderHandlerImpl) GetOrders(ctx context.Context, input *dto.GetOrdersInputDto) (*dto.GetOrdersOutputDto, error) {
	// Validate order input
	var order string

	if input.Order == "date_asc" {
		order = "created_at ASC"
	} else {
		order = "created_at DESC"
	}

	orders, total, err := h.orderRepository.GetOrdersByUserID(ctx, input.UserID, input.PageNumber, input.PageSize, input.Status, order)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve orders", err)
	}

	results := make([]dto.OrderResponse, len(orders))
	for i, ord := range orders {
		results[i] = h.mapOrderToResponse(ctx, ord)
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
		Body: h.mapOrderToResponse(ctx, *order),
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
func (h *orderHandlerImpl) mapOrderToResponse(ctx context.Context, ord database.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(ord.OrderItems))

	var wg sync.WaitGroup

	for i, item := range ord.OrderItems {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// generate product image url
			itemImageUrl, err := h.fileRepository.GeneratePresignUrl(ctx, item.Product.ImageObjKey)
			if err != nil {
				fmt.Printf("Error generating URL for %s: %v\n", item.Product.ID, err)
				itemImageUrl = ""
			}

			// Map to DTO
			items[i] = dto.OrderItemResponse{
				ID:              item.ID,
				ProductID:       item.ProductID,
				PriceAtPurchase: item.PriceAtPurchase,
				Quantity:        item.Quantity,
				Product: productDto.ProductDtoBase{
					Name:        item.Product.Name,
					Description: item.Product.Description,
					Price:       item.Product.Price,
					Available:   item.Product.Available,
					ImageURL:    itemImageUrl,
				},
			}
		}()
	}

	wg.Wait()

	return dto.OrderResponse{
		ID:          ord.ID,
		UserID:      ord.UserID,
		TotalPrice:  ord.TotalPrice,
		OrderStatus: ord.OrderStatus,
		OrderItems:  items,
		CreatedAt:   ord.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
