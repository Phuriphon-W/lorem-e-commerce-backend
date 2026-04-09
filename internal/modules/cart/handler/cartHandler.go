package handler

import (
	"context"
	"lorem-backend/internal/modules/cart/dto"
)

type CartHandler interface {
	GetCartByUserId(ctx context.Context, input *dto.GetCartByUserIdInputDto) (*dto.GetCartByUserIdOutputDto, error)
	CreateCartItem(ctx context.Context, input *dto.CreateCartItemInputDto) (*dto.CreateCartItemOutputDto, error)
	EditCartItem(ctx context.Context, input *dto.EditCartItemInputDto) (*dto.EditCartItemOutputDto, error)
	DeleteCartItems(ctx context.Context, input *dto.DeleteCartItemsInputDto) (*dto.DeleteCartItemsOutputDto, error)
}
