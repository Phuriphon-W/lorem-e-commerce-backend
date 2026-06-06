package handler

import (
	"context"
	"fmt"
	"sync"

	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/cart/dto"
	"lorem-backend/internal/modules/cart/repository"
	catDto "lorem-backend/internal/modules/category/dto"
	fileRepo "lorem-backend/internal/modules/file/repository"
	productRepo "lorem-backend/internal/modules/product/repository"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type cartHandlerImpl struct {
	cartRepo    repository.CartRepository
	productRepo productRepo.ProductRepository
	fileRepo    fileRepo.ObjectStorage
}

func NewCartHandler(cartRepo repository.CartRepository, fileRepo fileRepo.FileRepository, productRepo productRepo.ProductRepository) CartHandler {
	return &cartHandlerImpl{
		cartRepo:    cartRepo,
		fileRepo:    fileRepo,
		productRepo: productRepo,
	}
}

// Get Cart
func (h *cartHandlerImpl) GetCartByUserId(ctx context.Context, input *dto.GetCartByUserIdInputDto) (*dto.GetCartByUserIdOutputDto, error) {
	cart, err := h.cartRepo.GetCartByUserId(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Cart not found", err)
	}

	var activeItems []database.CartItem
	var orphanedProductIDs []uuid.UUID

	for _, item := range cart.CartItems {
		if item.Product.ID == uuid.Nil {
			orphanedProductIDs = append(orphanedProductIDs, item.ProductID)
		} else {
			activeItems = append(activeItems, item)
		}
	}

	// If there are orphaned items, clean them up from the DB
	if len(orphanedProductIDs) > 0 {
		_ = h.cartRepo.RemoveCartItems(ctx, cart.ID, orphanedProductIDs)
	}

	cartItems := make([]dto.CartItemDto, len(activeItems))

	var wg sync.WaitGroup

	for i, item := range activeItems {
		wg.Add(1)

		go func(idx int, cItem database.CartItem) {
			defer wg.Done()

			// generate product image url
			itemImageUrl, err := h.fileRepo.GeneratePresignUrl(ctx, cItem.Product.ImageObjKey)
			if err != nil {
				fmt.Printf("Error generating URL for %s: %v\n", cItem.Product.ID, err)
				itemImageUrl = ""
			}

			// get available in stock
			available, err := h.productRepo.GetProductStock(ctx, cItem.ProductID)
			if err != nil {
				fmt.Printf("Error getting available amount for product with id: %v", cItem.ProductID)
				available = 0
			}

			// Map to DTO
			cartItems[idx] = dto.CartItemDto{
				ProductID:   cItem.Product.ID,
				Name:        cItem.Product.Name,
				Description: cItem.Product.Description,
				Price:       cItem.Product.Price,
				ImageURL:    itemImageUrl,
				Quantity:    cItem.Quantity,
				Available:   available,
				Category: catDto.CategoryDto{
					ID:   cItem.Product.Category.ID,
					Name: cItem.Product.Category.Name,
				},
			}
		}(i, item)
	}

	wg.Wait()

	return &dto.GetCartByUserIdOutputDto{
		Body: dto.GetCartByUserIdOutputDtoBody{
			CartID:    cart.ID,
			CartItems: cartItems,
		},
	}, nil
}

// Create/Add Item
func (h *cartHandlerImpl) CreateCartItem(ctx context.Context, input *dto.CreateCartItemInputDto) (*dto.CreateCartItemOutputDto, error) {
	// Find cart first
	cart, err := h.cartRepo.GetCartByUserId(ctx, input.UserID)
	if err != nil {
		return nil, huma.Error404NotFound("Cart not found", err)
	}

	// Fetch actual product stock
	availableStock, err := h.productRepo.GetProductStock(ctx, input.Body.ProductID)
	if err != nil {
		return nil, huma.Error404NotFound("Product not found", err)
	}

	// Check if item already exists in the cart
	existItem, err := h.cartRepo.GetCartItem(ctx, cart.ID, input.Body.ProductID)

	// Item already exists in cart. Add quantity to the cart instead
	if err == nil && existItem != nil {
		newTotalQuantity := existItem.Quantity + input.Body.Quantity

		// Validate stock for the new total
		if newTotalQuantity > availableStock {
			errMsg := fmt.Sprintf("Cannot add more to cart: Only %d left in stock (you have %d in cart)", availableStock, existItem.Quantity)
			return nil, huma.Error400BadRequest(errMsg)
		}

		editReq := &dto.EditCartItemInputDto{}
		editReq.UserID = input.UserID
		editReq.Body.ProductID = input.Body.ProductID
		editReq.Body.Quantity = newTotalQuantity

		_, editErr := h.EditCartItem(ctx, editReq)
		if editErr != nil {
			return nil, editErr
		}

		// Return the existing ID
		return &dto.CreateCartItemOutputDto{
			Body: dto.CreateCartItemOutputDtoBody{
				CartItemID: existItem.ID,
			},
		}, nil
	}

	// Item not exist in cart. Create a new one.
	if input.Body.Quantity > availableStock {
		errMsg := fmt.Sprintf("insufficient stock: only %d available", availableStock)
		return nil, huma.Error400BadRequest(errMsg)
	}

	cartItem := &database.CartItem{
		CartID:    cart.ID,
		ProductID: input.Body.ProductID,
		Quantity:  input.Body.Quantity,
	}

	// Create a new item
	itemId, err := h.cartRepo.CreateCartItem(ctx, cartItem)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create cart item", err)
	}

	res := &dto.CreateCartItemOutputDto{
		Body: dto.CreateCartItemOutputDtoBody{
			CartItemID: itemId,
		},
	}

	return res, nil
}

// Edit Item
func (h *cartHandlerImpl) EditCartItem(ctx context.Context, input *dto.EditCartItemInputDto) (*dto.EditCartItemOutputDto, error) {
	cart, err := h.cartRepo.GetCartByUserId(ctx, input.UserID)
	if err != nil {
		return nil, huma.Error404NotFound("Cart not found", err)
	}

	err = h.cartRepo.EditCartItem(ctx, cart.ID, input.Body.ProductID, input.Body.Quantity)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update cart item", err)
	}

	resp := &dto.EditCartItemOutputDto{}
	resp.Body.Message = "Cart item updated successfully"
	return resp, nil
}

// Delete Item(s)
func (h *cartHandlerImpl) DeleteCartItems(ctx context.Context, input *dto.DeleteCartItemsInputDto) (*dto.DeleteCartItemsOutputDto, error) {
	cart, err := h.cartRepo.GetCartByUserId(ctx, input.UserID)
	if err != nil {
		return nil, huma.Error404NotFound("Cart not found", err)
	}

	err = h.cartRepo.RemoveCartItems(ctx, cart.ID, input.Body.ProductIDs)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete cart items", err)
	}

	res := &dto.DeleteCartItemsOutputDto{
		Body: dto.DeleteCartItemsOutputDtoBody{
			Message: "Cart item(s) deleted successfully",
		},
	}
	return res, nil
}
