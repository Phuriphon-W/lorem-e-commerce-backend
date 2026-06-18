package handler

import (
	"context"
	"errors"
	"fmt"
	"lorem-backend/internal/database"
	fileRepo "lorem-backend/internal/modules/file/repository"
	"lorem-backend/internal/modules/order/dto"
	"lorem-backend/internal/modules/order/repository"
	productDto "lorem-backend/internal/modules/product/dto"
	productRepo "lorem-backend/internal/modules/product/repository"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type orderHandlerImpl struct {
	db                database.Database
	orderRepository   repository.OrderRepository
	productRepository productRepo.ProductRepository
	fileRepository    fileRepo.FileRepository
}

func NewOrderHandlerImpl(db database.Database, orderRepo repository.OrderRepository, prodRepo productRepo.ProductRepository, fileRepo fileRepo.FileRepository) OrderHandler {
	return &orderHandlerImpl{
		db:                db,
		orderRepository:   orderRepo,
		productRepository: prodRepo,
		fileRepository:    fileRepo,
	}
}

func (h *orderHandlerImpl) CreateOrder(ctx context.Context, input *dto.CreateOrderInputDto) (*dto.CreatedOrderOutputDto, error) {
	// Ownership verification
	authenticatedUserIDStr, ok := ctx.Value("userID").(string)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}
	isAdmin, _ := ctx.Value("isAdmin").(bool)
	if !isAdmin && authenticatedUserIDStr != input.Body.UserID.String() {
		return nil, huma.Error403Forbidden("Forbidden: You do not own this resource")
	}

	if len(input.Body.Items) == 0 {
		return nil, huma.Error400BadRequest("Order must contain at least one item")
	}

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

	const maxRetries = 5
	var baseDelay = 10 * time.Millisecond

	for attempt := range maxRetries {
		var createdOrderID uuid.UUID

		// Run everything in a single transaction
		err := h.db.GetDb().Transaction(func(tx *gorm.DB) error {
			// Propagate transaction in context
			txCtx := database.WithTransaction(ctx, tx)

			// 1. Fetch products inside transaction to see current state
			products, err := h.productRepository.GetProductsByIDs(txCtx, productIDs)
			if err != nil {
				return err
			}

			// Verify all requested products exist
			if len(products) != len(productIDs) {
				return fmt.Errorf("one or more products could not be found")
			}

			var totalPrice float32
			var orderItems []database.OrderItem
			var deductions []productRepo.StockDeduction

			for _, product := range products {
				requestedQty := itemMap[product.ID]

				priceAtPurchase := product.Price
				totalPrice += priceAtPurchase * float32(requestedQty)

				orderItems = append(orderItems, database.OrderItem{
					ProductID:       product.ID,
					PriceAtPurchase: priceAtPurchase,
					Quantity:        requestedQty,
				})

				deductions = append(deductions, productRepo.StockDeduction{
					ProductID: product.ID,
					Quantity:  requestedQty,
				})
			}

			// 2. Deduct stock (runs atomically and checks bounds inside transaction)
			if err := h.productRepository.DeductProductStocks(txCtx, deductions); err != nil {
				return err
			}

			// 3. Build and save the final order inside transaction
			order := &database.Order{
				UserID:      input.Body.UserID,
				TotalPrice:  totalPrice,
				OrderStatus: database.Pending,
				OrderItems:  orderItems,
			}

			oid, err := h.orderRepository.CreateOrder(txCtx, order)
			if err != nil {
				return err
			}

			createdOrderID = oid
			return nil
		})

		if err != nil {
			// If it's a deadlock and we have retries left, retry
			if isDeadlock(err) && attempt < maxRetries-1 {
				time.Sleep(baseDelay * time.Duration(1<<attempt))
				continue
			}

			// Check if it's a known error from DeductProductStocks or Huma Error
			var humaErr huma.StatusError
			if errors.As(err, &humaErr) {
				return nil, humaErr
			}

			if strings.Contains(err.Error(), "insufficient stock") {
				return nil, huma.Error400BadRequest(err.Error())
			}

			if strings.Contains(err.Error(), "could not be found") {
				return nil, huma.Error404NotFound(err.Error())
			}

			return nil, huma.Error500InternalServerError("Failed to create order", err)
		}

		return &dto.CreatedOrderOutputDto{
			Body: dto.CreatedOrderOutputDtoBody{
				ID: createdOrderID,
			},
		}, nil
	}

	return nil, huma.Error500InternalServerError("Failed to create order due to persistent database contention", nil)
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

	// Ownership verification
	authenticatedUserIDStr, ok := ctx.Value("userID").(string)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}
	isAdmin, _ := ctx.Value("isAdmin").(bool)
	if !isAdmin && order.UserID.String() != authenticatedUserIDStr {
		return nil, huma.Error403Forbidden("Forbidden: You do not own this order")
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

func (h *orderHandlerImpl) GetOrdersCount(ctx context.Context, input *struct{}) (*dto.GetOrdersCountOutputDto, error) {
	count, err := h.orderRepository.GetOrdersCount(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve orders count", err)
	}

	return &dto.GetOrdersCountOutputDto{
		Body: struct {
			Count int64 `json:"count" doc:"Total number of orders"`
		}{
			Count: count,
		},
	}, nil
}

// Helper function to keep handler clean
func (h *orderHandlerImpl) mapOrderToResponse(ctx context.Context, ord database.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(ord.OrderItems))

	for i, item := range ord.OrderItems {
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
	}

	var stripeExpiresAt *int64
	if ord.StripeSessionExpiresAt != nil {
		ts := ord.StripeSessionExpiresAt.Unix()
		stripeExpiresAt = &ts
	}

	return dto.OrderResponse{
		ID:                     ord.ID,
		UserID:                 ord.UserID,
		TotalPrice:             ord.TotalPrice,
		OrderStatus:            ord.OrderStatus,
		StripeSessionExpiresAt: stripeExpiresAt,
		OrderItems:             items,
		CreatedAt:              ord.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func isDeadlock(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40P01"
	}
	return strings.Contains(err.Error(), "40P01") || strings.Contains(err.Error(), "deadlock detected")
}
