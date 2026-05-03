package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MockDatabase struct {
	Db   *gorm.DB
	Mock sqlmock.Sqlmock
}

func NewMockDatabase(t *testing.T) *MockDatabase {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening gorm database", err)
	}

	return &MockDatabase{
		Db:   db,
		Mock: mock,
	}
}

func (m *MockDatabase) GetDb() *gorm.DB {
	return m.Db
}

func (m *MockDatabase) DisconnectDB() {
	// Not really needed for mock
}
