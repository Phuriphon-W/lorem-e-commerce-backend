package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Port                int    `mapstructure:"PORT"`
	DBHost              string `mapstructure:"DB_HOST"`
	DBUser              string `mapstructure:"DB_USER"`
	DBPassword          string `mapstructure:"DB_PASSWORD"`
	DBName              string `mapstructure:"DB_NAME"`
	DBPort              int    `mapstructure:"DB_PORT"`
	JWTSecret           string `mapstructure:"JWT_SECRET"`
	JWTExpire           string `mapstructure:"JWT_EXPIRE"`
	FrontendURL         string `mapstructure:"FRONTEND_URL"`
	S3Endpoint          string `mapstructure:"S3_URL"`
	BucketName          string `mapstructure:"BUCKET_NAME"`
	AwsAccessKey        string `mapstructure:"AWS_ACCESS_KEY"`
	AwsSecretKey        string `mapstructure:"AWS_SECRET_KEY"`
	AwsRegion           string `mapstructure:"AWS_REGION"`
	StripeSecretKey     string `mapstructure:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
}

var GlobalConfig *Config

func LoadConfig() {
	// Setup Viper to read the .env file
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // Read from system environment variables if they exist

	// Set Default Values (Used if not found in .env)
	viper.SetDefault("PORT", 5000)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_USER", "admin")
	viper.SetDefault("DB_PASSWORD", "password")
	viper.SetDefault("DB_NAME", "lorem")
	viper.SetDefault("DB_PORT", 5433)
	viper.SetDefault("JWT_EXPIRE", "24h")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("AWS_REGION", "us-east-1")

	// Read the file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: No .env file found, using defaults and system env")
	}

	// Unmarshal the config into our struct
	err := viper.Unmarshal(&GlobalConfig)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	log.Println("Config loaded successfully")
}
