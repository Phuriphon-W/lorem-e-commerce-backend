package handler

import (
	"context"
	"lorem-backend/internal/modules/order/dto"
)

type OrderHandler interface {
	CreateOrder(ctx context.Context, input *dto.CreateOrderInputDto) (*dto.CreatedOrderOutputDto, error)
	GetOrders(ctx context.Context, input *dto.GetOrdersInputDto) (*dto.GetOrdersOutputDto, error)
	GetOrderById(ctx context.Context, input *dto.GetOrderByIdInputDto) (*dto.GetOrderByIdOutputDto, error)
	UpdateOrderStatus(ctx context.Context, input *dto.UpdateOrderStatusInputDto) (*dto.UpdateOrderStatusOutputDto, error)
}
