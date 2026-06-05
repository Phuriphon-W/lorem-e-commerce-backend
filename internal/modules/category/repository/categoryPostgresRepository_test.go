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

type CategoryRepositoryTestSuite struct {
	suite.Suite
	mockDB       *database.MockDatabase
	categoryRepo CategoryRepository
	ctx          context.Context
}

func (s *CategoryRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.categoryRepo = NewCategoryPostgresRepository(s.mockDB)
	s.ctx = context.Background()
}

func (s *CategoryRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

// ────────────────────────────────────────────────────────────
// TestCreateCategory
// ────────────────────────────────────────────────────────────

func (s *CategoryRepositoryTestSuite) TestCreateCategory() {
	catID := uuid.New()
	category := &database.Category{
		Name: "Apparel",
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - inserts category",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "categories"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(catID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - insert error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "categories"`)).
					WillReturnError(errors.New("db error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			id, err := s.categoryRepo.CreateCategory(s.ctx, category)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
				s.Equal(uuid.Nil, id)
			} else {
				s.Require().NoError(err)
				s.Equal(catID, id)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetCategoryByID
// ────────────────────────────────────────────────────────────

func (s *CategoryRepositoryTestSuite) TestGetCategoryByID() {
	catID := uuid.New()

	testCases := []struct {
		name    string
		id      uuid.UUID
		setup   func()
		wantErr error
	}{
		{
			name: "Success - returns category",
			id:   catID,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(catID, "Apparel")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories" WHERE id = \$1`).
					WithArgs(catID, 1).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Failure - record not found",
			id:   catID,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories" WHERE id = \$1`).
					WithArgs(catID, 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			res, err := s.categoryRepo.GetCategoryByID(s.ctx, tc.id)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Nil(res)
			} else {
				s.Require().NoError(err)
				s.NotNil(res)
				s.Equal(catID, res.ID)
				s.Equal("Apparel", res.Name)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestGetCategories
// ────────────────────────────────────────────────────────────

func (s *CategoryRepositoryTestSuite) TestGetCategories() {
	cat1ID := uuid.New()
	cat2ID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
		verify  func([]database.Category)
	}{
		{
			name: "Success - returns all categories",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(cat1ID, "Apparel").
					AddRow(cat2ID, "Electronics")
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories"`).WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(cats []database.Category) {
				s.Len(cats, 2)
				s.Equal(cat1ID, cats[0].ID)
				s.Equal("Apparel", cats[0].Name)
				s.Equal(cat2ID, cats[1].ID)
				s.Equal("Electronics", cats[1].Name)
			},
		},
		{
			name: "Success - returns empty list",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name"})
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories"`).WillReturnRows(rows)
			},
			wantErr: nil,
			verify: func(cats []database.Category) {
				s.Len(cats, 0)
			},
		},
		{
			name: "Failure - database error",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`SELECT \* FROM "categories"`).
					WillReturnError(errors.New("db query failed"))
			},
			wantErr: errors.New("db query failed"),
			verify: func(cats []database.Category) {
				s.Nil(cats)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			cats, err := s.categoryRepo.GetCategories(s.ctx)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
			tc.verify(cats)
		})
	}
}

// ────────────────────────────────────────────────────────────
// TestUpdateCategoryByID
// ────────────────────────────────────────────────────────────

func (s *CategoryRepositoryTestSuite) TestUpdateCategoryByID() {
	catID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - updates category name",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - DB execute update error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories"`)).
					WillReturnError(errors.New("update statement error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("update statement error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.categoryRepo.UpdateCategoryByID(s.ctx, catID, "Updated Apparel")

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
// TestDeleteCategoryByID
// ────────────────────────────────────────────────────────────

func (s *CategoryRepositoryTestSuite) TestDeleteCategoryByID() {
	catID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr error
	}{
		{
			name: "Success - soft deletes category",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories" SET "deleted_at"=`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name: "Failure - database delete error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectExec(regexp.QuoteMeta(`UPDATE "categories" SET "deleted_at"=`)).
					WillReturnError(errors.New("delete error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: errors.New("delete error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			err := s.categoryRepo.DeleteCategoryByID(s.ctx, catID)

			if tc.wantErr != nil {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErr.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestCategoryRepository(t *testing.T) {
	suite.Run(t, new(CategoryRepositoryTestSuite))
}
