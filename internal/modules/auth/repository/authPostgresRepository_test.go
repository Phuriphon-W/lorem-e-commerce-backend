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

type AuthRepositoryTestSuite struct {
	suite.Suite
	mockDB   *database.MockDatabase
	authRepo AuthRepository
	ctx      context.Context
}

func (s *AuthRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.authRepo = NewAuthPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *AuthRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

func (s *AuthRepositoryTestSuite) TestRegisterUser() {
	userID := uuid.New()
	user := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:     "johndoe",
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john@example.com",
		PasswordHash: "hashed_password",
	}

	testCases := []struct {
		name      string
		setup     func()
		wantErr   error
		expectUID uuid.UUID
	}{
		{
			name: "Success - registers user and creates cart",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				// GORM INSERT INTO "users"
				s.mockDB.Mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
				// GORM INSERT INTO "carts"
				s.mockDB.Mock.ExpectQuery(`INSERT INTO "carts"`).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr:   nil,
			expectUID: userID,
		},
		{
			name: "Failure - user creation db error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnError(errors.New("insert user failed"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr:   errors.New("insert user failed"),
			expectUID: uuid.Nil,
		},
		{
			name: "Failure - cart creation db error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
				s.mockDB.Mock.ExpectQuery(`INSERT INTO "carts"`).
					WillReturnError(errors.New("insert cart failed"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr:   errors.New("insert cart failed"),
			expectUID: uuid.Nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()

			uid, username, err := s.authRepo.RegisterUser(s.ctx, user)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				s.Equal(uuid.Nil, uid)
				s.Empty(username)
			} else {
				s.Require().NoError(err)
				s.Equal(tc.expectUID, uid)
				s.Equal("johndoe", username)
			}
		})
	}
}

func (s *AuthRepositoryTestSuite) TestGetUserByEmail() {
	email := "john@example.com"
	userID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(res *struct {
			ID           uuid.UUID
			Username     string
			PasswordHash string
			IsAdmin      bool
		})
	}{
		{
			name: "Success - returns user details by email",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "is_admin"}).
					AddRow(userID, "johndoe", "hashed_password", false)
				s.mockDB.Mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*`).
					WithArgs(email, 1).
					WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(res *struct {
				ID           uuid.UUID
				Username     string
				PasswordHash string
				IsAdmin      bool
			}) {
				s.NotNil(res)
				s.Equal(userID, res.ID)
				s.Equal("johndoe", res.Username)
				s.Equal("hashed_password", res.PasswordHash)
				s.Equal(false, res.IsAdmin)
			},
		},
		{
			name: "Failure - record not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*`).
					WithArgs(email, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(res *struct {
				ID           uuid.UUID
				Username     string
				PasswordHash string
				IsAdmin      bool
			}) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - database connection error",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*`).
					WithArgs(email, 1).
					WillReturnError(errors.New("db error"))
			},
			wantErr: errors.New("db error"),
			verify: func(res *struct {
				ID           uuid.UUID
				Username     string
				PasswordHash string
				IsAdmin      bool
			}) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()

			res, err := s.authRepo.GetUserByEmail(s.ctx, email)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				tc.verify(res)
			} else {
				s.Require().NoError(err)
				tc.verify(res)
			}
		})
	}
}

func (s *AuthRepositoryTestSuite) TestGetUserByUsername() {
	username := "johndoe"
	userID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func(res *struct {
			ID       uuid.UUID
			Username string
		})
	}{
		{
			name: "Success - returns user details by username",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "username"}).
					AddRow(userID, username)
				s.mockDB.Mock.ExpectQuery(`SELECT .* FROM "users" WHERE username = \$1.*`).
					WithArgs(username, 1).
					WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(res *struct {
				ID       uuid.UUID
				Username string
			}) {
				s.NotNil(res)
				s.Equal(userID, res.ID)
				s.Equal(username, res.Username)
			},
		},
		{
			name: "Failure - record not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT .* FROM "users" WHERE username = \$1.*`).
					WithArgs(username, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
			verify: func(res *struct {
				ID       uuid.UUID
				Username string
			}) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()

			res, err := s.authRepo.GetUserByUsername(s.ctx, username)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				tc.verify(res)
			} else {
				s.Require().NoError(err)
				tc.verify(res)
			}
		})
	}
}

func (s *AuthRepositoryTestSuite) TestUpdatePassword() {
	userID := uuid.New()
	newHash := "new_hashed_password"

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - updates user password",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "password_hash"=$1 WHERE id = $2`)).
					WithArgs(newHash, userID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - database exec error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "password_hash"=$1 WHERE id = $2`)).
					WithArgs(newHash, userID).
					WillReturnError(errors.New("update failed"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update failed"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()

			err := s.authRepo.UpdatePassword(s.ctx, userID, newHash)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestAuthRepository(t *testing.T) {
	suite.Run(t, new(AuthRepositoryTestSuite))
}
