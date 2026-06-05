package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"lorem-backend/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type CartRepositoryTestSuite struct {
	suite.Suite
	mockDB   *database.MockDatabase
	cartRepo CartRepository
	ctx      context.Context
}

func (s *CartRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.cartRepo = NewCartPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *CartRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

// ────────────────────────────────────────────────────────────
// TestGetCartByUserId
// ────────────────────────────────────────────────────────────

func (s *CartRepositoryTestSuite) TestGetCartByUserId() {
	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()
	catID := uuid.New()
	cartItemID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(*database.Cart)
	}{
		{
			name: "Success - returns cart with preloaded items, products, and categories",
			setup: func() {
				// 1. SELECT carts WHERE user_id = ?
				cartRows := sqlmock.NewRows([]string{"id", "user_id"}).
					AddRow(cartID, userID)
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "carts" WHERE user_id = \$1`).
					WithArgs(userID, 1).
					WillReturnRows(cartRows)

				// 2. Preload CartItems (ORDER BY created_at ASC)
				cartItemRows := sqlmock.NewRows([]string{"id", "cart_id", "product_id", "quantity"}).
					AddRow(cartItemID, cartID, productID, 3)
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "cart_items" WHERE`).
					WithArgs(cartID).
					WillReturnRows(cartItemRows)

				// 3. Preload CartItems.Product
				productRows := sqlmock.NewRows([]string{"id", "name", "description", "price", "available", "obj_key", "category_id"}).
					AddRow(productID, "Sweater", "A warm sweater", float32(49.99), 10, "images/sweater.png", catID)
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE`).
					WithArgs(productID).
					WillReturnRows(productRows)

				// 4. Preload CartItems.Product.Category
				categoryRows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(catID, "Apparel")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories" WHERE`).
					WithArgs(catID).
					WillReturnRows(categoryRows)
			},
			wantErr: nil,
			verify: func(cart *database.Cart) {
				s.NotNil(cart)
				s.Equal(cartID, cart.ID)
				s.Equal(userID, cart.UserID)
				s.Len(cart.CartItems, 1)
				s.Equal(productID, cart.CartItems[0].ProductID)
				s.Equal(uint(3), cart.CartItems[0].Quantity)
				s.Equal("Sweater", cart.CartItems[0].Product.Name)
				s.Equal("Apparel", cart.CartItems[0].Product.Category.Name)
			},
		},
		{
			name: "Failure - cart not found (record not found error)",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "carts" WHERE user_id = \$1`).
					WithArgs(userID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(cart *database.Cart) {
				s.Nil(cart)
			},
		},
		{
			name: "Failure - database query error",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "carts" WHERE user_id = \$1`).
					WithArgs(userID, 1).
					WillReturnError(errors.New("db query failed"))
			},
			wantErr: errors.New("db query failed"),
			verify: func(cart *database.Cart) {
				s.Nil(cart)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			cart, err := s.cartRepo.GetCartByUserId(s.ctx, userID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(cart)
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestCreateCartItem
// ────────────────────────────────────────────────────────────

func (s *CartRepositoryTestSuite) TestCreateCartItem() {
	cartID := uuid.New()
	productID := uuid.New()
	cartItemID := uuid.New()

	cartItem := &database.CartItem{
		CartID:    cartID,
		ProductID: productID,
		Quantity:  2,
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - inserts cart item",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cart_items"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(cartItemID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - insert error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cart_items"`)).
					WillReturnError(errors.New("db insert error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("db insert error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			id, err := s.cartRepo.CreateCartItem(s.ctx, cartItem)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				s.Equal(uuid.Nil, id)
			} else {
				s.Require().NoError(err)
				s.NotEqual(uuid.Nil, id)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetCartItem
// ────────────────────────────────────────────────────────────

func (s *CartRepositoryTestSuite) TestGetCartItem() {
	cartID := uuid.New()
	productID := uuid.New()
	cartItemID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(*database.CartItem)
	}{
		{
			name: "Success - returns cart item",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "cart_id", "product_id", "quantity"}).
					AddRow(cartItemID, cartID, productID, 5)
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "cart_items" WHERE \(cart_id = \$1 AND product_id = \$2\)`).
					WithArgs(cartID, productID, 1).
					WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(item *database.CartItem) {
				s.NotNil(item)
				s.Equal(cartItemID, item.ID)
				s.Equal(cartID, item.CartID)
				s.Equal(productID, item.ProductID)
				s.Equal(uint(5), item.Quantity)
			},
		},
		{
			name: "Failure - cart item not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "cart_items" WHERE \(cart_id = \$1 AND product_id = \$2\)`).
					WithArgs(cartID, productID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(item *database.CartItem) {
				s.Nil(item)
			},
		},
		{
			name: "Failure - database error",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "cart_items" WHERE \(cart_id = \$1 AND product_id = \$2\)`).
					WithArgs(cartID, productID, 1).
					WillReturnError(errors.New("db error"))
			},
			wantErr: errors.New("db error"),
			verify: func(item *database.CartItem) {
				s.Nil(item)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			item, err := s.cartRepo.GetCartItem(s.ctx, cartID, productID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(item)
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestEditCartItem
// ────────────────────────────────────────────────────────────

func (s *CartRepositoryTestSuite) TestEditCartItem() {
	cartID := uuid.New()
	productID := uuid.New()

	testCases := []struct {
		name     string
		quantity uint
		setup    func()
		wantErr  error
	}{
		{
			name:     "Success - updates quantity",
			quantity: 5,
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "cart_items"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:     "Failure - database update error",
			quantity: 5,
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "cart_items"`)).
					WillReturnError(errors.New("update error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.cartRepo.EditCartItem(s.ctx, cartID, productID, tc.quantity)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestRemoveCartItems
// ────────────────────────────────────────────────────────────

func (s *CartRepositoryTestSuite) TestRemoveCartItems() {
	cartID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	testCases := []struct {
		name       string
		productIDs []uuid.UUID
		setup      func()
		wantErr    error
	}{
		{
			name:       "Success - hard deletes cart items",
			productIDs: []uuid.UUID{productID1, productID2},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cart_items" WHERE`)).
					WillReturnResult(sqlmock.NewResult(1, 2))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:       "Failure - database delete error",
			productIDs: []uuid.UUID{productID1},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cart_items" WHERE`)).
					WillReturnError(errors.New("delete error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("delete error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.cartRepo.RemoveCartItems(s.ctx, cartID, tc.productIDs)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestCartRepository(t *testing.T) {
	suite.Run(t, new(CartRepositoryTestSuite))
}
