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

type ProductRepositoryTestSuite struct {
	suite.Suite
	mockDB      *database.MockDatabase
	productRepo ProductRepository
	ctx         context.Context
}

func (s *ProductRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.productRepo = NewProductPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *ProductRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

func (s *ProductRepositoryTestSuite) TestCreateProduct() {
	productID := uuid.New()
	product := &database.Product{
		Name:        "Cool Cap",
		Description: "A cool baseball cap",
		Price:       15.99,
		Available:   50,
		ImageObjKey: "caps/cool.png",
		CategoryID:  uuid.New(),
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - inserts product",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "products"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(productID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - insert error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "products"`)).
					WillReturnError(errors.New("db error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			id, err := s.productRepo.CreateProduct(s.ctx, product)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				s.Equal(uuid.Nil, id)
			} else {
				s.Require().NoError(err)
				s.Equal(productID, id)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestGetProducts() {
	catID := uuid.New()
	prodID := uuid.New()

	testCases := []struct {
		name      string
		page      int64
		pageSize  int64
		category  string
		search    string
		order     string
		setup     func()
		wantErr   error
		verifyRes func(products []database.Product, total int64)
	}{
		{
			name:     "Success - returns products list",
			page:     1,
			pageSize: 10,
			category: "Shirts",
			search:   "Cotton",
			order:    "products.price ASC",
			setup: func() {
				// Count query expectation
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))
				s.mockDB.Mock.ExpectQuery(`SELECT count\(\*\) FROM "products"`).WillReturnRows(countRows)

				// Find products query expectation
				prodRows := sqlmock.NewRows([]string{"id", "name", "description", "price", "available", "obj_key", "category_id"}).
					AddRow(prodID, "Cotton T-Shirt", "100% cotton", float32(19.99), uint(100), "images/cotton.png", catID)
				s.mockDB.Mock.ExpectQuery(`SELECT "products"\..* FROM "products"`).WillReturnRows(prodRows)

				// Preload category query expectation
				catRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(catID, "Shirts")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories" WHERE "categories"\."id" = \$1`).
					WithArgs(catID).WillReturnRows(catRows)
			},
			wantErr: nil,
			verifyRes: func(products []database.Product, total int64) {
				s.Len(products, 1)
				s.Equal(total, int64(1))
				s.Equal("Cotton T-Shirt", products[0].Name)
				s.Equal("Shirts", products[0].Category.Name)
			},
		},
		{
			name:     "Failure - count error",
			page:     1,
			pageSize: 10,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT count\(\*\) FROM "products"`).
					WillReturnError(errors.New("count failed"))
			},
			wantErr: errors.New("count failed"),
			verifyRes: func(products []database.Product, total int64) {
				s.Nil(products)
				s.Equal(int64(0), total)
			},
		},
		{
			name:     "Failure - find error",
			page:     1,
			pageSize: 10,
			setup: func() {
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(5))
				s.mockDB.Mock.ExpectQuery(`SELECT count\(\*\) FROM "products"`).WillReturnRows(countRows)

				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products"`).
					WillReturnError(errors.New("find query failed"))
			},
			wantErr: errors.New("find query failed"),
			verifyRes: func(products []database.Product, total int64) {
				s.Nil(products)
				s.Equal(int64(0), total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			products, total, err := s.productRepo.GetProducts(s.ctx, tc.page, tc.pageSize, tc.category, tc.search, tc.order)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				tc.verifyRes(products, total)
			} else {
				s.Require().NoError(err)
				tc.verifyRes(products, total)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestGetProductByID() {
	catID := uuid.New()
	prodID := uuid.New()

	testCases := []struct {
		name    string
		id      uuid.UUID
		setup   func()
		wantErr error
	}{
		{
			name: "Success - returns product with category preloaded",
			id:   prodID,
			setup: func() {
				prodRows := sqlmock.NewRows([]string{"id", "name", "description", "price", "available", "obj_key", "category_id"}).
					AddRow(prodID, "Hoodie", "A cool hoodie", float32(59.99), uint(20), "images/hoodie.png", catID)
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1`).
					WithArgs(prodID, 1).WillReturnRows(prodRows)

				catRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(catID, "Outerwear")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories" WHERE "categories"\."id" = \$1`).
					WithArgs(catID).WillReturnRows(catRows)
			},
			wantErr: nil,
		},
		{
			name: "Failure - record not found",
			id:   prodID,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1`).
					WithArgs(prodID, 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			res, err := s.productRepo.GetProductByID(s.ctx, tc.id)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Nil(res)
			} else {
				s.Require().NoError(err)
				s.NotNil(res)
				s.Equal(prodID, res.ID)
				s.Equal("Outerwear", res.Category.Name)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestGetProductsByIDs() {
	id1 := uuid.New()
	id2 := uuid.New()

	testCases := []struct {
		name    string
		ids     []uuid.UUID
		setup   func()
		wantErr error
		verify  func([]database.Product)
	}{
		{
			name: "Success - returns matching products",
			ids:  []uuid.UUID{id1, id2},
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(id1, "Prod 1").
					AddRow(id2, "Prod 2")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id IN \(\$1,\$2\)`).
					WithArgs(id1, id2).WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(res []database.Product) {
				s.Len(res, 2)
				s.Equal(id1, res[0].ID)
				s.Equal(id2, res[1].ID)
			},
		},
		{
			name: "Failure - database query error",
			ids:  []uuid.UUID{id1},
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id IN \(\$1\)`).
					WithArgs(id1).WillReturnError(errors.New("db error"))
			},
			wantErr: errors.New("db error"),
			verify: func(res []database.Product) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			res, err := s.productRepo.GetProductsByIDs(s.ctx, tc.ids)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(res)
		})
	}
}

func (s *ProductRepositoryTestSuite) TestGetProductStock() {
	prodID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(stock uint)
	}{
		{
			name: "Success - returns available stock",
			setup: func() {
				rows := sqlmock.NewRows([]string{"available"}).AddRow(uint(45))
				s.mockDB.Mock.ExpectQuery(`SELECT "available" FROM "products" WHERE id = \$1`).
					WithArgs(prodID, 1).WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(stock uint) {
				s.Equal(uint(45), stock)
			},
		},
		{
			name: "Failure - stock lookup error",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT "available" FROM "products" WHERE id = \$1`).
					WithArgs(prodID, 1).WillReturnError(errors.New("lookup failed"))
			},
			wantErr: errors.New("lookup failed"),
			verify: func(stock uint) {
				s.Equal(uint(0), stock)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			stock, err := s.productRepo.GetProductStock(s.ctx, prodID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(stock)
		})
	}
}

func (s *ProductRepositoryTestSuite) TestUpdateProductByID() {
	prodID := uuid.New()
	updateData := map[string]interface{}{"price": float32(12.50), "available": uint(100)}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - updates attributes",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "products"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - DB execute update error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "products"`)).
					WillReturnError(errors.New("update statement error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update statement error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.productRepo.UpdateProductByID(s.ctx, prodID, updateData)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestDeductProductStocks() {
	prodID := uuid.New()
	deductions := []StockDeduction{
		{
			ProductID: prodID,
			Quantity:  5,
		},
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - locks row and deducts stock",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				// Expect SELECT ... FOR UPDATE
				rows := sqlmock.NewRows([]string{"id", "available"}).AddRow(prodID, uint(10))
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1.*FOR UPDATE`).
					WithArgs(prodID, 1).WillReturnRows(rows)
				// Expect UPDATE available
				s.mockDB.Mock.ExpectExec(`UPDATE "products"`).WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - insufficient stock",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"id", "available"}).AddRow(prodID, uint(3))
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1.*FOR UPDATE`).
					WithArgs(prodID, 1).WillReturnRows(rows)
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("insufficient stock for product"),
		},
		{
			name: "Failure - product not found in transaction",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1.*FOR UPDATE`).
					WithArgs(prodID, 1).WillReturnError(gorm.ErrRecordNotFound)
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "Failure - DB error on update query",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"id", "available"}).AddRow(prodID, uint(10))
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "products" WHERE id = \$1.*FOR UPDATE`).
					WithArgs(prodID, 1).WillReturnRows(rows)
				s.mockDB.Mock.ExpectExec(`UPDATE "products"`).WillReturnError(errors.New("update deduction failed"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update deduction failed"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.productRepo.DeductProductStocks(s.ctx, deductions)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestAddProductStocks() {
	prodID := uuid.New()
	additions := []StockDeduction{
		{
			ProductID: prodID,
			Quantity:  10,
		},
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - increments stock in transaction",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(`UPDATE "products"`).WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - db error on increment",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(`UPDATE "products"`).WillReturnError(errors.New("increment failed"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("increment failed"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.productRepo.AddProductStocks(s.ctx, additions)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ProductRepositoryTestSuite) TestDeleteProductByID() {
	prodID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - soft deletes product",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "products" SET "deleted_at"=`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - database delete error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "products" SET "deleted_at"=`)).
					WillReturnError(errors.New("delete error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("delete error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.productRepo.DeleteProductByID(s.ctx, prodID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestProductRepository(t *testing.T) {
	suite.Run(t, new(ProductRepositoryTestSuite))
}
