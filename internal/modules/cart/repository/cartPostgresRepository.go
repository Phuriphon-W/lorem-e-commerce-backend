package repository

import (
	"context"
	"lorem-backend/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cartPostgresRepository struct {
	db database.Database
}

func NewCartPostgresRepository(db database.Database) CartRepository {
	return &cartPostgresRepository{
		db: db,
	}
}

func (c *cartPostgresRepository) GetCartByUserId(ctx context.Context, userId uuid.UUID) (*database.Cart, error) {
	cart, err := gorm.G[database.Cart](c.db.GetDb()).
		Preload("CartItems", func(db gorm.PreloadBuilder) error {
			db.Order("created_at ASC")
			return nil
		}).
		Preload("CartItems.Product", nil).
		Preload("CartItems.Product.Category", nil).
		Where("user_id = ?", userId).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (c *cartPostgresRepository) CreateCartItem(ctx context.Context, cartItem *database.CartItem) (uuid.UUID, error) {
	err := gorm.G[database.CartItem](c.db.GetDb()).Create(ctx, cartItem)
	if err != nil {
		return uuid.Nil, err
	}

	return cartItem.ID, nil
}

func (c *cartPostgresRepository) GetCartItem(ctx context.Context, cartId, productId uuid.UUID) (*database.CartItem, error) {
	cartItem, err := gorm.G[database.CartItem](c.db.GetDb()).
		Where("cart_id = ? AND product_id = ?", cartId, productId).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &cartItem, nil
}

func (c *cartPostgresRepository) EditCartItem(ctx context.Context, cartId uuid.UUID, productId uuid.UUID, quantity uint) error {
	return c.db.GetDb().WithContext(ctx).
		Model(&database.CartItem{}).
		Where("cart_id = ? AND product_id = ?", cartId, productId).
		Update("quantity", quantity).Error
}

// Hard Delete for cart items with Unscoped
func (c *cartPostgresRepository) RemoveCartItems(ctx context.Context, cartId uuid.UUID, productIds []uuid.UUID) error {
	return c.db.GetDb().WithContext(ctx).
		Unscoped().
		Where("cart_id = ? AND product_id IN ?", cartId, productIds).
		Delete(&database.CartItem{}).Error
}

func (c *cartPostgresRepository) GetProductStock(ctx context.Context, productId uuid.UUID) (uint, error) {
	var product database.Product
	err := c.db.GetDb().WithContext(ctx).
		Select("available").
		Where("id = ?", productId).
		First(&product).Error

	if err != nil {
		return 0, err
	}

	return product.Available, nil
}
