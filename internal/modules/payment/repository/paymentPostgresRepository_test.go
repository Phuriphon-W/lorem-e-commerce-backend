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

type PaymentRepositoryTestSuite struct {
	suite.Suite
	mockDB      *database.MockDatabase
	paymentRepo PaymentRepository
	ctx         context.Context
}

func (s *PaymentRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.paymentRepo = NewPaymentPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *PaymentRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

// ────────────────────────────────────────────────────────────
// TestCreatePayment
// ────────────────────────────────────────────────────────────

func (s *PaymentRepositoryTestSuite) TestCreatePayment() {
	paymentID := uuid.New()
	orderID := uuid.New()
	userID := uuid.New()

	testCases := []struct {
		name    string
		payment *database.Payment
		setup   func()
		wantErr error
		verify  func(uuid.UUID)
	}{
		{
			name: "Success - inserts payment",
			payment: &database.Payment{
				OrderID:       orderID,
				UserID:        userID,
				PaymentMethod: "card",
				PaymentAmount: 100.00,
				PaymentStatus: "paid",
			},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "payments"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paymentID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
			verify: func(id uuid.UUID) {
				s.Equal(paymentID, id)
			},
		},
		{
			name: "Failure - database insert error triggers rollback",
			payment: &database.Payment{
				OrderID:       orderID,
				UserID:        userID,
				PaymentMethod: "card",
				PaymentAmount: 100.00,
				PaymentStatus: "paid",
			},
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "payments"`)).
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
			s.SetupTest()
			tc.setup()
			id, err := s.paymentRepo.CreatePayment(s.ctx, tc.payment)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(id)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetUserPaymentByOrderID
// ────────────────────────────────────────────────────────────

func (s *PaymentRepositoryTestSuite) TestGetUserPaymentByOrderID() {
	paymentID := uuid.New()
	orderID := uuid.New()
	userID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(*database.Payment)
	}{
		{
			name: "Success - returns payment",
			setup: func() {
				paymentRows := sqlmock.NewRows([]string{"id", "order_id", "user_id", "payment_method", "payment_amount", "payment_status"}).
					AddRow(paymentID, orderID, userID, "card", 100.00, "paid")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "payments" WHERE`).
					WithArgs(userID, orderID, 1).
					WillReturnRows(paymentRows)
			},
			wantErr: nil,
			verify: func(p *database.Payment) {
				s.NotNil(p)
				s.Equal(paymentID, p.ID)
				s.Equal(orderID, p.OrderID)
				s.Equal(userID, p.UserID)
				s.Equal("card", p.PaymentMethod)
				s.Equal(100.00, p.PaymentAmount)
				s.Equal("paid", p.PaymentStatus)
			},
		},
		{
			name: "Failure - record not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "payments" WHERE`).
					WithArgs(userID, orderID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(p *database.Payment) {
				s.Nil(p)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			payment, err := s.paymentRepo.GetUserPaymentByOrderID(s.ctx, orderID, userID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(payment)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetUserPaymentsByUserID
// ────────────────────────────────────────────────────────────

func (s *PaymentRepositoryTestSuite) TestGetUserPaymentsByUserID() {
	userID := uuid.New()
	paymentID1 := uuid.New()
	paymentID2 := uuid.New()
	orderID1 := uuid.New()
	orderID2 := uuid.New()

	testCases := []struct {
		name     string
		page     int64
		pageSize int64
		status   string
		orderBy  string
		setup    func()
		wantErr  error
		verify   func([]database.Payment, int64)
	}{
		{
			name:     "Success - with status filter + custom order",
			page:     1,
			pageSize: 10,
			status:   "paid",
			orderBy:  "created_at ASC",
			setup: func() {
				// Count Query
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "payments"`).
					WithArgs(userID, "paid").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

				// Find Query
				paymentRows := sqlmock.NewRows([]string{"id", "order_id", "user_id", "payment_method", "payment_amount", "payment_status"}).
					AddRow(paymentID1, orderID1, userID, "card", 100.00, "paid").
					AddRow(paymentID2, orderID2, userID, "card", 150.00, "paid")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "payments"`).
					WithArgs(userID, "paid", 10).
					WillReturnRows(paymentRows)
			},
			wantErr: nil,
			verify: func(payments []database.Payment, total int64) {
				s.Len(payments, 2)
				s.Equal(int64(2), total)
				s.Equal(paymentID1, payments[0].ID)
				s.Equal(paymentID2, payments[1].ID)
			},
		},
		{
			name:     "Success - no status filter, default order",
			page:     2,
			pageSize: 5,
			status:   "",
			orderBy:  "",
			setup: func() {
				// Count Query
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "payments"`).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

				// Find Query
				paymentRows := sqlmock.NewRows([]string{"id", "order_id", "user_id", "payment_method", "payment_amount", "payment_status"}).
					AddRow(paymentID1, orderID1, userID, "card", 100.00, "pending")
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "payments"`).
					WithArgs(userID, 5, 5). // limit 5, offset 5
					WillReturnRows(paymentRows)
			},
			wantErr: nil,
			verify: func(payments []database.Payment, total int64) {
				s.Len(payments, 1)
				s.Equal(int64(12), total)
				s.Equal(paymentID1, payments[0].ID)
			},
		},
		{
			name:     "Failure - count query fails",
			page:     1,
			pageSize: 10,
			status:   "",
			orderBy:  "",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "payments"`).
					WithArgs(userID).
					WillReturnError(errors.New("count query failed"))
			},
			wantErr: errors.New("count query failed"),
			verify: func(payments []database.Payment, total int64) {
				s.Nil(payments)
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
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "payments"`).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "payments"`).
					WithArgs(userID, 10).
					WillReturnError(errors.New("find query failed"))
			},
			wantErr: errors.New("find query failed"),
			verify: func(payments []database.Payment, total int64) {
				s.Nil(payments)
				s.Equal(int64(0), total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			payments, total, err := s.paymentRepo.GetUserPaymentsByUserID(s.ctx, userID, tc.page, tc.pageSize, tc.orderBy, tc.status)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(payments, total)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestUpdatePaymentStatusByOrderID
// ────────────────────────────────────────────────────────────

func (s *PaymentRepositoryTestSuite) TestUpdatePaymentStatusByOrderID() {
	orderID := uuid.New()

	testCases := []struct {
		name    string
		status  string
		setup   func()
		wantErr error
	}{
		{
			name:   "Success - updates payment status to paid",
			status: "paid",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "payments"`)).
					WithArgs("paid", sqlmock.AnyArg(), orderID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:   "Failure - update fails due to database error",
			status: "paid",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "payments"`)).
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
			err := s.paymentRepo.UpdatePaymentStatusByOrderID(s.ctx, orderID, tc.status)

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

func TestPaymentRepository(t *testing.T) {
	suite.Run(t, new(PaymentRepositoryTestSuite))
}
