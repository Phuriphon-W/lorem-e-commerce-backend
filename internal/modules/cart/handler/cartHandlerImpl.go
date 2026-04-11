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

	"github.com/danielgtaylor/huma/v2"
)

type cartHandlerImpl struct {
	cartRepo repository.CartRepository
	fileRepo fileRepo.ObjectStorage
}

func NewCartHandler(cartRepo repository.CartRepository, fileRepo fileRepo.FileRepository) CartHandler {
	return &cartHandlerImpl{
		cartRepo: cartRepo,
		fileRepo: fileRepo,
	}
}

// Get Cart
func (h *cartHandlerImpl) GetCartByUserId(ctx context.Context, input *dto.GetCartByUserIdInputDto) (*dto.GetCartByUserIdOutputDto, error) {
	cart, err := h.cartRepo.GetCartByUserId(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Cart not found", err)
	}

	cartItems := make([]dto.CartItemDto, len(cart.CartItems))

	var wg sync.WaitGroup

	for i, item := range cart.CartItems {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// generate product image url
			itemImageUrl, err := h.fileRepo.GeneratePresignUrl(ctx, item.Product.ImageObjKey)
			if err != nil {
				fmt.Printf("Error generating URL for %s: %v\n", item.Product.ID, err)
				itemImageUrl = ""
			}

			// get available in stock
			available, err := h.cartRepo.GetProductStock(ctx, item.ProductID)
			if err != nil {
				fmt.Printf("Error getting available amount for product with id: %v", item.ProductID)
				available = 0
			}

			// Map to DTO
			cartItems[i] = dto.CartItemDto{
				ProductID:   item.Product.ID,
				Name:        item.Product.Name,
				Description: item.Product.Description,
				Price:       item.Product.Price,
				ImageURL:    itemImageUrl,
				Quantity:    item.Quantity,
				Available:   available,
				Category: catDto.CategoryDto{
					ID:   item.Product.Category.ID,
					Name: item.Product.Category.Name,
				},
			}
		}()
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
	availableStock, err := h.cartRepo.GetProductStock(ctx, input.Body.ProductID)
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
