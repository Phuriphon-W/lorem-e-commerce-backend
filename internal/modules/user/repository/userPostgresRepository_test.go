package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"lorem-backend/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserPostgresRepository_GetUsers(t *testing.T) {
	mockDB := database.NewMockDatabase(t)
	repo := NewUserPostgresRepository(mockDB)
	ctx := context.Background()

	mockUsers := sqlmock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at"}).
		AddRow(uuid.New().String(), "user1", "user1@example.com", "User", "One", time.Now()).
		AddRow(uuid.New().String(), "user2", "user2@example.com", "User", "Two", time.Now())

	mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."deleted_at" IS NULL`)).
		WillReturnRows(mockUsers)

	users, err := repo.GetUsers(ctx)

	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1", users[0].Username)
	assert.Equal(t, "user2", users[1].Username)

	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}

func TestUserPostgresRepository_GetUserByID(t *testing.T) {
	mockDB := database.NewMockDatabase(t)
	repo := NewUserPostgresRepository(mockDB)
	ctx := context.Background()

	userID := uuid.New()

	mockUser := sqlmock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at"}).
		AddRow(userID.String(), "user1", "user1@example.com", "User", "One", time.Now())

	mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(userID, 1).
		WillReturnRows(mockUser)

	user, err := repo.GetUserByID(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "user1", user.Username)

	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}

func TestUserPostgresRepository_UpdateUser(t *testing.T) {
	mockDB := database.NewMockDatabase(t)
	repo := NewUserPostgresRepository(mockDB)
	ctx := context.Background()

	userID := uuid.New()
	user := &database.User{
		Base: database.Base{
			ID: userID,
		},
		Username:  "updateduser",
		FirstName: "Updated",
		LastName:  "User",
		Email:     "updated@example.com",
	}

	mockDB.Mock.ExpectBegin()
	mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockDB.Mock.ExpectCommit()

	err := repo.UpdateUser(ctx, user)

	assert.NoError(t, err)
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}
