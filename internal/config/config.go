package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        int
	DBHost      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBPort      int
	JWTSecret   string
	JWTExpire   string
	FrontendURL string
	S3Endpoint  string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	portStr := getEnv("PORT", "5000")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("Invalid PORT value in .env file")
	}

	dbPortStr := getEnv("DB_PORT", "5432")
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatal("Invalid DB_PORT value in .env file")
	}

	config := &Config{
		Port:        port,
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBUser:      getEnv("DB_USER", "admin"),
		DBPassword:  getEnv("DB_PASSWORD", "password"),
		DBName:      getEnv("DB_NAME", "lorem"),
		DBPort:      dbPort,
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTExpire:   getEnv("JWT_EXPIRE", "24h"),
		FrontendURL: getEnv("FRONTEND_URL", "localhost:3000"),
		S3Endpoint:  getEnv("S3_URL", "http://localhost:8333"),
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
