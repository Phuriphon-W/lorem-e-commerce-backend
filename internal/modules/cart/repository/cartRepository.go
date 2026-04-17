package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
)

type CartRepository interface {
	GetCartByUserId(ctx context.Context, userId uuid.UUID) (*database.Cart, error)
	CreateCartItem(ctx context.Context, cartItem *database.CartItem) (uuid.UUID, error)
	GetCartItem(ctx context.Context, cartId, productId uuid.UUID) (*database.CartItem, error)
	EditCartItem(ctx context.Context, cartId uuid.UUID, productId uuid.UUID, quantity uint) error
	RemoveCartItems(ctx context.Context, cartId uuid.UUID, productIds []uuid.UUID) error
}
