package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"lorem-backend/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type OrderRepositoryTestSuite struct {
	suite.Suite
	mockDB    *database.MockDatabase
	orderRepo OrderRepository
	ctx       context.Context
}

func (s *OrderRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.orderRepo = NewOrderPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *OrderRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

// ────────────────────────────────────────────────────────────
// TestCreateOrder
// ────────────────────────────────────────────────────────────

func (s *OrderRepositoryTestSuite) TestCreateOrder() {
	userID := uuid.New()
	orderID := uuid.New()
	productID := uuid.New()
	orderItemID := uuid.New()

	testCases := []struct {
		name    string
		order   *database.Order
		setup   func()
		wantErr error
		verify  func(uuid.UUID)
	}{
		{
			name: "Success - inserts order and association order items",
			order: &database.Order{
				UserID:      userID,
				TotalPrice:  120.00,
				OrderStatus: database.Pending,
				OrderItems: []database.OrderItem{
					{
						ProductID:       productID,
						PriceAtPurchase: 60.00,
						Quantity:        2,
					},
				},
			},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				// GORM inserts parent order
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "orders"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(orderID))
				// GORM inserts child order item
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "order_items"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(orderItemID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
			verify: func(id uuid.UUID) {
				s.Equal(orderID, id)
			},
		},
		{
			name: "Failure - cannot create an order without items",
			order: &database.Order{
				UserID:      userID,
				TotalPrice:  50.00,
				OrderStatus: database.Pending,
			},
			setup:   func() {},
			wantErr: errors.New("cannot create an order without items"),
			verify: func(id uuid.UUID) {
				s.Equal(uuid.Nil, id)
			},
		},
		{
			name: "Failure - database insert error triggers rollback",
			order: &database.Order{
				UserID:      userID,
				TotalPrice:  50.00,
				OrderStatus: database.Pending,
				OrderItems: []database.OrderItem{
					{
						ProductID:       productID,
						PriceAtPurchase: 50.00,
						Quantity:        1,
					},
				},
			},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "orders"`)).
					WillReturnError(errors.New("db insert fail"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("db insert fail"),
			verify: func(id uuid.UUID) {
				s.Equal(uuid.Nil, id)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			id, err := s.orderRepo.CreateOrder(s.ctx, tc.order)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(id)
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetOrdersByUserID
// ────────────────────────────────────────────────────────────

func (s *OrderRepositoryTestSuite) TestGetOrdersByUserID() {
	userID := uuid.New()
	orderID := uuid.New()
	productID := uuid.New()
	orderItemID := uuid.New()

	testCases := []struct {
		name     string
		page     int64
		pageSize int64
		status   string
		orderBy  string
		setup    func()
		wantErr  error
		verify   func([]database.Order, int64)
	}{
		{
			name:     "Success - retrieves orders with count and preloads",
			page:     1,
			pageSize: 10,
			status:   "pending",
			orderBy:  "created_at ASC",
			setup: func() {
				// Count Query
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "orders"`).
					WithArgs(userID, "pending").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				// Find Query
				orderRows := sqlmock.NewRows([]string{"id", "user_id", "total_price", "order_status"}).
					AddRow(orderID, userID, float32(100.00), database.Pending)
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "orders"`).
					WithArgs(userID, "pending", 10).
					WillReturnRows(orderRows)

				// Preload OrderItems
				itemRows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "price_at_purchase", "quantity"}).
					AddRow(orderItemID, orderID, productID, float32(50.00), uint(2))
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "order_items"`).
					WithArgs(orderID).
					WillReturnRows(itemRows)

				// Preload OrderItems.Product
				productRows := sqlmock.NewRows([]string{"id", "name", "price", "obj_key"}).
					AddRow(productID, "Hoodie", float32(50.00), "hoodie.jpg")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "products"`).
					WithArgs(productID).
					WillReturnRows(productRows)
			},
			wantErr: nil,
			verify: func(orders []database.Order, total int64) {
				s.Len(orders, 1)
				s.Equal(int64(1), total)
				s.Equal(orderID, orders[0].ID)
				s.Len(orders[0].OrderItems, 1)
				s.Equal(productID, orders[0].OrderItems[0].ProductID)
				s.Equal("Hoodie", orders[0].OrderItems[0].Product.Name)
			},
		},
		{
			name:     "Success - default sort and no status filter",
			page:     2,
			pageSize: 5,
			status:   "",
			orderBy:  "",
			setup: func() {
				// Count Query
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "orders"`).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

				// Find Query
				orderRows := sqlmock.NewRows([]string{"id", "user_id", "total_price", "order_status"}).
					AddRow(orderID, userID, float32(40.00), database.Pending)
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "orders"`).
					WithArgs(userID, 5, 5). // Offset: (2-1)*5 = 5
					WillReturnRows(orderRows)

				// Preload OrderItems
				itemRows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "price_at_purchase", "quantity"}).
					AddRow(orderItemID, orderID, productID, float32(40.00), uint(1))
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "order_items"`).
					WithArgs(orderID).
					WillReturnRows(itemRows)

				// Preload OrderItems.Product
				productRows := sqlmock.NewRows([]string{"id", "name", "price", "obj_key"}).
					AddRow(productID, "Jeans", float32(40.00), "jeans.jpg")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "products"`).
					WithArgs(productID).
					WillReturnRows(productRows)
			},
			wantErr: nil,
			verify: func(orders []database.Order, total int64) {
				s.Len(orders, 1)
				s.Equal(int64(12), total)
				s.Len(orders[0].OrderItems, 1)
				s.Equal(productID, orders[0].OrderItems[0].ProductID)
				s.Equal("Jeans", orders[0].OrderItems[0].Product.Name)
			},
		},
		{
			name:     "Failure - count query fails",
			page:     1,
			pageSize: 10,
			status:   "",
			orderBy:  "",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "orders"`).
					WithArgs(userID).
					WillReturnError(errors.New("count query failed"))
			},
			wantErr: errors.New("count query failed"),
			verify: func(orders []database.Order, total int64) {
				s.Nil(orders)
				s.Equal(int64(0), total)
			},
		},
		{
			name:     "Failure - find query fails",
			page:     1,
			pageSize: 10,
			status:   "",
			orderBy:  "",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "orders"`).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "orders"`).
					WithArgs(userID, 10).
					WillReturnError(errors.New("find query failed"))
			},
			wantErr: errors.New("find query failed"),
			verify: func(orders []database.Order, total int64) {
				s.Nil(orders)
				s.Equal(int64(0), total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // Reset sqlmock expectations for each subtest case
			tc.setup()
			orders, total, err := s.orderRepo.GetOrdersByUserID(s.ctx, userID, tc.page, tc.pageSize, tc.status, tc.orderBy)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(orders, total)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetOrderByID
// ────────────────────────────────────────────────────────────

func (s *OrderRepositoryTestSuite) TestGetOrderByID() {
	orderID := uuid.New()
	userID := uuid.New()
	productID := uuid.New()
	orderItemID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(*database.Order)
	}{
		{
			name: "Success - returns order with preloads",
			setup: func() {
				// Find Query
				orderRows := sqlmock.NewRows([]string{"id", "user_id", "total_price"}).
					AddRow(orderID, userID, float32(45.00))
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "orders" WHERE id =`).
					WithArgs(orderID, 1).
					WillReturnRows(orderRows)

				// Preload OrderItems
				itemRows := sqlmock.NewRows([]string{"id", "order_id", "product_id"}).
					AddRow(orderItemID, orderID, productID)
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "order_items"`).
					WithArgs(orderID).
					WillReturnRows(itemRows)

				// Preload OrderItems.Product
				productRows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(productID, "Shoes")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "products"`).
					WithArgs(productID).
					WillReturnRows(productRows)
			},
			wantErr: nil,
			verify: func(o *database.Order) {
				s.NotNil(o)
				s.Equal(orderID, o.ID)
				s.Len(o.OrderItems, 1)
				s.Equal("Shoes", o.OrderItems[0].Product.Name)
			},
		},
		{
			name: "Failure - record not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "orders" WHERE id =`).
					WithArgs(orderID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(o *database.Order) {
				s.Nil(o)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			order, err := s.orderRepo.GetOrderByID(s.ctx, orderID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(order)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestUpdateOrderStatus
// ────────────────────────────────────────────────────────────

func (s *OrderRepositoryTestSuite) TestUpdateOrderStatus() {
	orderID := uuid.New()

	testCases := []struct {
		name    string
		status  database.OrderStatus
		setup   func()
		wantErr error
	}{
		{
			name:   "Success - updates order status to Paid",
			status: database.Paid,
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "orders"`)).
					WithArgs(database.Paid, sqlmock.AnyArg(), orderID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:   "Failure - updates fails due to database error",
			status: database.Completed,
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "orders"`)).
					WillReturnError(errors.New("update status error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update status error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			err := s.orderRepo.UpdateOrderStatus(s.ctx, orderID, tc.status)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestUpdateOrderSession
// ────────────────────────────────────────────────────────────

func (s *OrderRepositoryTestSuite) TestUpdateOrderSession() {
	orderID := uuid.New()
	sessionID := "sess_12345"
	sessionURL := "https://checkout.stripe.com/pay/sess_12345"
	expiresAt := time.Now().Add(1 * time.Hour)

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - updates stripe session fields",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "orders"`)).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - updates fails",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "orders"`)).
					WillReturnError(errors.New("stripe updates error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("stripe updates error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			err := s.orderRepo.UpdateOrderSession(s.ctx, orderID, sessionID, sessionURL, &expiresAt)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			s.TearDownTest()
		})
	}
}

func TestOrderRepository(t *testing.T) {
	suite.Run(t, new(OrderRepositoryTestSuite))
}

func (s *OrderRepositoryTestSuite) TestGetOrdersCount() {
	s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "orders"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(200))

	count, err := s.orderRepo.GetOrdersCount(s.ctx)
	s.NoError(err)
	s.Equal(int64(200), count)
}
