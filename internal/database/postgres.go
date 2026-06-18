package database

import (
	"fmt"
	"log"
	"lorem-backend/internal/config"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type postgresDatabase struct {
	db *gorm.DB
}

func NewPostgresDb(conf *config.Config) Database {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%v sslmode=disable TimeZone=Asia/Bangkok",
		conf.DBHost, conf.DBUser, conf.DBPassword, conf.DBName, conf.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(
		&User{},
		&Cart{},
		&CartItem{},
		&Product{},
		&Category{},
		&Order{},
		&OrderItem{},
		&Payment{},
		&File{},
	); err != nil {
		panic("Failed to migrate database")
	}

	sqlDB, _ := db.DB()

	// Ping to the database to verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(conf.DBMaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(conf.DBMaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Duration(conf.DBConnMaxLifetimeMin) * time.Minute)

	// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	sqlDB.SetConnMaxIdleTime(time.Duration(conf.DBConnMaxIdleTimeMin) * time.Minute)

	log.Println("Database Connected Successfully")
	return &postgresDatabase{
		db: db,
	}
}

func (p *postgresDatabase) DisconnectDB() {
	db := p.GetDb()

	if db == nil {
		return
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		panic(fmt.Sprintf("Error closing database connection: %v", err))
	}

	log.Println("Database Disconnected Successfully")
}

func (p *postgresDatabase) GetDb() *gorm.DB {
	return p.db
}
