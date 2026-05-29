package repository

import (
	"context"
	"errors"
	"lorem-backend/internal/database"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// Define Test Suite
type UserRepositoryTestSuite struct {
	suite.Suite
	mockDB   *database.MockDatabase
	userRepo UserRepository
	ctx      context.Context
}

// Set Up
func (s *UserRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.userRepo = NewUserPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

// Tear Down
func (s *UserRepositoryTestSuite) TearDownTest() {
	// Automatically checks expectations after every single test ends
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

func (s *UserRepositoryTestSuite) TestGetUsers() {
	mockUsers := s.mockDB.Mock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at"}).
		AddRow(uuid.New(), "User1", "user1@mail.com", "John", "Doe", time.Now()).
		AddRow(uuid.New(), "User2", "user2@mail.com", "Jane", "Doe", time.Now()).
		AddRow(uuid.New(), "User3", "user3@mail.com", "Jack", "Doe", time.Now())

	s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."deleted_at" IS NULL`)).
		WillReturnRows(mockUsers)

	users, err := s.userRepo.GetUsers(s.ctx)

	s.NoError(err)
	s.Len(users, 3)
	s.Equal("John", users[0].FirstName)
	s.Equal("user3@mail.com", users[2].Email)
	s.Equal("User2", users[1].Username)
}

func (s *UserRepositoryTestSuite) TestGetUserByID() {
	expectedQuery := regexp.QuoteMeta(
		`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`,
	)

	testCases := []struct {
		name    string
		setup   func(id uuid.UUID)
		wantErr error
	}{
		{
			name: "Success - returns user when ID exists",
			setup: func(id uuid.UUID) {
				rows := s.mockDB.Mock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at", "updated_at", "deleted_at"}).
					AddRow(id, "john_doe", "john@example.com", "John", "Doe", time.Now(), time.Now(), nil)
				s.mockDB.Mock.ExpectQuery(expectedQuery).WithArgs(id, 1).WillReturnRows(rows)
			},
		},
		{
			name: "Failure - returns ErrRecordNotFound when ID does not exist",
			setup: func(id uuid.UUID) {
				s.mockDB.Mock.ExpectQuery(expectedQuery).WithArgs(id, 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "Failure - returns error on database failure",
			setup: func(id uuid.UUID) {
				s.mockDB.Mock.ExpectQuery(expectedQuery).WithArgs(id, 1).WillReturnError(errors.New("connection refused"))
			},
			wantErr: errors.New("connection refused"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			id := uuid.New()
			tc.setup(id)

			user, err := s.userRepo.GetUserByID(s.ctx, id)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.EqualError(err, tc.wantErr.Error())
				s.Nil(user)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(user)
				s.Equal(id, user.ID)
				s.Equal("john_doe", user.Username)
				s.Equal("John", user.FirstName)
				s.Equal("Doe", user.LastName)
				s.Equal("john@example.com", user.Email)
			}
		})
	}
}

func TestUserRepository(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
