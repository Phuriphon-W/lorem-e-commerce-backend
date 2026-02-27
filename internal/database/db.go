package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB(DBHost, DBUser, DBPassword, DBName string, DBPort int) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%v sslmode=disable TimeZone=Asia/Bangkok",
		DBHost, DBUser, DBPassword, DBName, DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	sqlDB, _ := db.DB()

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(20)

	log.Println("Database Connected Successfully")
	return db
}

func DisconnectDB(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		log.Fatal("Error closing database connection:", err)
	}

	log.Println("Database Disconnected Successfully")
}
